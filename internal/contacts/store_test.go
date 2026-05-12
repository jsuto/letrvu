package contacts

import (
	"testing"

	"github.com/jsuto/letrvu/internal/db"
)

const (
	owner    = "alice"
	imapHost = "imap.example.com"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	stmts := []string{
		`CREATE TABLE contacts (
			id        INTEGER PRIMARY KEY,
			owner     TEXT NOT NULL,
			imap_host TEXT NOT NULL,
			name      TEXT NOT NULL DEFAULT '',
			notes     TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE contact_emails (
			id         INTEGER PRIMARY KEY,
			contact_id INTEGER NOT NULL,
			email      TEXT    NOT NULL,
			label      TEXT    NOT NULL DEFAULT '',
			UNIQUE (contact_id, email)
		)`,
		`CREATE TABLE contact_groups (
			id        INTEGER PRIMARY KEY,
			owner     TEXT NOT NULL,
			imap_host TEXT NOT NULL,
			name      TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE contact_group_members (
			group_id   INTEGER NOT NULL,
			contact_id INTEGER NOT NULL,
			PRIMARY KEY (group_id, contact_id)
		)`,
	}
	for _, s := range stmts {
		if _, err := database.Exec(s); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func newStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(openTestDB(t))
}

// --- Contact CRUD ------------------------------------------------------------

func TestContact_CreateAndGet(t *testing.T) {
	s := newStore(t)
	c, err := s.Create(owner, imapHost, "Alice", "notes", []ContactEmail{{Email: "alice@example.com", Label: "work"}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.Name != "Alice" {
		t.Errorf("Name = %q", c.Name)
	}
	if len(c.Emails) != 1 || c.Emails[0].Email != "alice@example.com" {
		t.Errorf("Emails = %v", c.Emails)
	}

	got, err := s.Get(c.ID, owner, imapHost)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil || got.ID != c.ID {
		t.Fatal("Get returned nil or wrong contact")
	}
}

func TestContact_GetNotFound(t *testing.T) {
	s := newStore(t)
	got, err := s.Get(999, owner, imapHost)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Error("expected nil for missing contact")
	}
}

func TestContact_Update(t *testing.T) {
	s := newStore(t)
	c, _ := s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})

	updated, err := s.Update(c.ID, owner, imapHost, "Alice Smith", "new notes", []ContactEmail{
		{Email: "alice@example.com", Label: "work"},
		{Email: "alice.home@example.com", Label: "home"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Alice Smith" {
		t.Errorf("Name = %q", updated.Name)
	}
	if updated.Notes != "new notes" {
		t.Errorf("Notes = %q", updated.Notes)
	}
	if len(updated.Emails) != 2 {
		t.Errorf("Emails len = %d", len(updated.Emails))
	}
}

func TestContact_Delete(t *testing.T) {
	s := newStore(t)
	c, _ := s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})
	if err := s.Delete(c.ID, owner, imapHost); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, _ := s.Get(c.ID, owner, imapHost)
	if got != nil {
		t.Error("contact should be gone after Delete")
	}
}

func TestContact_ListSortedByName(t *testing.T) {
	s := newStore(t)
	s.Create(owner, imapHost, "Zara", "", []ContactEmail{{Email: "z@example.com"}})
	s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "a@example.com"}})
	s.Create(owner, imapHost, "Bob", "", []ContactEmail{{Email: "b@example.com"}})

	list, err := s.List(owner, imapHost)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("len = %d", len(list))
	}
	if list[0].Name != "Alice" || list[1].Name != "Bob" || list[2].Name != "Zara" {
		t.Errorf("order wrong: %v", []string{list[0].Name, list[1].Name, list[2].Name})
	}
}

func TestContact_OwnerIsolation(t *testing.T) {
	s := newStore(t)
	s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})
	s.Create("bob", imapHost, "Bob", "", []ContactEmail{{Email: "bob@example.com"}})

	list, _ := s.List(owner, imapHost)
	if len(list) != 1 || list[0].Name != "Alice" {
		t.Errorf("expected only Alice, got %v", list)
	}
}

func TestContact_Autocomplete(t *testing.T) {
	s := newStore(t)
	s.Create(owner, imapHost, "Alice Smith", "", []ContactEmail{{Email: "alice@example.com"}})
	s.Create(owner, imapHost, "Bob", "", []ContactEmail{{Email: "bob@example.com"}})

	// Match by name prefix.
	res, err := s.Autocomplete(owner, imapHost, "ali")
	if err != nil {
		t.Fatalf("Autocomplete: %v", err)
	}
	if len(res) != 1 || res[0].Email != "alice@example.com" {
		t.Errorf("got %v", res)
	}

	// Match by email prefix.
	res, _ = s.Autocomplete(owner, imapHost, "bob@")
	if len(res) != 1 || res[0].Name != "Bob" {
		t.Errorf("got %v", res)
	}
}

func TestContact_FindByEmail(t *testing.T) {
	s := newStore(t)
	c, _ := s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})

	found, err := s.FindByEmail(owner, imapHost, "alice@example.com")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	if found == nil || found.ID != c.ID {
		t.Error("expected to find contact by email")
	}

	notFound, _ := s.FindByEmail(owner, imapHost, "nobody@example.com")
	if notFound != nil {
		t.Error("expected nil for unknown email")
	}
}

func TestContact_SaveFromMessage_Idempotent(t *testing.T) {
	s := newStore(t)
	c1, _ := s.SaveFromMessage(owner, imapHost, "Alice", "alice@example.com")
	c2, _ := s.SaveFromMessage(owner, imapHost, "Alice", "alice@example.com")
	if c1.ID != c2.ID {
		t.Error("SaveFromMessage should return same contact on duplicate email")
	}
	list, _ := s.List(owner, imapHost)
	if len(list) != 1 {
		t.Errorf("expected 1 contact, got %d", len(list))
	}
}

// --- Group CRUD --------------------------------------------------------------

func TestGroup_CreateAndGet(t *testing.T) {
	s := newStore(t)
	g, err := s.CreateGroup(owner, imapHost, "Team Alpha")
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if g.Name != "Team Alpha" {
		t.Errorf("Name = %q", g.Name)
	}
	if len(g.Members) != 0 {
		t.Errorf("new group should have no members, got %d", len(g.Members))
	}

	got, err := s.GetGroup(g.ID, owner, imapHost)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got == nil || got.ID != g.ID {
		t.Fatal("GetGroup returned nil or wrong group")
	}
}

func TestGroup_GetNotFound(t *testing.T) {
	s := newStore(t)
	got, err := s.GetGroup(999, owner, imapHost)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got != nil {
		t.Error("expected nil for missing group")
	}
}

func TestGroup_Update(t *testing.T) {
	s := newStore(t)
	g, _ := s.CreateGroup(owner, imapHost, "Old Name")

	updated, err := s.UpdateGroup(g.ID, owner, imapHost, "New Name")
	if err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q", updated.Name)
	}
}

func TestGroup_UpdateNotFound(t *testing.T) {
	s := newStore(t)
	got, err := s.UpdateGroup(999, owner, imapHost, "whatever")
	if err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	if got != nil {
		t.Error("expected nil for missing group")
	}
}

func TestGroup_Delete(t *testing.T) {
	s := newStore(t)
	g, _ := s.CreateGroup(owner, imapHost, "To Delete")
	if err := s.DeleteGroup(g.ID, owner, imapHost); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	got, _ := s.GetGroup(g.ID, owner, imapHost)
	if got != nil {
		t.Error("group should be gone after Delete")
	}
}

func TestGroup_ListSortedByName(t *testing.T) {
	s := newStore(t)
	s.CreateGroup(owner, imapHost, "Zeta")
	s.CreateGroup(owner, imapHost, "Alpha")
	s.CreateGroup(owner, imapHost, "Beta")

	groups, err := s.ListGroups(owner, imapHost)
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(groups) != 3 {
		t.Fatalf("len = %d", len(groups))
	}
	if groups[0].Name != "Alpha" || groups[1].Name != "Beta" || groups[2].Name != "Zeta" {
		t.Errorf("order wrong: %v", []string{groups[0].Name, groups[1].Name, groups[2].Name})
	}
}

func TestGroup_OwnerIsolation(t *testing.T) {
	s := newStore(t)
	s.CreateGroup(owner, imapHost, "Alice's Group")
	s.CreateGroup("bob", imapHost, "Bob's Group")

	groups, _ := s.ListGroups(owner, imapHost)
	if len(groups) != 1 || groups[0].Name != "Alice's Group" {
		t.Errorf("expected only Alice's group, got %v", groups)
	}
}

// --- Group membership --------------------------------------------------------

func TestGroup_AddAndRemoveMember(t *testing.T) {
	s := newStore(t)
	g, _ := s.CreateGroup(owner, imapHost, "Team")
	c, _ := s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})

	if err := s.AddGroupMember(g.ID, c.ID, owner, imapHost); err != nil {
		t.Fatalf("AddGroupMember: %v", err)
	}

	got, _ := s.GetGroup(g.ID, owner, imapHost)
	if len(got.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(got.Members))
	}
	if got.Members[0].ContactID != c.ID {
		t.Errorf("wrong member: %+v", got.Members[0])
	}
	if got.Members[0].Email != "alice@example.com" {
		t.Errorf("member email = %q", got.Members[0].Email)
	}

	if err := s.RemoveGroupMember(g.ID, c.ID, owner, imapHost); err != nil {
		t.Fatalf("RemoveGroupMember: %v", err)
	}
	got, _ = s.GetGroup(g.ID, owner, imapHost)
	if len(got.Members) != 0 {
		t.Errorf("expected 0 members after remove, got %d", len(got.Members))
	}
}

func TestGroup_AddMemberIdempotent(t *testing.T) {
	s := newStore(t)
	g, _ := s.CreateGroup(owner, imapHost, "Team")
	c, _ := s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})

	s.AddGroupMember(g.ID, c.ID, owner, imapHost)
	if err := s.AddGroupMember(g.ID, c.ID, owner, imapHost); err != nil {
		t.Fatalf("adding duplicate member should not error: %v", err)
	}
	got, _ := s.GetGroup(g.ID, owner, imapHost)
	if len(got.Members) != 1 {
		t.Errorf("expected 1 member after duplicate add, got %d", len(got.Members))
	}
}

func TestGroup_AddMemberInvalidGroup(t *testing.T) {
	s := newStore(t)
	c, _ := s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})
	if err := s.AddGroupMember(999, c.ID, owner, imapHost); err == nil {
		t.Error("expected error for non-existent group")
	}
}

func TestGroup_AddMemberInvalidContact(t *testing.T) {
	s := newStore(t)
	g, _ := s.CreateGroup(owner, imapHost, "Team")
	if err := s.AddGroupMember(g.ID, 999, owner, imapHost); err == nil {
		t.Error("expected error for non-existent contact")
	}
}

func TestGroup_DeleteCascadesMemberships(t *testing.T) {
	s := newStore(t)
	g, _ := s.CreateGroup(owner, imapHost, "Team")
	c, _ := s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})
	s.AddGroupMember(g.ID, c.ID, owner, imapHost)

	s.DeleteGroup(g.ID, owner, imapHost)

	// Contact should still exist.
	contact, _ := s.Get(c.ID, owner, imapHost)
	if contact == nil {
		t.Error("contact should not be deleted when group is deleted")
	}
}

func TestGroup_MultipleMembers(t *testing.T) {
	s := newStore(t)
	g, _ := s.CreateGroup(owner, imapHost, "Team")
	c1, _ := s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})
	c2, _ := s.Create(owner, imapHost, "Bob", "", []ContactEmail{{Email: "bob@example.com"}})
	c3, _ := s.Create(owner, imapHost, "Carol", "", []ContactEmail{{Email: "carol@example.com"}})

	s.AddGroupMember(g.ID, c1.ID, owner, imapHost)
	s.AddGroupMember(g.ID, c2.ID, owner, imapHost)
	s.AddGroupMember(g.ID, c3.ID, owner, imapHost)

	got, _ := s.GetGroup(g.ID, owner, imapHost)
	if len(got.Members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(got.Members))
	}
}

// --- AutocompleteAll ---------------------------------------------------------

func TestAutocompleteAll_ReturnsContacts(t *testing.T) {
	s := newStore(t)
	s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})

	results, err := s.AutocompleteAll(owner, imapHost, "ali")
	if err != nil {
		t.Fatalf("AutocompleteAll: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != "contact" {
		t.Errorf("Type = %q, want contact", results[0].Type)
	}
	if results[0].Email != "alice@example.com" {
		t.Errorf("Email = %q", results[0].Email)
	}
}

func TestAutocompleteAll_ReturnsGroups(t *testing.T) {
	s := newStore(t)
	c, _ := s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})
	g, _ := s.CreateGroup(owner, imapHost, "Team Alpha")
	s.AddGroupMember(g.ID, c.ID, owner, imapHost)

	results, err := s.AutocompleteAll(owner, imapHost, "team")
	if err != nil {
		t.Fatalf("AutocompleteAll: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != "group" {
		t.Errorf("Type = %q, want group", results[0].Type)
	}
	if results[0].GroupID != g.ID {
		t.Errorf("GroupID = %d", results[0].GroupID)
	}
	if len(results[0].Emails) != 1 {
		t.Errorf("group Emails = %v", results[0].Emails)
	}
}

func TestAutocompleteAll_MixedResults(t *testing.T) {
	s := newStore(t)
	// "al" matches contact "Alice" and group "All Hands".
	s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})
	s.CreateGroup(owner, imapHost, "All Hands")

	results, err := s.AutocompleteAll(owner, imapHost, "al")
	if err != nil {
		t.Fatalf("AutocompleteAll: %v", err)
	}
	types := map[string]int{}
	for _, r := range results {
		types[r.Type]++
	}
	if types["contact"] != 1 {
		t.Errorf("expected 1 contact hit, got %d", types["contact"])
	}
	if types["group"] != 1 {
		t.Errorf("expected 1 group hit, got %d", types["group"])
	}
}

func TestAutocompleteAll_EmptyPrefix(t *testing.T) {
	s := newStore(t)
	s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})

	results, err := s.AutocompleteAll(owner, imapHost, "")
	if err != nil {
		t.Fatalf("AutocompleteAll: %v", err)
	}
	// Empty prefix matches everything — at least Alice should be returned.
	if len(results) == 0 {
		t.Error("expected results for empty prefix")
	}
}

func TestAutocompleteAll_NoMatch(t *testing.T) {
	s := newStore(t)
	s.Create(owner, imapHost, "Alice", "", []ContactEmail{{Email: "alice@example.com"}})

	results, err := s.AutocompleteAll(owner, imapHost, "zzz")
	if err != nil {
		t.Fatalf("AutocompleteAll: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
}

func TestAutocompleteAll_GroupEmailsFormatted(t *testing.T) {
	s := newStore(t)
	c, _ := s.Create(owner, imapHost, "Bob Smith", "", []ContactEmail{{Email: "bob@example.com"}})
	g, _ := s.CreateGroup(owner, imapHost, "Sales")
	s.AddGroupMember(g.ID, c.ID, owner, imapHost)

	results, _ := s.AutocompleteAll(owner, imapHost, "sales")
	if len(results) != 1 || results[0].Type != "group" {
		t.Fatal("expected one group result")
	}
	// Emails should be formatted as "Name <email>" when name is set.
	if len(results[0].Emails) != 1 {
		t.Fatalf("expected 1 email in group, got %d", len(results[0].Emails))
	}
	email := results[0].Emails[0]
	if email != "Bob Smith <bob@example.com>" {
		t.Errorf("formatted email = %q", email)
	}
}
