import { describe, it, expect } from 'vitest'
import { extractEmail, buildReplyAllCc, isPreviewable } from '../utils/mail.js'

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
