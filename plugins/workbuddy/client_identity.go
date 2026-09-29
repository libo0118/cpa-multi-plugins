package main

import "net/http"

const (
	workbuddyVersion    = "5.5.6"
	workbuddyCLIVersion = "2.137.1"
)

// CN Desktop 5.5.6 + bundled CLI 2.137.1, verified by official requests.
// Common requests such as /v3/config do not carry the chat-only IDE headers.
func workbuddySoftwareHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "WorkBuddy/"+workbuddyVersion+" WorkBuddy/"+workbuddyVersion+" CLI/"+workbuddyCLIVersion)
	req.Header.Set("X-Product", "SaaS")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
}

// X-Product-Version was absent from the verified chat request; do not invent it.
func workbuddyHeaders(req *http.Request) {
	workbuddySoftwareHeaders(req)
	req.Header.Set("X-IDE-Name", "WorkBuddy")
	req.Header.Set("X-IDE-Type", "WorkBuddy")
	req.Header.Set("X-IDE-Version", workbuddyVersion)
	req.Header.Set("X-CodeBuddy-Request", "1")
}
