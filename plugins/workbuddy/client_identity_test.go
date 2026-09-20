package main

import (
	"net/http"
	"testing"
)

func TestBackendHeadersClientAttribution(t *testing.T) {
	for _, tc := range []struct{ domain, platform, name, kind, version string }{
		{"copilot.tencent.com", "", "WorkBuddy", "WorkBuddy", "5.5.6"},
		{"copilot.tencent.com", "CLI", "WorkBuddy", "WorkBuddy", "5.5.6"},
		{"workbuddy.ai", "CLI", "WorkBuddy", "WorkBuddy", "5.5.6"},
		{"copilot.tencent.com", "ide", "CodeBuddyIDE", "CodeBuddyIDE", "4.9.7"},
		{"codebuddy.ai", "ide", "CodeBuddy", "IDE", "1.100.0"},
	} {
		t.Run(tc.domain+"/"+tc.platform, func(t *testing.T) {
			sa := &storedAuth{Auth: storedTokens{Domain: tc.domain, LoginPlatform: tc.platform, AccessToken: "test-token"}}
			req, _ := http.NewRequest(http.MethodPost, endpointChatFor(sa), nil)
			backendHeaders(req, sa)
			for header, want := range map[string]string{"X-IDE-Name": tc.name, "X-IDE-Type": tc.kind, "X-IDE-Version": tc.version, "Authorization": "Bearer test-token"} {
				if got := req.Header.Get(header); got != want {
					t.Errorf("%s = %q, want %q", header, got, want)
				}
			}
			if req.Header.Get("X-Refresh-Token") != "" {
				t.Fatal("chat must not send refresh credentials")
			}
		})
	}
}
