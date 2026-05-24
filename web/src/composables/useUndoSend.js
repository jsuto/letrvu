import { ref } from 'vue'

// Module-level singleton — one pending send at a time across the app.
const pending = ref(null)
// pending shape: { delay, startedAt, _resolve, _reject, _timeoutId }

export function useUndoSend() {
  /**
   * Schedule a send with a delay. Returns a Promise that:
   *   - resolves when the countdown expires (caller should send)
   *   - rejects with Error('undo') when the user cancels
   *
   * If a send is already pending it is flushed immediately before scheduling
   * the new one, so there is never more than one pending at a time.
   */
  function schedule(delay) {
    return new Promise((resolve, reject) => {
      if (pending.value) {
        clearTimeout(pending.value._timeoutId)
        pending.value._resolve()
      }
      const _timeoutId = setTimeout(() => {
        pending.value = null
        resolve()
      }, delay * 1000)
      pending.value = { delay, startedAt: Date.now(), _resolve: resolve, _reject: reject, _timeoutId }
    })
  }

  /** Cancel the pending send. Returns true if there was something to cancel. */
  function undo() {
    if (!pending.value) return false
    clearTimeout(pending.value._timeoutId)
    pending.value._reject(new Error('undo'))
    pending.value = null
    return true
  }

  /** Send immediately without waiting for the countdown to expire. */
  function flush() {
    if (!pending.value) return
    clearTimeout(pending.value._timeoutId)
    pending.value._resolve()
    pending.value = null
  }

  return { pending, schedule, undo, flush }
}
