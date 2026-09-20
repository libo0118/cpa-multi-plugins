package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCatalogThinkingAndRequest(t *testing.T) {
	models, err := parseQoderModelCatalog([]byte(`{"chat":[
	 {"key":"dfmodel","enable":true,"thinking_config":{"disabled":{},"enabled":{"efforts":{"low":{},"high":{},"max":{"is_default":true}}}}},
	 {"key":"qfmodel","enable":true,"thinking_config":{"disabled":{},"enabled":{"efforts":{"low":{},"medium":{},"xhigh":{}}}}},
	 {"key":"gmodel","enable":true,"thinking_config":{"enabled":{"efforts":{"low":{},"high":{},"max":{}}}}}
	]}`), "cn")
	if err != nil {
		t.Fatal(err)
	}
	for i, levels := range [][]string{{"low", "high", "max"}, {"low", "medium", "xhigh"}, {"low", "high", "max"}} {
		if models[i].Thinking == nil || !reflect.DeepEqual(models[i].Thinking.Levels, levels) || models[i].Thinking.ZeroAllowed != (i < 2) {
			t.Fatalf("model %s thinking = %#v", models[i].ID, models[i].Thinking)
		}
	}
	for _, effort := range []string{"high", "max"} {
		body, err := buildQoderBody(&openAIRequest{Messages: []openAIMessage{{Role: "user", Content: "test"}}, ReasoningEffort: effort}, "dfmodel", "personal")
		if err != nil {
			t.Fatal(err)
		}
		var payload struct {
			Parameters map[string]any `json:"parameters"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Parameters["enable_thinking"] != true || payload.Parameters["reasoning_effort"] != effort {
			t.Fatal("effort was changed or lost")
		}
	}
}
