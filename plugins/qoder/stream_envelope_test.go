package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestQoderEnvelopeErrorsAndToolPayload(t *testing.T) {
	rejected := `data:{"body":"{\"error\":{\"code\":406,\"message\":\"\",\"type\":\"intention_rejected\"}}","statusCodeValue":406}`
	if _, err := unwrapQoderFrame(rejected); err == nil || !strings.Contains(err.Error(), "intention_rejected") {
		t.Fatalf("lost upstream error: %v", err)
	}
	if _, err := aggregateQoderSSE(strings.NewReader(rejected+"\n\n"), "smodel"); err == nil {
		t.Fatal("rejected stream became successful completion")
	}
	if _, err := aggregateQoderSSE(strings.NewReader("data:{\"body\":\"[DONE]\"}\n"), "smodel"); err == nil {
		t.Fatal("empty stream became successful completion")
	}
	chunk := `{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call1","type":"function","function":{"name":"ping","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`
	outer, _ := json.Marshal(map[string]any{"body": chunk, "statusCodeValue": 200})
	got, err := unwrapQoderFrame("data:" + string(outer))
	if err != nil || got != chunk {
		t.Fatalf("tool payload lost: %v", err)
	}
	if _, err := aggregateQoderSSE(strings.NewReader("data:"+string(outer)+"\n"), "smodel"); err != nil {
		t.Fatal(err)
	}
}
