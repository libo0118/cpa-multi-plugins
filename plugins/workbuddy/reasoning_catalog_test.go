package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestReasoningCatalogShapes(t *testing.T) {
	for _, raw := range []string{
		`{"id":"deepseek-v4.1-flash","reasoning":{"supportedEfforts":["low","high","max"],"canDisableThinking":true}}`,
		`{"id":"test-flat","supportedEfforts":["low","high","max"],"canDisableThinking":true}`,
	} {
		var model discoveredModel
		if err := json.Unmarshal([]byte(raw), &model); err != nil {
			t.Fatal(err)
		}
		info := discoverToInfo(model)
		if info.Thinking == nil || !info.Thinking.ZeroAllowed || !reflect.DeepEqual(info.Thinking.Levels, []string{"low", "high", "max"}) {
			t.Fatalf("thinking = %#v", info.Thinking)
		}
	}
	obj := map[string]any{"model": "hy3", "reasoning_effort": "low"}
	if forceMaxThinking(obj) || obj["reasoning_effort"] != "low" {
		t.Fatal("advertised Hy3 low must survive request preparation")
	}
}
