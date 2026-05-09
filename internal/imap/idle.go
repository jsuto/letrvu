package imap

import (
	"fmt"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	_ "github.com/emersion/go-message/charset" // register charset decoders
)

// MailboxEvent is emitted when the server reports a change to a watched folder.
type MailboxEvent struct {
	Folder      string `json:"folder"`
	NumMessages uint32 `json:"num_messages"`
}

// WatchFolder dials IMAP, selects folder in read-only mode, and starts IDLE.
// Mailbox change events are sent to the returned channel. The channel is
// closed when the connection drops or cancel is called.
func WatchFolder(host string, port int, username, password, folder string) (<-chan MailboxEvent, func(), error) {
	events := make(chan MailboxEvent, 16)

	opts := dialOptions(host)
	opts.UnilateralDataHandler = &imapclient.UnilateralDataHandler{
		Mailbox: func(data *imapclient.UnilateralDataMailbox) {
			if data.NumMessages != nil {
				select {
				case events <- MailboxEvent{Folder: folder, NumMessages: *data.NumMessages}:
				default: // slow consumer — drop rather than block the decoder
				}
			}
		},
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	c, err := imapclient.DialTLS(addr, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("imap dial: %w", err)
	}
	if err := c.Login(username, password).Wait(); err != nil {
		c.Close()
		return nil, nil, fmt.Errorf("imap login: %w", err)
	}
	if _, err := c.Select(folder, &goimap.SelectOptions{ReadOnly: true}).Wait(); err != nil {
		c.Logout().Wait()
		return nil, nil, fmt.Errorf("select %q: %w", folder, err)
	}

	idle, err := c.Idle()
	if err != nil {
		c.Logout().Wait()
		return nil, nil, fmt.Errorf("idle: %w", err)
	}

	// Close the events channel once IDLE ends (cancelled or connection dropped).
	go func() {
		idle.Wait()
		c.Logout().Wait() //nolint:errcheck
		close(events)
	}()

	cancel := func() {
		idle.Close() //nolint:errcheck
	}

	return events, cancel, nil
}
