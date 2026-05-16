import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as openpgp from 'openpgp'
import { apiFetch } from '../api'

export const usePGPStore = defineStore('pgp', () => {
  // Passphrase-protected armored private key stored on server
  const encryptedKey = ref(null)   // string | null
  // Unlocked private key (in memory only — never persisted)
  const privateKey = ref(null)     // openpgp.PrivateKey | null
  const publicKey = ref(null)      // openpgp.PublicKey | null

  const hasKey    = computed(() => encryptedKey.value !== null)
  const isUnlocked = computed(() => privateKey.value !== null)
  const isLocked  = computed(() => hasKey.value && !isUnlocked.value)

  const fingerprint = computed(() => {
    if (!privateKey.value) return null
    const hex = privateKey.value.getFingerprint().toUpperCase()
    // Format as groups of 4: "ABCD 1234 …"
    return hex.match(/.{4}/g)?.join(' ') ?? hex
  })

  const keyId = computed(() => {
    if (!privateKey.value) return null
    return privateKey.value.getKeyID().toHex().toUpperCase()
  })

  const userId = computed(() => {
    if (!privateKey.value) return null
    return privateKey.value.getUserIDs()[0] ?? null
  })

  // ── Key lifecycle ─────────────────────────────────────────────────────────

  async function fetchKey() {
    try {
      const res = await apiFetch('/api/pgp/key')
      if (res.status === 404) { encryptedKey.value = null; return }
      if (!res.ok) return
      const data = await res.json()
      encryptedKey.value = data.key
    } catch { /* ignore network errors */ }
  }

  // Unlock in-memory: decrypt the stored armored key with the passphrase.
  // Throws if the passphrase is wrong.
  async function unlock(passphrase) {
    if (!encryptedKey.value) throw new Error('No key stored')
    const encKey = await openpgp.readPrivateKey({ armoredKey: encryptedKey.value })
    const decKey = await openpgp.decryptKey({ privateKey: encKey, passphrase })
    privateKey.value = decKey
    publicKey.value = decKey.toPublic()
  }

  function lock() {
    privateKey.value = null
    publicKey.value = null
  }

  // Generate a new ECC key pair, protect it with passphrase, save to server,
  // and unlock in memory. Returns the armored public key for display/export.
  async function generateKey(name, email, passphrase) {
    const result = await openpgp.generateKey({
      type: 'ecc',
      curve: 'curve25519',
      userIDs: [{ name, email }],
      passphrase,
      format: 'armored',
    })
    await _saveToServer(result.privateKey)
    encryptedKey.value = result.privateKey
    await unlock(passphrase)
    return result.publicKey
  }

  // Import an existing armored private key (must already be passphrase-protected).
  // Validates the passphrase before saving.
  async function importKey(armoredKey, passphrase) {
    const encKey = await openpgp.readPrivateKey({ armoredKey })
    // Validate passphrase — throws if wrong
    await openpgp.decryptKey({ privateKey: encKey, passphrase })
    await _saveToServer(armoredKey)
    encryptedKey.value = armoredKey
    await unlock(passphrase)
  }

  async function deleteKey() {
    await apiFetch('/api/pgp/key', { method: 'DELETE' })
    encryptedKey.value = null
    privateKey.value = null
    publicKey.value = null
  }

  async function _saveToServer(armoredEncryptedKey) {
    const res = await apiFetch('/api/pgp/key', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ key: armoredEncryptedKey }),
    })
    if (!res.ok) throw new Error('Failed to save key to server')
  }

  // Returns the armored public key string, or null if locked/no key.
  function armoredPublicKey() {
    return publicKey.value?.armor() ?? null
  }

  // ── Crypto operations ─────────────────────────────────────────────────────

  // Sign for PGP/MIME (RFC 3156). Constructs the canonical MIME body part
  // (matching exactly what the backend will emit as the first part of
  // multipart/signed), signs its raw bytes as a detached signature, and
  // returns { text, signature, micalg } for inclusion in the send payload.
  // The backend reconstructs the identical MIME part and uses the detached
  // signature as the second multipart/signed part.
  async function signMIME(text) {
    if (!privateKey.value) throw new Error('Key not unlocked')
    // Canonical CRLF — must match normalizeCRLF() in smtp/sender.go
    const crlf = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n').replace(/\n/g, '\r\n')
    // Byte-identical to what the backend writes as the first MIME part
    const bodyPart = 'Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n' + crlf
    const binary = new TextEncoder().encode(bodyPart)
    const signature = await openpgp.sign({
      message: await openpgp.createMessage({ binary }),
      signingKeys: privateKey.value,
      detached: true,
    })
    return { text, signature, micalg: 'pgp-sha512' }
  }

  // Sign plain text as inline cleartext (kept for compatibility).
  async function signText(text) {
    if (!privateKey.value) throw new Error('Key not unlocked')
    return openpgp.sign({
      message: await openpgp.createCleartextMessage({ text }),
      signingKeys: privateKey.value,
    })
  }

  // Encrypt (and optionally sign) text for an array of armored public keys.
  // Returns an armored PGP message.
  async function encryptText(text, recipientArmoredKeys, sign = true) {
    const encryptionKeys = await Promise.all(
      recipientArmoredKeys.map(a => openpgp.readKey({ armoredKey: a }))
    )
    const opts = {
      message: await openpgp.createMessage({ text }),
      encryptionKeys,
    }
    if (sign && privateKey.value) opts.signingKeys = privateKey.value
    return openpgp.encrypt(opts)
  }

  // Decrypt an armored PGP message. Returns { text, signatures }.
  async function decryptMessage(armoredMessage) {
    if (!privateKey.value) throw new Error('Key not unlocked')
    const message = await openpgp.readMessage({ armoredMessage })
    return openpgp.decrypt({ message, decryptionKeys: privateKey.value })
  }

  // Verify an armored cleartext signed message using the sender's armored public key.
  // Returns { text, valid, keyId }.
  async function verifyCleartext(armoredSigned, senderArmoredKey) {
    const message = await openpgp.readCleartextMessage({ cleartextMessage: armoredSigned })
    const verificationKeys = await openpgp.readKey({ armoredKey: senderArmoredKey })
    const { data, signatures } = await openpgp.verify({ message, verificationKeys })
    let valid = false
    try { valid = await signatures[0]?.verified } catch { valid = false }
    return { text: data, valid, keyId: signatures[0]?.keyID?.toHex()?.toUpperCase() }
  }

  // ── Contact public key helpers ────────────────────────────────────────────

  // Fetch the stored public key for a given email. Returns armored string or null.
  async function getKeyForEmail(email) {
    try {
      const res = await apiFetch(`/api/pgp/key-for-email?email=${encodeURIComponent(email)}`)
      if (!res.ok) return null
      const data = await res.json()
      return data.key
    } catch { return null }
  }

  // Fetch a public key for an email via WKD (proxied through the backend).
  // Returns armored string or null.
  async function wkdLookup(email) {
    try {
      const res = await apiFetch(`/api/pgp/wkd?email=${encodeURIComponent(email)}`)
      if (!res.ok) return null
      const data = await res.json()
      // Backend returns binary OpenPGP packets as base64; convert to armored.
      const binary = Uint8Array.from(atob(data.key_b64), c => c.charCodeAt(0))
      const key = await openpgp.readKey({ binaryKey: binary })
      return key.armor()
    } catch { return null }
  }

  return {
    encryptedKey, privateKey, publicKey,
    hasKey, isUnlocked, isLocked,
    fingerprint, keyId, userId,
    fetchKey, unlock, lock,
    generateKey, importKey, deleteKey,
    armoredPublicKey,
    signMIME, signText, encryptText, decryptMessage, verifyCleartext,
    getKeyForEmail, wkdLookup,
  }
})
