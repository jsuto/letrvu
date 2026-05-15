//go:build integration

package api_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/jsuto/letrvu/internal/api"
	"github.com/jsuto/letrvu/internal/calendar"
	"github.com/jsuto/letrvu/internal/contacts"
	"github.com/jsuto/letrvu/internal/db"
	"github.com/jsuto/letrvu/internal/index"
	"github.com/jsuto/letrvu/internal/session"
	"github.com/jsuto/letrvu/internal/settings"
)

const (
	testUsername = "user@letrvu.test"
	testPassword = "hunter2"
)

// --- test environment --------------------------------------------------------

type testEnv struct {
	httpServer *httptest.Server
	imapPort   int
	memServer  *imapmemserver.Server
	user       *imapmemserver.User
	client     *http.Client
	csrf       string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// 1. In-memory IMAP backend with standard mailboxes.
	memServer := imapmemserver.New()
	user := imapmemserver.NewUser(testUsername, testPassword)
	for _, mbox := range []string{"INBOX", "Sent", "Drafts", "Trash"} {
		if err := user.Create(mbox, nil); err != nil {
			t.Fatalf("create mailbox %q: %v", mbox, err)
		}
	}
	memServer.AddUser(user)

	// 2. TLS listener with a self-signed cert (client uses InsecureSkipVerify).
	tlsCert := selfSignedCert(t)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	})
	if err != nil {
		t.Fatalf("tls.Listen: %v", err)
	}
	imapPort := ln.Addr().(*net.TCPAddr).Port

	imapSrv := imapserver.New(&imapserver.Options{
		NewSession:   func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memServer.NewSession(), nil, nil
		},
		InsecureAuth: true,
		Caps:         goimap.CapSet{goimap.CapIMAP4rev2: {}},
	})
	go imapSrv.Serve(ln) //nolint:errcheck
	t.Cleanup(func() { imapSrv.Close() })

	// 3. SQLite in-memory database.
	database, err := db.Open("sqlite", "file::memory:?cache=shared&mode=memory")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	// 4. Stores and router.
	secret := make([]byte, 32)
	rand.Read(secret) //nolint:errcheck
	sessionStore := session.NewStore(database, secret)
	settingsStore := settings.NewStore(database)
	contactsStore := contacts.NewStore(database)
	calendarStore := calendar.NewStore(database)
	indexStore := index.NewStore(database)

	router := api.NewRouter(sessionStore, settingsStore, contactsStore, calendarStore, indexStore, api.ServerConfig{
		IMAPHost:         "127.0.0.1",
		IMAPPort:         imapPort,
		FolderCacheTTL:   0, // no caching — always fresh in tests
		LoginMaxAttempts: 100,
		LoginWindow:      time.Minute,
		LoginLockout:     time.Second,
	})

	// 5. HTTP test server. Use a cookie jar so the client automatically sends
	//    session and CSRF cookies on every request.
	httpServer := httptest.NewServer(router)
	t.Cleanup(func() { httpServer.Close() })

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	env := &testEnv{
		httpServer: httpServer,
		imapPort:   imapPort,
		memServer:  memServer,
		user:       user,
		client:     client,
	}
	env.login(t)
	return env
}

// login authenticates against the test HTTP server and stores the CSRF token.
func (e *testEnv) login(t *testing.T) {
	t.Helper()
	body := fmt.Sprintf(`{"imap_host":"127.0.0.1","imap_port":%d,"smtp_host":"127.0.0.1","smtp_port":587,"username":%q,"password":%q}`,
		e.imapPort, testUsername, testPassword)
	resp := e.do(t, http.MethodPost, "/api/auth/login", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("login: status %d: %s", resp.StatusCode, b)
	}
	// Grab the CSRF token from the cookie so we can send it as a header.
	for _, c := range resp.Cookies() {
		if c.Name == "letrvu_csrf" {
			e.csrf = c.Value
		}
	}
	if e.csrf == "" {
		t.Fatal("login: no letrvu_csrf cookie in response")
	}
}

// do makes an HTTP request against the test server.
// For mutating methods the CSRF token is added automatically.
func (e *testEnv) do(t *testing.T, method, path, body string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, e.httpServer.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if e.csrf != "" && method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("X-CSRF-Token", e.csrf)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

// getJSON is a convenience wrapper that decodes the response body into v.
func (e *testEnv) getJSON(t *testing.T, path string, v any) {
	t.Helper()
	resp := e.do(t, http.MethodGet, path, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s: status %d: %s", path, resp.StatusCode, b)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("GET %s: decode: %v", path, err)
	}
}

// seed appends a raw RFC 5322 message to a mailbox and returns its UID.
func (e *testEnv) seed(t *testing.T, mailbox string, raw []byte, flags ...goimap.Flag) uint32 {
	t.Helper()
	opts := &goimap.AppendOptions{Flags: flags, Time: time.Now()}
	data, err := e.user.Append(mailbox, &literal{bytes.NewReader(raw)}, opts)
	if err != nil {
		t.Fatalf("seed %q: %v", mailbox, err)
	}
	return uint32(data.UID)
}

// literal wraps bytes.Reader to satisfy goimap.LiteralReader.
type literal struct{ *bytes.Reader }

func (l *literal) Size() int64 { return int64(l.Len()) }

// --- email fixtures ----------------------------------------------------------

func plainEmail(from, to, subject, body string) []byte {
	return []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nContent-Type: text/plain\r\n\r\n%s",
		from, to, subject,
		time.Now().Format(time.RFC1123Z),
		body,
	))
}

func htmlEmail(from, to, subject, htmlBody string) []byte {
	return []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nContent-Type: text/html; charset=utf-8\r\n\r\n%s",
		from, to, subject,
		time.Now().Format(time.RFC1123Z),
		htmlBody,
	))
}

// --- TLS helper --------------------------------------------------------------

func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("x509.CreateCertificate: %v", err)
	}
	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}
}

// =============================================================================
// Integration tests
// =============================================================================

func TestIntegration_Login(t *testing.T) {
	env := newTestEnv(t)

	// A second login with wrong password must fail.
	body := fmt.Sprintf(`{"imap_host":"127.0.0.1","imap_port":%d,"username":%q,"password":"wrong"}`,
		env.imapPort, testUsername)
	resp := env.do(t, http.MethodPost, "/api/auth/login", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("bad password: want 401, got %d", resp.StatusCode)
	}
}

func TestIntegration_ListFolders(t *testing.T) {
	env := newTestEnv(t)

	var folders []struct {
		Name string `json:"name"`
	}
	env.getJSON(t, "/api/folders", &folders)

	want := map[string]bool{"INBOX": true, "Sent": true, "Drafts": true, "Trash": true}
	for _, f := range folders {
		delete(want, f.Name)
	}
	if len(want) > 0 {
		t.Errorf("missing folders: %v", want)
	}
}

func TestIntegration_ListMessages(t *testing.T) {
	env := newTestEnv(t)

	env.seed(t, "INBOX", plainEmail("alice@example.com", testUsername, "Hello", "World"))
	env.seed(t, "INBOX", plainEmail("bob@example.com", testUsername, "Meeting", "Tomorrow"))

	var messages []struct {
		UID     uint32 `json:"uid"`
		Subject string `json:"subject"`
		From    string `json:"from"`
	}
	env.getJSON(t, "/api/folders/INBOX/messages", &messages)

	if len(messages) != 2 {
		t.Fatalf("want 2 messages, got %d", len(messages))
	}
	subjects := map[string]bool{}
	for _, m := range messages {
		subjects[m.Subject] = true
		if m.UID == 0 {
			t.Errorf("message %q has zero UID", m.Subject)
		}
	}
	for _, want := range []string{"Hello", "Meeting"} {
		if !subjects[want] {
			t.Errorf("subject %q not found in list", want)
		}
	}
}

func TestIntegration_GetMessage(t *testing.T) {
	env := newTestEnv(t)
	uid := env.seed(t, "INBOX", htmlEmail(
		"alice@example.com", testUsername, "Rich message",
		"<h1>Hello</h1><p>This is HTML.</p>",
	))

	var msg struct {
		UID      uint32 `json:"uid"`
		Subject  string `json:"subject"`
		From     string `json:"from"`
		HTMLBody string `json:"html_body"`
	}
	env.getJSON(t, fmt.Sprintf("/api/folders/INBOX/messages/%d", uid), &msg)

	if msg.UID != uid {
		t.Errorf("uid: want %d, got %d", uid, msg.UID)
	}
	if msg.Subject != "Rich message" {
		t.Errorf("subject: want %q, got %q", "Rich message", msg.Subject)
	}
	if !strings.Contains(msg.HTMLBody, "<h1>Hello</h1>") {
		t.Errorf("html_body missing expected content, got: %q", msg.HTMLBody)
	}
}

func TestIntegration_MarkRead(t *testing.T) {
	env := newTestEnv(t)
	uid := env.seed(t, "INBOX", plainEmail("alice@example.com", testUsername, "Unread", "body"))

	// Confirm message starts unread.
	var list []struct {
		UID  uint32 `json:"uid"`
		Read bool   `json:"read"`
	}
	env.getJSON(t, "/api/folders/INBOX/messages", &list)
	if len(list) == 0 {
		t.Fatal("no messages")
	}
	if list[0].Read {
		t.Error("new message should start as unread")
	}

	// Mark as read.
	resp := env.do(t, http.MethodPatch,
		fmt.Sprintf("/api/folders/INBOX/messages/%d/read", uid),
		`{"read":true}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mark read: want 200, got %d", resp.StatusCode)
	}

	// Verify it is now read.
	env.getJSON(t, "/api/folders/INBOX/messages", &list)
	for _, m := range list {
		if m.UID == uid && !m.Read {
			t.Error("message should be read after marking")
		}
	}
}

func TestIntegration_MoveMessage(t *testing.T) {
	env := newTestEnv(t)
	uid := env.seed(t, "INBOX", plainEmail("alice@example.com", testUsername, "To move", "body"))

	resp := env.do(t, http.MethodPost,
		fmt.Sprintf("/api/folders/INBOX/messages/%d/move", uid),
		`{"dest":"Trash"}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("move: want 200, got %d", resp.StatusCode)
	}

	// INBOX should now be empty.
	var inbox []struct{ UID uint32 `json:"uid"` }
	env.getJSON(t, "/api/folders/INBOX/messages", &inbox)
	if len(inbox) != 0 {
		t.Errorf("INBOX should be empty after move, got %d messages", len(inbox))
	}

	// Trash should have the message.
	var trash []struct{ UID uint32 `json:"uid"` }
	env.getJSON(t, "/api/folders/Trash/messages", &trash)
	if len(trash) != 1 {
		t.Errorf("Trash should have 1 message, got %d", len(trash))
	}
}

func TestIntegration_DeleteMessage(t *testing.T) {
	env := newTestEnv(t)
	uid := env.seed(t, "INBOX", plainEmail("alice@example.com", testUsername, "To delete", "body"))

	resp := env.do(t, http.MethodDelete,
		fmt.Sprintf("/api/folders/INBOX/messages/%d", uid), "")
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete: want 200, got %d", resp.StatusCode)
	}

	var inbox []struct{ UID uint32 `json:"uid"` }
	env.getJSON(t, "/api/folders/INBOX/messages", &inbox)
	for _, m := range inbox {
		if m.UID == uid {
			t.Error("deleted message still present in INBOX")
		}
	}
}

func TestIntegration_SearchMessages(t *testing.T) {
	env := newTestEnv(t)
	env.seed(t, "INBOX", plainEmail("alice@example.com", testUsername, "Project Alpha", "kick-off notes"))
	env.seed(t, "INBOX", plainEmail("bob@example.com", testUsername, "Lunch plans", "pizza?"))

	var results []struct {
		Subject string `json:"subject"`
	}
	env.getJSON(t, "/api/folders/INBOX/messages?q=Alpha", &results)

	if len(results) != 1 {
		t.Fatalf("search 'Alpha': want 1 result, got %d", len(results))
	}
	if results[0].Subject != "Project Alpha" {
		t.Errorf("unexpected subject: %q", results[0].Subject)
	}
}

func TestIntegration_BulkMarkRead(t *testing.T) {
	env := newTestEnv(t)
	uid1 := env.seed(t, "INBOX", plainEmail("a@x.com", testUsername, "Msg 1", "body"))
	uid2 := env.seed(t, "INBOX", plainEmail("b@x.com", testUsername, "Msg 2", "body"))

	body, _ := json.Marshal(map[string]any{"uids": []uint32{uid1, uid2}, "read": true})
	resp := env.do(t, http.MethodPost, "/api/folders/INBOX/messages/read", string(body))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bulk mark read: want 200, got %d", resp.StatusCode)
	}

	var list []struct {
		UID  uint32 `json:"uid"`
		Read bool   `json:"read"`
	}
	env.getJSON(t, "/api/folders/INBOX/messages", &list)
	for _, m := range list {
		if !m.Read {
			t.Errorf("message %d should be read", m.UID)
		}
	}
}

func TestIntegration_UnreadCountInFolderList(t *testing.T) {
	env := newTestEnv(t)
	env.seed(t, "INBOX", plainEmail("a@x.com", testUsername, "Unseen 1", "body"))
	env.seed(t, "INBOX", plainEmail("b@x.com", testUsername, "Unseen 2", "body"))

	var folders []struct {
		Name   string `json:"name"`
		Unseen uint32 `json:"unseen"`
	}
	env.getJSON(t, "/api/folders", &folders)

	for _, f := range folders {
		if f.Name == "INBOX" {
			if f.Unseen != 2 {
				t.Errorf("INBOX unseen: want 2, got %d", f.Unseen)
			}
			return
		}
	}
	t.Error("INBOX not found in folder list")
}
