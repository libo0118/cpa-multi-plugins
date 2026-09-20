package main

import (
	"strings"
	"testing"
)

func TestWorkBuddyStreamErrorsAndBilling(t *testing.T) {
	for _, body := range []string{"data: [DONE]\n", "event: error\n", `data: {"error":{"message":"quota unavailable"}}` + "\n", `data: {"code":11128}` + "\n"} {
		if _, err := aggregateCompletion(strings.NewReader(body), "hy3"); err == nil {
			t.Fatal("accepted failed completion")
		}
		if _, err := aggregateSSEWithCollector(strings.NewReader(body), true, nil); err == nil {
			t.Fatal("accepted failed stream")
		}
	}
	body := `data: {"choices":[{"delta":{"content":"OK"}}],"usage":{"credit":0.12345,"total_tokens":3}}` + "\ndata: [DONE]\n"
	chunks, err := aggregateSSEWithCollector(strings.NewReader(body), true, nil)
	if err != nil || len(chunks) != 1 || !strings.Contains(string(chunks[0].Payload), `"credit":0.12345`) {
		t.Fatalf("billing lost: %v", err)
	}
}

func TestWorkBuddyMultiplierIsDisplayMetadata(t *testing.T) {
	info := discoverToInfo(discoveredModel{ID: "hy3", Name: "Hy3", Credits: "x0.00 credits"})
	if info.ID != "hy3" || info.Name != "Hy3" || info.DisplayName != "Hy3 · x0.00 credits" || !strings.Contains(info.Description, "x0.00") {
		t.Fatalf("multiplier changed routing: %+v", info)
	}
}
