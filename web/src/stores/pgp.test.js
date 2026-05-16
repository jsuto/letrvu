// @vitest-environment node
import { describe, it, expect, beforeAll } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import * as openpgp from 'openpgp'
import { usePGPStore } from './pgp'

// Shared key pair generated once for all tests.
let testPrivateKey
let testPublicKey

beforeAll(async () => {
  const result = await openpgp.generateKey({
    type: 'ecc',
    curve: 'curve25519',
    userIDs: [{ name: 'Test User', email: 'test@example.com' }],
    passphrase: 'testpass',
    format: 'armored',
  })
  const encKey = await openpgp.readPrivateKey({ armoredKey: result.privateKey })
  testPrivateKey = await openpgp.decryptKey({ privateKey: encKey, passphrase: 'testpass' })
  testPublicKey = testPrivateKey.toPublic()
})

function makePGPStore() {
  setActivePinia(createPinia())
  const pgp = usePGPStore()
  pgp.privateKey = testPrivateKey
  pgp.publicKey = testPublicKey
  return pgp
}

// ── signMIME canonical body part ─────────────────────────────────────────────

describe('signMIME canonical body part', () => {
  it('returns text, signature, and micalg fields', async () => {
    const pgp = makePGPStore()
    const result = await pgp.signMIME('hello world')
    expect(result).toHaveProperty('text', 'hello world')
    expect(result).toHaveProperty('signature')
    expect(result).toHaveProperty('micalg', 'pgp-sha512')
    expect(result.signature).toMatch(/-----BEGIN PGP SIGNATURE-----/)
    expect(result.signature).toMatch(/-----END PGP SIGNATURE-----/)
  })

  it('normalises bare LF to CRLF in the signed body part', async () => {
    const pgp = makePGPStore()
    const { signature } = await pgp.signMIME('line1\nline2')
    // Re-construct the canonical body part as the verifier would, and check it
    // contains CRLF line endings, not bare LF.
    const crlf = 'line1\r\nline2'
    const bodyPart =
      'Content-Type: text/plain; charset=UTF-8\r\n' +
      'Content-Transfer-Encoding: 8bit\r\n' +
      '\r\n' +
      crlf
    // The body part we reconstruct should start with the exact headers.
    expect(bodyPart).toContain('Content-Type: text/plain; charset=UTF-8\r\n')
    expect(bodyPart).toContain('Content-Transfer-Encoding: 8bit\r\n')
    // Verify the signature against the same canonical bytes.
    const binary = new TextEncoder().encode(bodyPart)
    const message = await openpgp.createMessage({ binary })
    const { signatures } = await openpgp.verify({
      message,
      signature: await openpgp.readSignature({ armoredSignature: signature }),
      verificationKeys: testPublicKey,
    })
    expect(await signatures[0].verified).toBe(true)
  })

  it('normalises already-CRLF text without double-conversion', async () => {
    const pgp = makePGPStore()
    const result = await pgp.signMIME('line1\r\nline2')
    // Should not throw and should produce a valid signature.
    expect(result.signature).toMatch(/-----BEGIN PGP SIGNATURE-----/)
  })
})

// ── signMIME roundtrip verification ──────────────────────────────────────────

describe('signMIME roundtrip', () => {
  it('signature verifies against reconstructed canonical body part', async () => {
    const pgp = makePGPStore()
    const text = 'Hello, PGP/MIME!'
    const { signature } = await pgp.signMIME(text)

    // Reconstruct the exact canonical body part (matches signMIME internals
    // and writePGPMIMESigned in sender.go).
    const crlf = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n').replace(/\n/g, '\r\n')
    const bodyPart =
      'Content-Type: text/plain; charset=UTF-8\r\n' +
      'Content-Transfer-Encoding: 8bit\r\n' +
      '\r\n' +
      crlf
    const binary = new TextEncoder().encode(bodyPart)
    const message = await openpgp.createMessage({ binary })
    const { signatures } = await openpgp.verify({
      message,
      signature: await openpgp.readSignature({ armoredSignature: signature }),
      verificationKeys: testPublicKey,
    })
    expect(await signatures[0].verified).toBe(true)
  })

  it('signature does NOT verify when body is tampered', async () => {
    const pgp = makePGPStore()
    const { signature } = await pgp.signMIME('original body')

    // Tamper: use different text.
    const tamperedText = 'tampered body'
    const crlf = tamperedText.replace(/\r\n/g, '\n').replace(/\n/g, '\r\n')
    const tamperedBodyPart =
      'Content-Type: text/plain; charset=UTF-8\r\n' +
      'Content-Transfer-Encoding: 8bit\r\n' +
      '\r\n' +
      crlf
    const binary = new TextEncoder().encode(tamperedBodyPart)
    const message = await openpgp.createMessage({ binary })
    const { signatures } = await openpgp.verify({
      message,
      signature: await openpgp.readSignature({ armoredSignature: signature }),
      verificationKeys: testPublicKey,
    })
    let valid = false
    try { valid = await signatures[0].verified } catch { valid = false }
    expect(valid).toBe(false)
  })

  it('signature does NOT verify when headers are changed', async () => {
    const pgp = makePGPStore()
    const { signature } = await pgp.signMIME('hello')

    // Tamper: omit Content-Transfer-Encoding header from the body part.
    const tamperedBodyPart =
      'Content-Type: text/plain; charset=UTF-8\r\n' +
      '\r\n' +
      'hello'
    const binary = new TextEncoder().encode(tamperedBodyPart)
    const message = await openpgp.createMessage({ binary })
    const { signatures } = await openpgp.verify({
      message,
      signature: await openpgp.readSignature({ armoredSignature: signature }),
      verificationKeys: testPublicKey,
    })
    let valid = false
    try { valid = await signatures[0].verified } catch { valid = false }
    expect(valid).toBe(false)
  })
})

// ── signMIME requires unlocked key ───────────────────────────────────────────

describe('signMIME key guard', () => {
  it('throws when private key is not set', async () => {
    setActivePinia(createPinia())
    const pgp = usePGPStore()
    // privateKey is null by default.
    await expect(pgp.signMIME('test')).rejects.toThrow('Key not unlocked')
  })
})
