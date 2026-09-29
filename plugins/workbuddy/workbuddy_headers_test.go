package main

import (
	"net/http"
	"testing"
)

func TestWorkbuddySoftwareHeaders(t *testing.T) {
	// Official Desktop client-info env supplies WorkBuddy/5.5.6 twice;
	// UserAgentHttpInterceptor appends the bundled CLI extension (2.137.1).
	// No verified X-Product-Version assignment: preserve it rather than infer it.
	req, _ := http.NewRequest(http.MethodPost, "https://copilot.tencent.com", nil)
	req.Header.Set("Authorization", "Bearer untouched")
	req.Header.Set("X-Product-Version", "untouched")
	workbuddyHeaders(req)
	for key, want := range map[string]string{
		"User-Agent": "WorkBuddy/5.5.6 WorkBuddy/5.5.6 CLI/2.137.1",
		"X-IDE-Name": "WorkBuddy", "X-IDE-Type": "WorkBuddy", "X-IDE-Version": "5.5.6",
		"Authorization": "Bearer untouched", "X-Product-Version": "untouched",
		"X-CodeBuddy-Request": "1", "X-Product": "SaaS", "X-Requested-With": "XMLHttpRequest",
	} {
		if got := req.Header.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}
