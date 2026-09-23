package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Every physical save must retain the credential kind, including host callbacks
// that rebuild runtime metadata directly from the file rather than auth.parse.
func qoderAuthDocument(raw []byte) (map[string]json.RawMessage, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil || doc == nil {
		return nil, fmt.Errorf("invalid Qoder credential object")
	}
	for _, key := range []string{"type", "provider"} {
		if value, exists := doc[key]; exists {
			var declared string
			if json.Unmarshal(value, &declared) != nil || (strings.TrimSpace(declared) != "" && !isOurDeclaredType(declared)) {
				return nil, fmt.Errorf("credential %s does not belong to Qoder", key)
			}
		}
	}
	doc["type"], doc["provider"], doc["auth_kind"] = json.RawMessage(`"qoder"`), json.RawMessage(`"qoder"`), json.RawMessage(`"oauth"`)
	return doc, nil
}

// Update only the rotating token pair and expiry; preserve PAT, region, account
// identity and operator settings in both nested and legacy flat credentials.
func refreshedAuthJSON(original []byte, tokens storedTokens) ([]byte, error) {
	doc, err := qoderAuthDocument(original)
	if err != nil {
		return nil, err
	}
	auth := doc
	_, nested := doc["auth"]
	if nested {
		auth = nil
		if err := json.Unmarshal(doc["auth"], &auth); err != nil || auth == nil {
			return nil, fmt.Errorf("invalid Qoder auth object")
		}
	}
	for key, value := range map[string]any{"accessToken": tokens.AccessToken, "refreshToken": tokens.RefreshToken, "expiresAt": tokens.ExpiresAt} {
		auth[key], err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
	}
	if nested {
		doc["auth"], err = json.Marshal(auth)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(doc)
}
