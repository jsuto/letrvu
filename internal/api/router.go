package api

import (
	"net/http"

	"github.com/yourusername/letrvu/internal/session"
)

// NewRouter wires all HTTP routes and returns the root handler.
// Route patterns use Go 1.22 enhanced ServeMux syntax (method + path).
func NewRouter(sessions *session.Store) http.Handler {
	mux := http.NewServeMux()
	h := &handler{sessions: sessions}

	// Auth
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/logout", h.logout)

	// Folders
	mux.HandleFunc("GET /api/folders", h.requireAuth(h.listFolders))

	// Messages
	mux.HandleFunc("GET /api/folders/{folder}/messages", h.requireAuth(h.listMessages))
	mux.HandleFunc("GET /api/folders/{folder}/messages/{uid}", h.requireAuth(h.getMessage))
	mux.HandleFunc("DELETE /api/folders/{folder}/messages/{uid}", h.requireAuth(h.deleteMessage))
	mux.HandleFunc("PATCH /api/folders/{folder}/messages/{uid}/read", h.requireAuth(h.markRead))
	mux.HandleFunc("GET /api/folders/{folder}/messages/{uid}/attachments/{index}", h.requireAuth(h.downloadAttachment))

	// Compose
	mux.HandleFunc("POST /api/send", h.requireAuth(h.sendMessage))

	// SSE — real-time new mail notifications via IMAP IDLE
	mux.HandleFunc("GET /api/events", h.requireAuth(h.events))

	// Serve embedded Vue frontend for all non-API routes
	mux.Handle("/", spaHandler())

	return mux
}
