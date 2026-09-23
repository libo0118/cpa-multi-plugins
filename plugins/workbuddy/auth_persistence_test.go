package main

import (
	"encoding/json"
	"testing"
)

func TestWorkbuddyAuthDocumentPreservesAttributionAndMetadata(t *testing.T) {
	for _, raw := range []string{
		`{"auth":{"accessToken":"a"},"account":{"uid":"u"},"disabled":true,"note":"keep","priority":7,"prefix":"team","custom":{"id":9007199254740993}}`,
		`{"type":"codebuddy-cn","provider":"workbuddy-cn","accessToken":"a","disabled":true,"note":"keep","priority":7,"prefix":"team","custom":{"id":9007199254740993}}`,
	} {
		doc, err := workbuddyAuthDocument([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if string(doc["auth_kind"]) != `"oauth"` || string(doc["type"]) != `"workbuddy"` {
			t.Fatal("classification missing")
		}
		var original map[string]json.RawMessage
		_ = json.Unmarshal([]byte(raw), &original)
		for _, key := range []string{"disabled", "note", "priority", "prefix", "custom"} {
			if string(doc[key]) != string(original[key]) {
				t.Fatalf("metadata changed: %s", key)
			}
		}
	}
	for _, raw := range []string{`null`, `[]`, `{"type":"qoder"}`, `{"provider":"codex"}`, `{"type":42}`} {
		if _, err := workbuddyAuthDocument([]byte(raw)); err == nil {
			t.Fatalf("unsafe object accepted: %s", raw)
		}
	}
}

func TestTokenRefreshPreservesCredentialSettings(t *testing.T) {
	for _, nested := range []bool{false, true} {
		doc := map[string]any{"type": "workbuddy", "auth_kind": "oauth", "disabled": true, "note": "operator note", "priority": 8, "prefix": "private", "account": map[string]any{"uid": "u", "custom": "keep"}, "custom": "keep"}
		tokenDoc := map[string]any{"accessToken": "old", "refreshToken": "old-refresh", "expiresAt": 1, "domain": "codebuddy.cn", "login_platform": "ide", "extension": "keep"}
		if nested {
			doc["auth"] = tokenDoc
		} else {
			for k, v := range tokenDoc {
				doc[k] = v
			}
		}
		original, _ := json.Marshal(doc)
		updated, err := refreshedAuthJSON(original, storedTokens{AccessToken: "new", RefreshToken: "new-refresh", ExpiresAt: 99})
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]json.RawMessage
		_ = json.Unmarshal(updated, &got)
		var before map[string]json.RawMessage
		_ = json.Unmarshal(original, &before)
		for _, key := range []string{"disabled", "note", "priority", "prefix", "account", "custom"} {
			if string(got[key]) != string(before[key]) {
				t.Fatalf("refresh changed %s", key)
			}
		}
		auth := got
		if nested {
			_ = json.Unmarshal(got["auth"], &auth)
		}
		if string(auth["accessToken"]) != `"new"` || string(auth["refreshToken"]) != `"new-refresh"` || string(auth["expiresAt"]) != "99" {
			t.Fatal("tokens not refreshed")
		}
		for _, key := range []string{"domain", "login_platform", "extension"} {
			want, _ := json.Marshal(tokenDoc[key])
			if string(auth[key]) != string(want) {
				t.Fatalf("refresh dropped auth metadata %s", key)
			}
		}
		if string(got["auth_kind"]) != `"oauth"` {
			t.Fatal("refresh lost attribution")
		}
	}
}
