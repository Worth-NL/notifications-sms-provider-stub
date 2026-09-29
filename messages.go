package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// ReceivedMessage is one SMS the stub accepted, kept in memory so tests
// (or a developer) can retrieve message content -- e.g. a 2FA code --
// without a human reading a log. Mirrors the shape of what each provider
// endpoint already logs (see mmg.go/firetext.go/spryng.go), just queryable
// over HTTP instead of stdout.
type ReceivedMessage struct {
	Provider  string    `json:"provider"`
	To        string    `json:"to"`
	Body      string    `json:"body"`
	Reference string    `json:"reference"`
	CreatedAt time.Time `json:"createdAt"`
}

var (
	messagesMu sync.Mutex
	messages   = []ReceivedMessage{}
)

// maxStoredMessages caps the in-memory list so a long-running dev session
// doesn't grow it unbounded.
const maxStoredMessages = 500

func recordMessage(provider, to, body, reference string) {
	messagesMu.Lock()
	defer messagesMu.Unlock()

	messages = append(messages, ReceivedMessage{
		Provider:  provider,
		To:        to,
		Body:      body,
		Reference: reference,
		CreatedAt: time.Now().UTC(),
	})
	if len(messages) > maxStoredMessages {
		messages = messages[len(messages)-maxStoredMessages:]
	}
}

// MessagesEndpoint returns every message this stub has received since it
// started, oldest first. GET /api/messages, optionally ?to=<recipient> to
// filter server-side.
func MessagesEndpoint(w http.ResponseWriter, r *http.Request) {
	to := r.URL.Query().Get("to")

	messagesMu.Lock()
	defer messagesMu.Unlock()

	result := messages
	if to != "" {
		result = make([]ReceivedMessage, 0, len(messages))
		for _, m := range messages {
			if m.To == to {
				result = append(result, m)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
