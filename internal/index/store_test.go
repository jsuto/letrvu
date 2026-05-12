package index

import (
	"testing"
	"time"

	"github.com/jsuto/letrvu/internal/db"
	"github.com/jsuto/letrvu/internal/imap"
)

const (
	username = "alice"
	imapHost = "imap.example.com"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	for _, s := range []string{
		`CREATE TABLE message_index (
			username        TEXT    NOT NULL,
			imap_host       TEXT    NOT NULL,
			folder          TEXT    NOT NULL,
			uid             INTEGER NOT NULL,
			subject         TEXT    NOT NULL DEFAULT '',
			from_addr       TEXT    NOT NULL DEFAULT '',
			date            TEXT    NOT NULL DEFAULT '',
			read            INTEGER NOT NULL DEFAULT 0,
			flagged         INTEGER NOT NULL DEFAULT 0,
			has_attachments INTEGER NOT NULL DEFAULT 0,
			size            INTEGER NOT NULL DEFAULT 0,
			message_id      TEXT    NOT NULL DEFAULT '',
			in_reply_to     TEXT    NOT NULL DEFAULT '',
			refs            TEXT    NOT NULL DEFAULT '',
			PRIMARY KEY (username, imap_host, folder, uid)
		)`,
		`CREATE TABLE folder_index_state (
			username     TEXT    NOT NULL,
			imap_host    TEXT    NOT NULL,
			folder       TEXT    NOT NULL,
			uid_validity INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (username, imap_host, folder)
		)`,
	} {
		if _, err := database.Exec(s); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}
	return database
}

func newStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(openTestDB(t))
}

func makeMsg(uid uint32, subject, from string, date time.Time) imap.Message {
	return imap.Message{
		UID:     uid,
		Subject: subject,
		From:    from,
		Date:    date,
	}
}

// ---------------------------------------------------------------------------
// Upsert / KnownUIDs
// ---------------------------------------------------------------------------

func TestUpsert_StoresMessages(t *testing.T) {
	s := newStore(t)
	msgs := []imap.Message{
		makeMsg(1, "Hello", "alice@example.com", time.Now()),
		makeMsg(2, "World", "bob@example.com", time.Now()),
	}
	if err := s.Upsert(username, imapHost, "INBOX", msgs); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	known, err := s.KnownUIDs(username, imapHost, "INBOX")
	if err != nil {
		t.Fatalf("KnownUIDs: %v", err)
	}
	if len(known) != 2 {
		t.Errorf("want 2 known UIDs, got %d", len(known))
	}
	if _, ok := known[1]; !ok {
		t.Error("UID 1 not found")
	}
	if _, ok := known[2]; !ok {
		t.Error("UID 2 not found")
	}
}

func TestUpsert_IsIdempotent(t *testing.T) {
	s := newStore(t)
	msg := makeMsg(1, "Original", "alice@example.com", time.Now())
	if err := s.Upsert(username, imapHost, "INBOX", []imap.Message{msg}); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}
	// Re-upsert with updated subject.
	msg.Subject = "Updated"
	if err := s.Upsert(username, imapHost, "INBOX", []imap.Message{msg}); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	results, err := s.Search(username, imapHost, "Updated")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Subject != "Updated" {
		t.Errorf("want subject %q, got %q", "Updated", results[0].Subject)
	}
}

func TestUpsert_IsolatedByUser(t *testing.T) {
	s := newStore(t)
	_ = s.Upsert("alice", imapHost, "INBOX", []imap.Message{makeMsg(1, "Alice msg", "a@x.com", time.Now())})
	_ = s.Upsert("bob", imapHost, "INBOX", []imap.Message{makeMsg(1, "Bob msg", "b@x.com", time.Now())})

	aliceUIDs, _ := s.KnownUIDs("alice", imapHost, "INBOX")
	bobUIDs, _ := s.KnownUIDs("bob", imapHost, "INBOX")
	if len(aliceUIDs) != 1 || len(bobUIDs) != 1 {
		t.Errorf("each user should have 1 UID; alice=%d bob=%d", len(aliceUIDs), len(bobUIDs))
	}
}

func TestUpsert_IsolatedByFolder(t *testing.T) {
	s := newStore(t)
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{makeMsg(1, "inbox", "a@x.com", time.Now())})
	_ = s.Upsert(username, imapHost, "Sent", []imap.Message{makeMsg(1, "sent", "a@x.com", time.Now())})

	inboxUIDs, _ := s.KnownUIDs(username, imapHost, "INBOX")
	sentUIDs, _ := s.KnownUIDs(username, imapHost, "Sent")
	if len(inboxUIDs) != 1 || len(sentUIDs) != 1 {
		t.Error("each folder should have exactly 1 UID")
	}
}

// ---------------------------------------------------------------------------
// Delete / DeleteFolder
// ---------------------------------------------------------------------------

func TestDelete_RemovesSpecifiedUIDs(t *testing.T) {
	s := newStore(t)
	msgs := []imap.Message{
		makeMsg(1, "A", "a@x.com", time.Now()),
		makeMsg(2, "B", "b@x.com", time.Now()),
		makeMsg(3, "C", "c@x.com", time.Now()),
	}
	_ = s.Upsert(username, imapHost, "INBOX", msgs)

	if err := s.Delete(username, imapHost, "INBOX", []uint32{1, 3}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	known, _ := s.KnownUIDs(username, imapHost, "INBOX")
	if len(known) != 1 {
		t.Errorf("want 1 remaining UID, got %d", len(known))
	}
	if _, ok := known[2]; !ok {
		t.Error("UID 2 should still be present")
	}
}

func TestDelete_EmptySliceIsNoop(t *testing.T) {
	s := newStore(t)
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{makeMsg(1, "A", "a@x.com", time.Now())})
	if err := s.Delete(username, imapHost, "INBOX", nil); err != nil {
		t.Fatalf("Delete(nil): %v", err)
	}
	known, _ := s.KnownUIDs(username, imapHost, "INBOX")
	if len(known) != 1 {
		t.Error("message should still be present after no-op delete")
	}
}

func TestDeleteFolder_RemovesAllMessagesInFolder(t *testing.T) {
	s := newStore(t)
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{
		makeMsg(1, "A", "a@x.com", time.Now()),
		makeMsg(2, "B", "b@x.com", time.Now()),
	})
	_ = s.Upsert(username, imapHost, "Sent", []imap.Message{
		makeMsg(1, "S", "s@x.com", time.Now()),
	})

	if err := s.DeleteFolder(username, imapHost, "INBOX"); err != nil {
		t.Fatalf("DeleteFolder: %v", err)
	}
	inboxUIDs, _ := s.KnownUIDs(username, imapHost, "INBOX")
	sentUIDs, _ := s.KnownUIDs(username, imapHost, "Sent")
	if len(inboxUIDs) != 0 {
		t.Errorf("INBOX should be empty, got %d UIDs", len(inboxUIDs))
	}
	if len(sentUIDs) != 1 {
		t.Error("Sent folder should be unaffected")
	}
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

func TestSearch_MatchesSubject(t *testing.T) {
	s := newStore(t)
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{
		makeMsg(1, "Project budget Q1", "cfo@example.com", time.Now()),
		makeMsg(2, "Team lunch Friday", "hr@example.com", time.Now()),
	})
	results, err := s.Search(username, imapHost, "budget")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].UID != 1 {
		t.Errorf("want UID 1, got %d", results[0].UID)
	}
}

func TestSearch_MatchesSender(t *testing.T) {
	s := newStore(t)
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{
		makeMsg(1, "Hello", "alice@corp.example.com", time.Now()),
		makeMsg(2, "Hi", "bob@other.example.com", time.Now()),
	})
	results, err := s.Search(username, imapHost, "corp.example")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].UID != 1 {
		t.Errorf("expected UID 1 in results, got %+v", results)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	s := newStore(t)
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{
		makeMsg(1, "Invoice OVERDUE", "billing@x.com", time.Now()),
	})
	results, _ := s.Search(username, imapHost, "overdue")
	if len(results) != 1 {
		t.Errorf("case-insensitive search: want 1 result, got %d", len(results))
	}
}

func TestSearch_ReturnsResultsFromMultipleFolders(t *testing.T) {
	s := newStore(t)
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{
		makeMsg(1, "Report Q1", "a@x.com", time.Now()),
	})
	_ = s.Upsert(username, imapHost, "Archive", []imap.Message{
		makeMsg(10, "Report Q2", "b@x.com", time.Now()),
	})
	results, _ := s.Search(username, imapHost, "Report")
	if len(results) != 2 {
		t.Errorf("want 2 cross-folder results, got %d", len(results))
	}
}

func TestSearch_ResultsIncludeFolderField(t *testing.T) {
	s := newStore(t)
	_ = s.Upsert(username, imapHost, "Sent", []imap.Message{
		makeMsg(5, "My sent mail", "me@x.com", time.Now()),
	})
	results, _ := s.Search(username, imapHost, "sent mail")
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Folder != "Sent" {
		t.Errorf("want folder %q, got %q", "Sent", results[0].Folder)
	}
}

func TestSearch_SortedNewestFirst(t *testing.T) {
	s := newStore(t)
	old := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{
		makeMsg(1, "old report", "a@x.com", old),
		makeMsg(2, "new report", "b@x.com", recent),
	})
	results, _ := s.Search(username, imapHost, "report")
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	if results[0].UID != 2 {
		t.Errorf("newest message should be first; got UID %d first", results[0].UID)
	}
}

func TestSearch_NoResultsReturnsEmptySlice(t *testing.T) {
	s := newStore(t)
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{
		makeMsg(1, "Hello world", "a@x.com", time.Now()),
	})
	results, err := s.Search(username, imapHost, "zzznomatch")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if results != nil && len(results) != 0 {
		t.Errorf("want empty results, got %d", len(results))
	}
}

func TestSearch_DoesNotReturnOtherUsersMessages(t *testing.T) {
	s := newStore(t)
	_ = s.Upsert("alice", imapHost, "INBOX", []imap.Message{
		makeMsg(1, "Secret report", "a@x.com", time.Now()),
	})
	results, _ := s.Search("bob", imapHost, "report")
	if len(results) != 0 {
		t.Errorf("bob should not see alice's messages, got %d results", len(results))
	}
}

// ---------------------------------------------------------------------------
// UpdateRead / UpdateFlagged
// ---------------------------------------------------------------------------

func TestUpdateRead_ChangesReadFlag(t *testing.T) {
	s := newStore(t)
	msg := makeMsg(1, "Test", "a@x.com", time.Now())
	msg.Read = false
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{msg})

	s.UpdateRead(username, imapHost, "INBOX", 1, true)

	results, _ := s.Search(username, imapHost, "Test")
	if len(results) != 1 || !results[0].Read {
		t.Error("UpdateRead should have set read=true")
	}
}

func TestUpdateRead_DoesNotChangeFlaggedField(t *testing.T) {
	s := newStore(t)
	msg := makeMsg(1, "Test", "a@x.com", time.Now())
	msg.Flagged = true
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{msg})

	s.UpdateRead(username, imapHost, "INBOX", 1, false)

	results, _ := s.Search(username, imapHost, "Test")
	if len(results) != 1 || !results[0].Flagged {
		t.Error("UpdateRead should not have changed the flagged field")
	}
}

func TestUpdateFlagged_ChangesFlaggedField(t *testing.T) {
	s := newStore(t)
	msg := makeMsg(1, "Test", "a@x.com", time.Now())
	msg.Flagged = false
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{msg})

	s.UpdateFlagged(username, imapHost, "INBOX", 1, true)

	results, _ := s.Search(username, imapHost, "Test")
	if len(results) != 1 || !results[0].Flagged {
		t.Error("UpdateFlagged should have set flagged=true")
	}
}

func TestUpdateFlagged_DoesNotChangeReadField(t *testing.T) {
	s := newStore(t)
	msg := makeMsg(1, "Test", "a@x.com", time.Now())
	msg.Read = true
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{msg})

	s.UpdateFlagged(username, imapHost, "INBOX", 1, false)

	results, _ := s.Search(username, imapHost, "Test")
	if len(results) != 1 || !results[0].Read {
		t.Error("UpdateFlagged should not have changed the read field")
	}
}

// ---------------------------------------------------------------------------
// UIDValidity / SetUIDValidity
// ---------------------------------------------------------------------------

func TestUIDValidity_ZeroWhenUnknown(t *testing.T) {
	s := newStore(t)
	if v := s.UIDValidity(username, imapHost, "INBOX"); v != 0 {
		t.Errorf("want 0 for unknown folder, got %d", v)
	}
}

func TestSetUIDValidity_PersistsValue(t *testing.T) {
	s := newStore(t)
	s.SetUIDValidity(username, imapHost, "INBOX", 12345)
	if v := s.UIDValidity(username, imapHost, "INBOX"); v != 12345 {
		t.Errorf("want 12345, got %d", v)
	}
}

func TestSetUIDValidity_CanBeUpdated(t *testing.T) {
	s := newStore(t)
	s.SetUIDValidity(username, imapHost, "INBOX", 100)
	s.SetUIDValidity(username, imapHost, "INBOX", 200)
	if v := s.UIDValidity(username, imapHost, "INBOX"); v != 200 {
		t.Errorf("want 200 after update, got %d", v)
	}
}

func TestSetUIDValidity_IsolatedByFolder(t *testing.T) {
	s := newStore(t)
	s.SetUIDValidity(username, imapHost, "INBOX", 111)
	s.SetUIDValidity(username, imapHost, "Sent", 222)
	if v := s.UIDValidity(username, imapHost, "INBOX"); v != 111 {
		t.Errorf("INBOX: want 111, got %d", v)
	}
	if v := s.UIDValidity(username, imapHost, "Sent"); v != 222 {
		t.Errorf("Sent: want 222, got %d", v)
	}
}

// ---------------------------------------------------------------------------
// IndexedCount
// ---------------------------------------------------------------------------

func TestIndexedCount_ZeroWhenEmpty(t *testing.T) {
	s := newStore(t)
	if n := s.IndexedCount(username, imapHost); n != 0 {
		t.Errorf("want 0, got %d", n)
	}
}

func TestIndexedCount_ReflectsTotalAcrossFolders(t *testing.T) {
	s := newStore(t)
	_ = s.Upsert(username, imapHost, "INBOX", []imap.Message{
		makeMsg(1, "A", "a@x.com", time.Now()),
		makeMsg(2, "B", "b@x.com", time.Now()),
	})
	_ = s.Upsert(username, imapHost, "Sent", []imap.Message{
		makeMsg(1, "C", "c@x.com", time.Now()),
	})
	if n := s.IndexedCount(username, imapHost); n != 3 {
		t.Errorf("want 3, got %d", n)
	}
}
