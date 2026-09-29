package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCosyClientIdentityConsistency(t *testing.T) {
	for _, tc := range []struct{ region, version, ua string }{
		{regionCN, "0.1.43", "Go-http-client/2.0"},
		{regionIntl, "1.1.64", "Bun/1.4.2"},
	} {
		t.Run(tc.region, func(t *testing.T) {
			session := &cosySession{CosyKey: "test-key", Info: "test-info"}
			headers, err := session.headers("test-user", "{}", "https://example.com/algo/chat", "text/event-stream", true, tc.region)
			if err != nil {
				t.Fatal(err)
			}
			if headers["cosy-version"] != tc.version || headers["user-agent"] != tc.ua {
				t.Fatal("wrong regional identity")
			}
			parts := strings.Split(strings.TrimPrefix(headers["authorization"], "Bearer COSY."), ".")
			if len(parts) != 2 {
				t.Fatal("unexpected COSY bearer format")
			}
			payload, err := base64.StdEncoding.DecodeString(parts[0])
			if err != nil {
				t.Fatal(err)
			}
			var signed map[string]string
			if err := json.Unmarshal(payload, &signed); err != nil {
				t.Fatal(err)
			}
			body, err := buildQoderBody(&openAIRequest{Messages: []openAIMessage{{Role: "user", Content: "test"}}}, "efficient", "test", tc.region)
			if err != nil {
				t.Fatal(err)
			}
			var decoded struct {
				Business struct {
					Version string `json:"version"`
				} `json:"business"`
			}
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatal(err)
			}
			if signed["cosyVersion"] != tc.version || decoded.Business.Version != tc.version {
				t.Fatal("body, header and signature versions differ")
			}
			if tc.region == regionIntl {
				for key, want := range map[string]string{"cosy-clienttype": "5", "cosy-business-product": "cli", "cosy-business-type": "agent", "cosy-scene": "assistant", "cosy-machineos": cosyMachineOS()} {
					if headers[key] != want {
						t.Errorf("%s=%q, want %q", key, headers[key], want)
					}
				}
				if _, exists := headers["cosy-clientip"]; exists {
					t.Fatal("Intl inference must not send the legacy hardcoded IP")
				}
			} else if headers["cosy-machineos"] != "" || headers["cosy-business-product"] != "" || headers["cosy-clientip"] != "169.254.198.161" {
				t.Fatal("CN identity changed")
			}
		})
	}
}

type identityTransport func(*http.Request) (*http.Response, error)

func (f identityTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestOfficialIdentityAtRequestBoundary(t *testing.T) {
	client := sharedHTTPClient()
	previous := client.Transport
	t.Cleanup(func() { client.Transport = previous })
	for _, region := range []string{regionCN, regionIntl} {
		t.Run(region, func(t *testing.T) {
			seen := map[string]http.Header{}
			client.Transport = identityTransport(func(req *http.Request) (*http.Response, error) {
				seen[req.URL.Path] = req.Header.Clone()
				status, body := http.StatusOK, `{"uid":"test"}`
				switch req.URL.Path {
				case "/algo/api/v2/model/list":
					body = `{"chat":[{"key":"efficient","enable":true}]}`
				case "/api/v1/deviceToken/poll":
					status, body = http.StatusNotFound, `{}`
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			sa := &storedAuth{Auth: storedTokens{AccessToken: "test-only", Region: region}, Account: storedAccount{UID: "identity-" + region}}
			if _, err := callModelsAPI(sa); err != nil {
				t.Fatal(err)
			}
			if _, err := fetchUserInfo("test-only", region); err != nil {
				t.Fatal(err)
			}
			if _, pending, err := pollDeviceToken("nonce", "verifier", region); err != nil || !pending {
				t.Fatalf("poll pending=%v err=%v", pending, err)
			}
			want := map[string]string{"/algo/api/v2/model/list": clientUA, "/api/v1/userinfo": clientUA, "/api/v1/deviceToken/poll": "QoderWork"}
			if region == regionIntl {
				want = map[string]string{"/algo/api/v2/model/list": "Bun/1.4.2", "/api/v1/userinfo": "qoder/1.1.64", "/api/v1/deviceToken/poll": "Bun/1.4.2"}
			}
			for path, ua := range want {
				h := seen[path]
				if h.Get("User-Agent") != ua {
					t.Errorf("%s UA=%q, want %q", path, h.Get("User-Agent"), ua)
				}
				if region == regionIntl && (h.Get("Cosy-Version") != "1.1.64" || h.Get("Cosy-MachineOS") == "") {
					t.Errorf("%s missing Intl identity", path)
				}
			}
			if seen["/algo/api/v2/model/list"].Get("Accept") != "application/json" {
				t.Fatal("model discovery accept changed")
			}
		})
	}
}
