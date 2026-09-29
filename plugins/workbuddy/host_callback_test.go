package main

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestHostCallbackRequestIsolation(t *testing.T) {
	// Create all requests before serializing any: a process-wide current ID
	// would mix the two executions and contaminate requests without an ID.
	ids := []string{"callback-first", "callback-second", ""}
	requests := make([]*http.Request, len(ids))
	for i, id := range ids {
		ctx, cancel := context.WithCancel(hostCallbackContext(id))
		t.Cleanup(cancel)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://example.invalid/chat", strings.NewReader("payload"))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("User-Agent", "original-client")
		requests[i] = req
	}
	for i, id := range ids {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			wire := hostHTTPRequestWire(requests[i], []byte("payload"))
			raw, err := json.Marshal(wire)
			if err != nil {
				t.Fatal(err)
			}
			var decoded rpcHostHTTPRequestWire
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.HostCallbackID != id {
				t.Fatalf("callback association = %q, want %q", decoded.HostCallbackID, id)
			}
			if id == "" && strings.Contains(string(raw), "host_callback_id") {
				t.Fatal("empty callback ID must be omitted")
			}
			if decoded.Request.Method != http.MethodPost || decoded.Request.URL != "https://example.invalid/chat" ||
				string(decoded.Request.Body) != "payload" ||
				!reflect.DeepEqual(decoded.Request.Headers, map[string][]string{"User-Agent": {"original-client"}}) {
				t.Fatalf("host association changed upstream request: %#v", decoded.Request)
			}
		})
	}
}
