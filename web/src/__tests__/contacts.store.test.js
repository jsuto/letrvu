import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useContactsStore } from '../stores/contacts.js'

beforeEach(() => {
  setActivePinia(createPinia())
  global.fetch = vi.fn()
})

function mockFetch(data, ok = true) {
  global.fetch = vi.fn().mockResolvedValue({
    ok,
    json: () => Promise.resolve(data),
  })
}

function mockApiFetch(data, ok = true) {
  // apiFetch wraps fetch; mock at the fetch level for simplicity
  mockFetch(data, ok)
}

// --- Initial state -----------------------------------------------------------

describe('contacts store — initial state', () => {
  it('starts with empty contacts and groups', () => {
    const store = useContactsStore()
    expect(store.contacts).toEqual([])
    expect(store.groups).toEqual([])
  })

  it('loading is false initially', () => {
    const store = useContactsStore()
    expect(store.loading).toBe(false)
  })
})

// --- fetchContacts -----------------------------------------------------------

describe('contacts store — fetchContacts', () => {
  it('loads contacts and groups in parallel', async () => {
    const contactsData = [{ id: 1, name: 'Alice', emails: [{ email: 'a@example.com' }] }]
    const groupsData = [{ id: 1, name: 'Team', members: [] }]
    global.fetch = vi.fn()
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(contactsData) })
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(groupsData) })

    const store = useContactsStore()
    await store.fetchContacts()

    expect(store.contacts).toEqual(contactsData)
    expect(store.groups).toEqual(groupsData)
  })

  it('sets loading false after fetch', async () => {
    global.fetch = vi.fn()
      .mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    const store = useContactsStore()
    await store.fetchContacts()
    expect(store.loading).toBe(false)
  })
})

// --- createContact -----------------------------------------------------------

describe('contacts store — createContact', () => {
  it('adds contact to list and sorts alphabetically', async () => {
    const store = useContactsStore()
    store.contacts = [{ id: 1, name: 'Zara', emails: [] }]

    mockApiFetch({ id: 2, name: 'Alice', emails: [] })
    await store.createContact({ name: 'Alice', notes: '', emails: [] })

    expect(store.contacts[0].name).toBe('Alice')
    expect(store.contacts[1].name).toBe('Zara')
  })

  it('returns the created contact', async () => {
    const store = useContactsStore()
    const newContact = { id: 5, name: 'Bob', emails: [] }
    mockApiFetch(newContact)

    const result = await store.createContact({ name: 'Bob', notes: '', emails: [] })
    expect(result).toEqual(newContact)
  })
})

// --- updateContact -----------------------------------------------------------

describe('contacts store — updateContact', () => {
  it('replaces the contact in the list', async () => {
    const store = useContactsStore()
    store.contacts = [{ id: 1, name: 'Old Name', emails: [] }]

    const updated = { id: 1, name: 'New Name', emails: [] }
    mockApiFetch(updated)
    await store.updateContact(1, { name: 'New Name', notes: '', emails: [] })

    expect(store.contacts[0].name).toBe('New Name')
  })

  it('keeps list sorted after update', async () => {
    const store = useContactsStore()
    store.contacts = [
      { id: 1, name: 'Alice', emails: [] },
      { id: 2, name: 'Charlie', emails: [] },
    ]
    mockApiFetch({ id: 2, name: 'Aaron', emails: [] })
    await store.updateContact(2, { name: 'Aaron', notes: '', emails: [] })

    expect(store.contacts[0].name).toBe('Aaron')
    expect(store.contacts[1].name).toBe('Alice')
  })
})

// --- deleteContact -----------------------------------------------------------

describe('contacts store — deleteContact', () => {
  it('removes the contact from the list', async () => {
    const store = useContactsStore()
    store.contacts = [
      { id: 1, name: 'Alice', emails: [] },
      { id: 2, name: 'Bob', emails: [] },
    ]
    mockApiFetch({ status: 'ok' })
    await store.deleteContact(1)

    expect(store.contacts).toHaveLength(1)
    expect(store.contacts[0].id).toBe(2)
  })
})

// --- createGroup -------------------------------------------------------------

describe('contacts store — createGroup', () => {
  it('adds group to list and sorts alphabetically', async () => {
    const store = useContactsStore()
    store.groups = [{ id: 1, name: 'Zeta', members: [] }]

    mockApiFetch({ id: 2, name: 'Alpha', members: [] })
    await store.createGroup('Alpha')

    expect(store.groups[0].name).toBe('Alpha')
    expect(store.groups[1].name).toBe('Zeta')
  })

  it('returns the created group', async () => {
    const store = useContactsStore()
    const newGroup = { id: 3, name: 'Sales', members: [] }
    mockApiFetch(newGroup)

    const result = await store.createGroup('Sales')
    expect(result).toEqual(newGroup)
  })

  it('throws when response is not ok', async () => {
    const store = useContactsStore()
    mockApiFetch({}, false)
    await expect(store.createGroup('Fail')).rejects.toThrow()
  })
})

// --- updateGroup -------------------------------------------------------------

describe('contacts store — updateGroup', () => {
  it('replaces the group in the list', async () => {
    const store = useContactsStore()
    store.groups = [{ id: 1, name: 'Old', members: [] }]

    mockApiFetch({ id: 1, name: 'New Name', members: [] })
    await store.updateGroup(1, 'New Name')

    expect(store.groups[0].name).toBe('New Name')
  })

  it('keeps groups sorted after rename', async () => {
    const store = useContactsStore()
    store.groups = [
      { id: 1, name: 'Alpha', members: [] },
      { id: 2, name: 'Gamma', members: [] },
    ]
    mockApiFetch({ id: 2, name: 'Beta', members: [] })
    await store.updateGroup(2, 'Beta')

    expect(store.groups[0].name).toBe('Alpha')
    expect(store.groups[1].name).toBe('Beta')
  })
})

// --- deleteGroup -------------------------------------------------------------

describe('contacts store — deleteGroup', () => {
  it('removes the group from the list', async () => {
    const store = useContactsStore()
    store.groups = [
      { id: 1, name: 'Alpha', members: [] },
      { id: 2, name: 'Beta', members: [] },
    ]
    mockApiFetch({ status: 'ok' })
    await store.deleteGroup(1)

    expect(store.groups).toHaveLength(1)
    expect(store.groups[0].id).toBe(2)
  })

  it('throws when response is not ok', async () => {
    const store = useContactsStore()
    mockApiFetch({}, false)
    await expect(store.deleteGroup(99)).rejects.toThrow()
  })
})

// --- addGroupMember ----------------------------------------------------------

describe('contacts store — addGroupMember', () => {
  it('updates the group in the list with the returned data', async () => {
    const store = useContactsStore()
    store.groups = [{ id: 1, name: 'Team', members: [] }]

    const updated = { id: 1, name: 'Team', members: [{ contact_id: 5, name: 'Alice', email: 'a@example.com' }] }
    mockApiFetch(updated)
    await store.addGroupMember(1, 5)

    expect(store.groups[0].members).toHaveLength(1)
    expect(store.groups[0].members[0].contact_id).toBe(5)
  })

  it('throws when response is not ok', async () => {
    const store = useContactsStore()
    mockApiFetch({}, false)
    await expect(store.addGroupMember(1, 5)).rejects.toThrow()
  })
})

// --- removeGroupMember -------------------------------------------------------

describe('contacts store — removeGroupMember', () => {
  it('updates the group in the list with the returned data', async () => {
    const store = useContactsStore()
    store.groups = [{
      id: 1,
      name: 'Team',
      members: [{ contact_id: 5, name: 'Alice', email: 'a@example.com' }],
    }]

    const updated = { id: 1, name: 'Team', members: [] }
    mockApiFetch(updated)
    await store.removeGroupMember(1, 5)

    expect(store.groups[0].members).toHaveLength(0)
  })
})

// --- autocomplete ------------------------------------------------------------

describe('contacts store — autocomplete', () => {
  it('returns empty array for empty query', async () => {
    const store = useContactsStore()
    const result = await store.autocomplete('')
    expect(result).toEqual([])
    expect(global.fetch).not.toHaveBeenCalled()
  })

  it('calls the autocomplete endpoint with encoded query', async () => {
    const store = useContactsStore()
    mockFetch([{ type: 'contact', contact_id: 1, name: 'Alice', email: 'a@example.com' }])

    const results = await store.autocomplete('ali')
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/contacts/autocomplete?q=ali'),
    )
    expect(results).toHaveLength(1)
  })

  it('returns empty array on fetch error', async () => {
    const store = useContactsStore()
    global.fetch = vi.fn().mockResolvedValue({ ok: false })
    const result = await store.autocomplete('test')
    expect(result).toEqual([])
  })
})
