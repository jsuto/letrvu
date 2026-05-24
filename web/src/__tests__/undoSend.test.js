import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { useUndoSend } from '../composables/useUndoSend'

// Reset the module-level singleton between tests by re-importing.
beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  // Cancel any leftover pending send to avoid polluting other tests.
  const { undo } = useUndoSend()
  undo()
  vi.useRealTimers()
})

describe('useUndoSend — schedule', () => {
  it('resolves after the delay', async () => {
    const { schedule } = useUndoSend()
    let resolved = false
    schedule(5).then(() => { resolved = true })
    expect(resolved).toBe(false)
    vi.advanceTimersByTime(5000)
    await Promise.resolve()
    expect(resolved).toBe(true)
  })

  it('rejects with "undo" when undo() is called', async () => {
    const { schedule, undo } = useUndoSend()
    let reason = null
    schedule(10).catch(e => { reason = e.message })
    undo()
    await Promise.resolve()
    expect(reason).toBe('undo')
  })

  it('undo() returns true when there is a pending send', () => {
    const { schedule, undo } = useUndoSend()
    schedule(10).catch(() => {})
    expect(undo()).toBe(true)
  })

  it('undo() returns false when there is nothing pending', () => {
    const { undo } = useUndoSend()
    expect(undo()).toBe(false)
  })

  it('flush() resolves the promise immediately', async () => {
    const { schedule, flush } = useUndoSend()
    let resolved = false
    schedule(10).then(() => { resolved = true })
    flush()
    await Promise.resolve()
    expect(resolved).toBe(true)
  })

  it('pending is null before scheduling', () => {
    const { pending } = useUndoSend()
    expect(pending.value).toBeNull()
  })

  it('pending is set after scheduling', () => {
    const { pending, schedule } = useUndoSend()
    schedule(5).catch(() => {})
    expect(pending.value).not.toBeNull()
    expect(pending.value.delay).toBe(5)
  })

  it('pending is null after undo', () => {
    const { pending, schedule, undo } = useUndoSend()
    schedule(5).catch(() => {})
    undo()
    expect(pending.value).toBeNull()
  })
})
