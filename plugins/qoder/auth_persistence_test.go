package main

import (
	"encoding/json"
	"testing"
)

func TestQoderCredentialsDeclareOAuth(t *testing.T) {
	sa := &storedAuth{Auth: storedTokens{AccessToken: "token", RefreshToken: "refresh", PersonalToken: "pat", Domain: "qoder.com", Region: "intl"}, Account: storedAccount{UID: "u"}}
	if got := toAuthData(sa).Metadata["auth_kind"]; got != "oauth" {
		t.Errorf("runtime OAuth classification missing: %v", got)
	}
	raw, err := buildAuthFileJSON(sa, true, "keep note", nil)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["auth_kind"] != "oauth" {
		t.Errorf("lifecycle/import OAuth classification missing: %v", doc["auth_kind"])
	}
	if doc["disabled"] != true || doc["note"] != "keep note" {
		t.Fatal("credential settings changed")
	}
}

func TestQoderAuthDocumentPreservesMetadata(t *testing.T) {
	for _, raw := range []string{
		`{"auth":{"accessToken":"old"},"account":{"uid":"u"},"disabled":true,"priority":4,"prefix":"team","custom":{"id":9007199254740993}}`,
		`{"type":"qoder-intl","provider":"qoderwork","accessToken":"old","disabled":true,"priority":4,"prefix":"team","custom":{"id":9007199254740993}}`,
	} {
		doc, err := qoderAuthDocument([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		var before map[string]json.RawMessage
		_ = json.Unmarshal([]byte(raw), &before)
		for _, key := range []string{"disabled", "priority", "prefix", "custom"} {
			if string(doc[key]) != string(before[key]) {
				t.Fatalf("changed %s", key)
			}
		}
		if string(doc["auth_kind"]) != `"oauth"` || string(doc["provider"]) != `"qoder"` {
			t.Fatal("classification missing")
		}
	}
	for _, raw := range []string{`null`, `[]`, `{"type":"workbuddy"}`, `{"provider":"codex"}`, `{"type":42}`} {
		if _, err := qoderAuthDocument([]byte(raw)); err == nil {
			t.Fatalf("unsafe object accepted: %s", raw)
		}
	}
}

func TestQoderTokenRefreshPreservesCredentialSettings(t *testing.T) {
	for _, nested := range []bool{false, true} {
		doc := map[string]any{"type": "qoder", "disabled": true, "note": "operator note", "priority": 8, "prefix": "private", "account": map[string]any{"uid": "u", "enterpriseId": "team", "custom": "keep"}, "custom": "keep"}
		tokens := map[string]any{"accessToken": "old", "refreshToken": "old-refresh", "personalToken": "keep-pat", "expiresAt": 1, "domain": "qoder.com", "region": "intl", "extension": "keep"}
		if nested {
			doc["auth"] = tokens
		} else {
			for k, v := range tokens {
				doc[k] = v
			}
		}
		raw, _ := json.Marshal(doc)
		out, err := refreshedAuthJSON(raw, storedTokens{AccessToken: "new", RefreshToken: "new-refresh", ExpiresAt: 99})
		if err != nil {
			t.Fatal(err)
		}
		var before, after map[string]json.RawMessage
		_ = json.Unmarshal(raw, &before)
		_ = json.Unmarshal(out, &after)
		for _, key := range []string{"disabled", "note", "priority", "prefix", "account", "custom"} {
			if string(before[key]) != string(after[key]) {
				t.Fatalf("refresh changed %s", key)
			}
		}
		auth := after
		if nested {
			auth = nil
			_ = json.Unmarshal(after["auth"], &auth)
		}
		if string(auth["accessToken"]) != `"new"` || string(auth["refreshToken"]) != `"new-refresh"` || string(auth["expiresAt"]) != "99" {
			t.Fatal("tokens not updated")
		}
		for _, key := range []string{"personalToken", "domain", "region", "extension"} {
			want, _ := json.Marshal(tokens[key])
			if string(auth[key]) != string(want) {
				t.Fatalf("refresh dropped %s", key)
			}
		}
		if string(after["auth_kind"]) != `"oauth"` {
			t.Fatal("OAuth classification lost")
		}
	}
}
