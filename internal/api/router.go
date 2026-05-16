package api

import (
	"net/http"

	"github.com/jsuto/letrvu/internal/calendar"
	"github.com/jsuto/letrvu/internal/contacts"
	"github.com/jsuto/letrvu/internal/index"
	"github.com/jsuto/letrvu/internal/session"
	"github.com/jsuto/letrvu/internal/settings"
)

// NewRouter wires all HTTP routes and returns the root handler.
func NewRouter(sessions *session.Store, settingsStore *settings.Store, contactsStore *contacts.Store, calendarStore *calendar.Store, indexStore *index.Store, cfg ServerConfig) http.Handler {
	mux := http.NewServeMux()
	h := &handler{
		sessions:     sessions,
		settings:     settingsStore,
		contacts:     contactsStore,
		calendar:     calendarStore,
		index:        indexStore,
		config:       cfg,
		folderCache:  newFolderCache(cfg.FolderCacheTTL),
		loginLimiter: newLoginLimiter(cfg.LoginMaxAttempts, cfg.LoginWindow, cfg.LoginLockout),
	}

	// Public
	mux.HandleFunc("GET /api/config", h.getConfig)

	// Auth
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/auth/sessions", h.requireAuth(h.listSessions))
	mux.HandleFunc("DELETE /api/auth/sessions", h.requireAuth(h.logoutAllSessions))

	// Folders
	mux.HandleFunc("GET /api/folders", h.requireAuth(h.listFolders))
	mux.HandleFunc("POST /api/folders", h.requireAuth(h.createFolder))
	mux.HandleFunc("PATCH /api/folders/{folder}", h.requireAuth(h.renameFolder))
	mux.HandleFunc("DELETE /api/folders/{folder}", h.requireAuth(h.deleteFolder))
	mux.HandleFunc("POST /api/folders/{folder}/subscribe", h.requireAuth(h.subscribeFolder))
	mux.HandleFunc("DELETE /api/folders/{folder}/subscribe", h.requireAuth(h.unsubscribeFolder))

	// Global search
	mux.HandleFunc("GET /api/search", h.requireAuth(h.searchGlobal))

	// Messages
	mux.HandleFunc("GET /api/folders/{folder}/messages", h.requireAuth(h.listMessages))
	mux.HandleFunc("GET /api/folders/{folder}/messages/{uid}", h.requireAuth(h.getMessage))
	mux.HandleFunc("DELETE /api/folders/{folder}/messages/{uid}", h.requireAuth(h.deleteMessage))
	mux.HandleFunc("PATCH /api/folders/{folder}/messages/{uid}/read", h.requireAuth(h.markRead))
	mux.HandleFunc("PATCH /api/folders/{folder}/messages/{uid}/flagged", h.requireAuth(h.markFlagged))
	mux.HandleFunc("GET /api/folders/{folder}/messages/{uid}/source", h.requireAuth(h.getMessageSource))
	mux.HandleFunc("GET /api/folders/{folder}/messages/{uid}/attachments/{index}", h.requireAuth(h.downloadAttachment))
	mux.HandleFunc("POST /api/folders/{folder}/messages/{uid}/move", h.requireAuth(h.moveMessage))
	mux.HandleFunc("POST /api/folders/{folder}/messages/move", h.requireAuth(h.moveMessages))
	mux.HandleFunc("POST /api/folders/{folder}/messages/delete", h.requireAuth(h.deleteMessages))
	mux.HandleFunc("POST /api/folders/{folder}/messages/read", h.requireAuth(h.markReadMessages))
	mux.HandleFunc("POST /api/folders/{folder}/messages/spam", h.requireAuth(h.markSpam))
	mux.HandleFunc("POST /api/folders/{folder}/messages/notspam", h.requireAuth(h.notSpam))

	// Compose
	mux.HandleFunc("POST /api/send", h.requireAuth(h.sendMessage))
	mux.HandleFunc("POST /api/draft", h.requireAuth(h.saveDraft))

	// User settings
	mux.HandleFunc("GET /api/settings", h.requireAuth(h.getSettings))
	mux.HandleFunc("PATCH /api/settings", h.requireAuth(h.updateSettings))

	// Contacts — specific paths before wildcard {id}
	mux.HandleFunc("GET /api/contacts/autocomplete", h.requireAuth(h.autocompleteContacts))
	mux.HandleFunc("GET /api/contacts/export", h.requireAuth(h.exportContacts))
	mux.HandleFunc("POST /api/contacts/import", h.requireAuth(h.importContacts))
	mux.HandleFunc("POST /api/contacts/save-from-message", h.requireAuth(h.saveContactFromMessage))
	mux.HandleFunc("GET /api/contacts", h.requireAuth(h.listContacts))
	mux.HandleFunc("POST /api/contacts", h.requireAuth(h.createContact))
	mux.HandleFunc("GET /api/contacts/{id}", h.requireAuth(h.getContact))
	mux.HandleFunc("PUT /api/contacts/{id}", h.requireAuth(h.updateContact))
	mux.HandleFunc("DELETE /api/contacts/{id}", h.requireAuth(h.deleteContact))

	// Contact groups
	mux.HandleFunc("GET /api/contact-groups", h.requireAuth(h.listGroups))
	mux.HandleFunc("POST /api/contact-groups", h.requireAuth(h.createGroup))
	mux.HandleFunc("PUT /api/contact-groups/{id}", h.requireAuth(h.updateGroup))
	mux.HandleFunc("DELETE /api/contact-groups/{id}", h.requireAuth(h.deleteGroup))
	mux.HandleFunc("POST /api/contact-groups/{id}/members", h.requireAuth(h.addGroupMember))
	mux.HandleFunc("DELETE /api/contact-groups/{id}/members/{contact_id}", h.requireAuth(h.removeGroupMember))

	// Calendar — specific paths before wildcard {id}
	mux.HandleFunc("GET /api/calendar/events/export", h.requireAuth(h.exportCalendar))
	mux.HandleFunc("POST /api/calendar/events/import", h.requireAuth(h.importCalendar))
	mux.HandleFunc("POST /api/calendar/events/import-invite", h.requireAuth(h.importCalendarFromInvite))
	mux.HandleFunc("GET /api/calendar/events", h.requireAuth(h.listCalendarEvents))
	mux.HandleFunc("POST /api/calendar/events", h.requireAuth(h.createCalendarEvent))
	mux.HandleFunc("GET /api/calendar/events/{id}", h.requireAuth(h.getCalendarEvent))
	mux.HandleFunc("PUT /api/calendar/events/{id}", h.requireAuth(h.updateCalendarEvent))
	mux.HandleFunc("DELETE /api/calendar/events/{id}", h.requireAuth(h.deleteCalendarEvent))
	mux.HandleFunc("GET /api/calendar/events/{id}/ical", h.requireAuth(h.getEventICal))
	mux.HandleFunc("POST /api/calendar/events/{id}/ical", h.requireAuth(h.createEventICal))

	// PGP — private key storage and contact public key management
	mux.HandleFunc("GET /api/pgp/key", h.requireAuth(h.getPGPKey))
	mux.HandleFunc("POST /api/pgp/key", h.requireAuth(h.setPGPKey))
	mux.HandleFunc("DELETE /api/pgp/key", h.requireAuth(h.deletePGPKey))
	mux.HandleFunc("GET /api/pgp/key-for-email", h.requireAuth(h.getKeyForEmail))
	mux.HandleFunc("GET /api/pgp/wkd", h.requireAuth(h.wkdLookup))
	mux.HandleFunc("GET /api/contacts/{id}/pgpkey", h.requireAuth(h.getContactPGPKey))
	mux.HandleFunc("PUT /api/contacts/{id}/pgpkey", h.requireAuth(h.setContactPGPKey))
	mux.HandleFunc("DELETE /api/contacts/{id}/pgpkey", h.requireAuth(h.deleteContactPGPKey))

	// SSE — real-time new mail notifications via IMAP IDLE
	mux.HandleFunc("GET /api/events", h.requireAuth(h.events))

	// Serve embedded Vue frontend for all non-API routes
	mux.Handle("/", spaHandler())

	return securityHeaders(mux)
}
