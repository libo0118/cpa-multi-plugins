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

func TestQoderEnvelopeReportsProviderDetails(t *testing.T) {
	const message = "Messages with role 'tool' must be a response to a preceding message with 'tool_calls'"
	inner := map[string]any{"error": map[string]any{"message": message + " Bearer secret-token-123456789"}}
	rawInner, _ := json.Marshal(inner)
	for _, details := range []any{string(rawInner), inner} {
		body, _ := json.Marshal(map[string]any{"code": "provider_error", "type": "provider_error", "message": "Error in upstream response", "details": details})
		frame, _ := json.Marshal(map[string]any{"body": string(body), "statusCodeValue": 400})
		_, err := unwrapQoderFrame("data:" + string(frame))
		if err == nil {
			t.Fatal("upstream rejection ignored")
		}
		for _, want := range []string{"status=400", "code=provider_error", "type=provider_error", message} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("missing %q: %v", want, err)
			}
		}
		if strings.Contains(err.Error(), "secret-token-123456789") {
			t.Fatal("credential exposed")
		}
	}
}
