package main

import (
	"net/http"
	"testing"
)

func TestBackendHeadersClientAttribution(t *testing.T) {
	for _, tc := range []struct{ domain, platform, name, kind, version string }{
		{"copilot.tencent.com", "", "WorkBuddy", "WorkBuddy", "5.5.6"},
		{"copilot.tencent.com", "CLI", "WorkBuddy", "WorkBuddy", "5.5.6"},
		{"www.codebuddy.cn", "CLI", "WorkBuddy", "WorkBuddy", "5.5.6"},
		{"workbuddy.ai", "CLI", "WorkBuddy", "WorkBuddy", "5.5.6"},
		{"copilot.tencent.com", "ide", "CodeBuddyIDE", "CodeBuddyIDE", "4.9.7"},
		{"codebuddy.ai", "ide", "CodeBuddy", "IDE", "1.100.0"},
	} {
		t.Run(tc.domain+"/"+tc.platform, func(t *testing.T) {
			sa := &storedAuth{Auth: storedTokens{Domain: tc.domain, LoginPlatform: tc.platform, AccessToken: "test-token"}}
			req, _ := http.NewRequest(http.MethodPost, endpointChatFor(sa), nil)
			backendHeaders(req, sa)
			wantUA := "WorkBuddy/5.5.6 WorkBuddy/5.5.6 CLI/2.137.1"
			wantProductVersion := ""
			wantCodebuddyRequest := "1"
			if accountRegion(sa) != regionCN || tc.platform == "ide" {
				wantUA = clientUA
				wantCodebuddyRequest = ""
			}
			if tc.platform == "ide" {
				wantUA = clientUA
				wantProductVersion = tc.version
			}
			if got := req.Header.Get("User-Agent"); got != wantUA {
				t.Errorf("User-Agent = %q, want %q", got, wantUA)
			}
			if got := req.Header.Get("X-Product-Version"); got != wantProductVersion {
				t.Errorf("X-Product-Version = %q, want %q", got, wantProductVersion)
			}
			if req.Header.Get("X-CodeBuddy-Request") != wantCodebuddyRequest || req.Header.Get("X-Product") != "SaaS" {
				t.Fatal("wrong request marker or product")
			}
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

func TestWorkbuddyIdentityScope(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://copilot.tencent.com", nil)
	commonHeaders(req)
	if req.Header.Get("User-Agent") != clientUA || req.Header.Get("X-IDE-Name") != "" {
		t.Fatal("unverified auth defaults changed")
	}
	for _, tc := range []struct {
		domain, platform string
		cn               bool
	}{
		{"www.codebuddy.cn", "CLI", true}, {"workbuddy.ai", "CLI", false}, {"copilot.tencent.com", "ide", false}, {"codebuddy.ai", "ide", false},
	} {
		req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
		billingHeaders(req, &storedAuth{Auth: storedTokens{Domain: tc.domain, LoginPlatform: tc.platform, AccessToken: "test"}})
		if tc.cn {
			if req.Header.Get("User-Agent") != "WorkBuddy/5.5.6 WorkBuddy/5.5.6 CLI/2.137.1" || req.Header.Get("X-IDE-Name") != "" || req.Header.Get("X-CodeBuddy-Request") != "" {
				t.Fatal("common request received chat-only headers")
			}
		} else if req.Header.Get("User-Agent") != "" {
			t.Fatal("non-CN billing identity changed")
		}
	}
}
