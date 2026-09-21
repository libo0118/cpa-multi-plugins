package main

import "testing"

func TestLabelUsesStoredAccountRegion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		account storedAuth
		want    string
	}{
		{"international", storedAuth{Auth: storedTokens{Region: regionIntl}, Account: storedAccount{Nickname: "u390685f3"}}, "u390685f3 [INTL]"},
		{"legacy international domain", storedAuth{Auth: storedTokens{Domain: "qoder.com"}, Account: storedAccount{Nickname: "u9e0e8061"}}, "u9e0e8061 [INTL]"},
		{"domestic", storedAuth{Auth: storedTokens{Region: regionCN}, Account: storedAccount{Nickname: "local"}}, "local [CN]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := labelForAuth(&tc.account); got != tc.want {
				t.Fatalf("label=%q, want %q", got, tc.want)
			}
		})
	}
}
