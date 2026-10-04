import { describe, it, expect, vi, afterEach } from 'vitest'
import {
  extractEmail, buildReplyAllCc, isPreviewable,
  escHtml, plainToHtml, buildForwardHtml, emlFilename, fetchSourceBase64,
  composeFocusTarget,
} from '../utils/mail.js'

// --- isPreviewable -----------------------------------------------------------

describe('isPreviewable', () => {
  it('returns true for image/jpeg', () => {
    expect(isPreviewable({ content_type: 'image/jpeg' })).toBe(true)
  })

  it('returns true for image/png', () => {
    expect(isPreviewable({ content_type: 'image/png' })).toBe(true)
  })

  it('returns true for image/gif', () => {
    expect(isPreviewable({ content_type: 'image/gif' })).toBe(true)
  })

  it('returns true for image/webp', () => {
    expect(isPreviewable({ content_type: 'image/webp' })).toBe(true)
  })

  it('returns true for application/pdf', () => {
    expect(isPreviewable({ content_type: 'application/pdf' })).toBe(true)
  })

  it('returns false for application/zip', () => {
    expect(isPreviewable({ content_type: 'application/zip' })).toBe(false)
  })

  it('returns false for text/plain', () => {
    expect(isPreviewable({ content_type: 'text/plain' })).toBe(false)
  })

  it('returns false for application/octet-stream', () => {
    expect(isPreviewable({ content_type: 'application/octet-stream' })).toBe(false)
  })

  it('returns false when content_type is missing', () => {
    expect(isPreviewable({})).toBe(false)
  })

  it('returns false for null', () => {
    expect(isPreviewable(null)).toBe(false)
  })
})

// --- extractEmail ------------------------------------------------------------

describe('extractEmail', () => {
  it('extracts email from "Name <email>" form', () => {
    expect(extractEmail('Alice <alice@example.com>')).toBe('alice@example.com')
  })

  it('returns plain email unchanged', () => {
    expect(extractEmail('bob@example.com')).toBe('bob@example.com')
  })

  it('trims whitespace', () => {
    expect(extractEmail('  carol@example.com  ')).toBe('carol@example.com')
  })

  it('handles angle brackets with inner whitespace', () => {
    expect(extractEmail('Dave < dave@example.com >')).toBe('dave@example.com')
  })

  it('returns empty string for null', () => {
    expect(extractEmail(null)).toBe('')
  })

  it('returns empty string for undefined', () => {
    expect(extractEmail(undefined)).toBe('')
  })

  it('returns empty string for empty string', () => {
    expect(extractEmail('')).toBe('')
  })
})

// --- buildReplyAllCc ---------------------------------------------------------

describe('buildReplyAllCc', () => {
  const own = ['me@example.com', 'alias@example.com']

  it('includes original To recipients in CC', () => {
    const cc = buildReplyAllCc(
      ['alice@example.com'],
      [],
      'sender@example.com',
      own,
    )
    expect(cc).toBe('alice@example.com')
  })

  it('includes original CC recipients', () => {
    const cc = buildReplyAllCc(
      [],
      ['cc@example.com'],
      'sender@example.com',
      own,
    )
    expect(cc).toBe('cc@example.com')
  })

  it('excludes the replyTo address from CC', () => {
    const cc = buildReplyAllCc(
      ['sender@example.com', 'alice@example.com'],
      [],
      'sender@example.com',
      own,
    )
    expect(cc).toBe('alice@example.com')
  })

  it('excludes all own addresses', () => {
    const cc = buildReplyAllCc(
      ['me@example.com', 'alice@example.com'],
      ['alias@example.com'],
      'sender@example.com',
      own,
    )
    expect(cc).toBe('alice@example.com')
  })

  it('is case-insensitive when excluding own addresses', () => {
    const cc = buildReplyAllCc(
      ['ME@EXAMPLE.COM', 'alice@example.com'],
      [],
      'sender@example.com',
      own,
    )
    expect(cc).toBe('alice@example.com')
  })

  it('is case-insensitive when excluding replyTo', () => {
    const cc = buildReplyAllCc(
      ['SENDER@EXAMPLE.COM', 'alice@example.com'],
      [],
      'sender@example.com',
      own,
    )
    expect(cc).toBe('alice@example.com')
  })

  it('handles "Name <email>" format in To/CC', () => {
    const cc = buildReplyAllCc(
      ['Alice <alice@example.com>'],
      ['Bob <bob@example.com>'],
      'sender@example.com',
      own,
    )
    expect(cc).toBe('Alice <alice@example.com>, Bob <bob@example.com>')
  })

  it('returns empty string when all recipients are excluded', () => {
    const cc = buildReplyAllCc(
      ['me@example.com'],
      ['alias@example.com'],
      'sender@example.com',
      own,
    )
    expect(cc).toBe('')
  })

  it('handles null/undefined To and CC gracefully', () => {
    const cc = buildReplyAllCc(null, null, 'sender@example.com', own)
    expect(cc).toBe('')
  })

  it('excludes the replyTo "Name <email>" form correctly', () => {
    const cc = buildReplyAllCc(
      ['Sender <sender@example.com>', 'alice@example.com'],
      [],
      'Sender <sender@example.com>',
      own,
    )
    expect(cc).toBe('alice@example.com')
  })
})

// --- escHtml / plainToHtml ---------------------------------------------------

describe('escHtml', () => {
  it('escapes &, < and >', () => {
    expect(escHtml('a & <b>')).toBe('a &amp; &lt;b&gt;')
  })
})

describe('plainToHtml', () => {
  it('returns empty string for empty input', () => {
    expect(plainToHtml('')).toBe('')
  })

  it('wraps each line in <p> and keeps blank lines', () => {
    expect(plainToHtml('hi\n\n<x>')).toBe('<p>hi</p><p><br></p><p>&lt;x&gt;</p>')
  })
})

// --- buildForwardHtml --------------------------------------------------------

describe('buildForwardHtml', () => {
  const msg = {
    from: 'Alice <alice@example.com>',
    to: ['bob@example.com', 'carol@example.com'],
    subject: 'Hello & bye',
    html_body: '<p>body</p>',
  }

  it('includes escaped header fields and the HTML body', () => {
    const html = buildForwardHtml(msg, 'Jan 1')
    expect(html.startsWith('<blockquote>')).toBe(true)
    expect(html).toContain('--- Forwarded message ---')
    expect(html).toContain('Alice &lt;alice@example.com&gt;')
    expect(html).toContain('<strong>Date:</strong> Jan 1')
    expect(html).toContain('Hello &amp; bye')
    expect(html).toContain('bob@example.com, carol@example.com')
    expect(html).toContain('<p>body</p></blockquote>')
  })

  it('omits empty date and To lines', () => {
    const html = buildForwardHtml({ ...msg, to: [] }, '')
    expect(html).not.toContain('Date:')
    expect(html).not.toContain('To:')
  })

  it('falls back to the plain-text body', () => {
    const html = buildForwardHtml({ ...msg, html_body: '', text_body: 'plain' }, '')
    expect(html).toContain('<p>plain</p>')
  })
})

// --- emlFilename -------------------------------------------------------------

describe('emlFilename', () => {
  it('sanitises unsafe filename characters', () => {
    expect(emlFilename('Re: a/b?')).toBe('Re_ a_b_.eml')
  })

  it('defaults to message.eml when the subject is empty', () => {
    expect(emlFilename('')).toBe('message.eml')
  })
})

// --- fetchSourceBase64 -------------------------------------------------------

describe('fetchSourceBase64', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('fetches the source and base64-encodes it', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      arrayBuffer: async () => new TextEncoder().encode('hi').buffer,
    })
    vi.stubGlobal('fetch', fetchMock)
    expect(await fetchSourceBase64('My Folder', 7)).toBe(btoa('hi'))
    expect(fetchMock).toHaveBeenCalledWith('/api/folders/My%20Folder/messages/7/source')
  })

  it('returns null on a failed response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))
    expect(await fetchSourceBase64('INBOX', 1)).toBeNull()
  })
})

// --- composeFocusTarget ------------------------------------------------------

describe('composeFocusTarget', () => {
  it('focuses To when no recipient is set', () => {
    expect(composeFocusTarget('')).toBe('to')
    expect(composeFocusTarget('   ')).toBe('to')
    expect(composeFocusTarget(undefined)).toBe('to')
  })

  it('focuses the body when To is prefilled (reply)', () => {
    expect(composeFocusTarget('alice@example.com')).toBe('body')
  })
})
