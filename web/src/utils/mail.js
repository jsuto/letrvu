/**
 * Extract the bare email address from an RFC 5322 address string.
 * Handles "Name <email>" and plain "email" forms.
 */
export function extractEmail(addr) {
  if (!addr) return ''
  const match = addr.match(/<([^>]+)>/)
  return match ? match[1].trim() : addr.trim()
}

/**
 * Returns true when the attachment can be previewed inline.
 * Images (image/*) and PDFs (application/pdf) are supported.
 */
export function isPreviewable(att) {
  const ct = att?.content_type ?? ''
  return ct.startsWith('image/') || ct === 'application/pdf'
}

export function escHtml(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

export function plainToHtml(text) {
  if (!text) return ''
  return text.split('\n').map(l => `<p>${escHtml(l) || '<br>'}</p>`).join('')
}

/**
 * Build the quoted HTML body for an inline forward.
 *
 * @param {object} msg  - full message (from, to, subject, html_body, text_body)
 * @param {string} date - already-formatted date string ('' to omit)
 * @returns {string}
 */
export function buildForwardHtml(msg, date) {
  const to = Array.isArray(msg.to) ? msg.to.join(', ') : (msg.to || '')
  const headerHtml = [
    '<p><strong>--- Forwarded message ---</strong></p>',
    `<p><strong>From:</strong> ${escHtml(msg.from || '')}</p>`,
    date ? `<p><strong>Date:</strong> ${escHtml(date)}</p>` : '',
    `<p><strong>Subject:</strong> ${escHtml(msg.subject || '')}</p>`,
    to ? `<p><strong>To:</strong> ${escHtml(to)}</p>` : '',
  ].filter(Boolean).join('')
  const bodyHtml = msg.html_body || plainToHtml(msg.text_body || '')
  return `<blockquote>${headerHtml}${bodyHtml}</blockquote>`
}

/**
 * Filename used when forwarding a message as an .eml attachment.
 */
export function emlFilename(subject) {
  return `${(subject || 'message').replace(/[/\\?%*:|"<>]/g, '_')}.eml`
}

/**
 * Fetch a message's raw source and return it base64-encoded, or null on error.
 */
export async function fetchSourceBase64(folder, uid) {
  const res = await fetch(`/api/folders/${encodeURIComponent(folder)}/messages/${uid}/source`)
  if (!res.ok) return null
  const uint8 = new Uint8Array(await res.arrayBuffer())
  let binary = ''
  for (const b of uint8) binary += String.fromCharCode(b)
  return btoa(binary)
}

/**
 * Build the CC list for a Reply All.
 *
 * Collects every address from the original To + CC, then removes:
 *   - the user's own addresses (all configured identities)
 *   - the address being replied to (already in the To field)
 *
 * Returns a comma-separated string suitable for the CC field.
 *
 * @param {string[]} originalTo   - original message To addresses
 * @param {string[]} originalCc   - original message CC addresses
 * @param {string}   replyToAddr  - address going into the To field
 * @param {string[]} ownEmails    - all of the user's own email addresses
 * @returns {string}
 */
export function buildReplyAllCc(originalTo, originalCc, replyToAddr, ownEmails) {
  const exclude = new Set([
    extractEmail(replyToAddr).toLowerCase(),
    ...ownEmails.map(e => e.toLowerCase()),
  ])
  return [...(originalTo ?? []), ...(originalCc ?? [])]
    .filter(addr => !exclude.has(extractEmail(addr).toLowerCase()))
    .join(', ')
}

/**
 * Which compose field should get the cursor when the window opens:
 * 'to' when no recipient is filled in yet (new message, forward),
 * otherwise 'body' (reply, reply all, editing a draft with recipients).
 */
export function composeFocusTarget(to) {
  return (to ?? '').trim() ? 'body' : 'to'
}
