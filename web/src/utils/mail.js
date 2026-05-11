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
