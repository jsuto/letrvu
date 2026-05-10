package api

import (
	"net/http"

	"github.com/yourusername/letrvu/internal/session"
	"github.com/yourusername/letrvu/internal/settings"
)

// NewRouter wires all HTTP routes and returns the root handler.
func NewRouter(sessions *session.Store, settingsStore *settings.Store, cfg ServerConfig) http.Handler {
	mux := http.NewServeMux()
	h := &handler{sessions: sessions, settings: settingsStore, config: cfg}

	// Public
	mux.HandleFunc("GET /api/config", h.getConfig)

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

	// User settings
	mux.HandleFunc("GET /api/settings", h.requireAuth(h.getSettings))
	mux.HandleFunc("PATCH /api/settings", h.requireAuth(h.updateSettings))

	// SSE — real-time new mail notifications via IMAP IDLE
	mux.HandleFunc("GET /api/events", h.requireAuth(h.events))

	// Serve embedded Vue frontend for all non-API routes
	mux.Handle("/", spaHandler())

	return mux
}
