package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestOpenAITextContent(t *testing.T) {
	var req openAIRequest
	err := json.Unmarshal([]byte(`{"model":"qoder/Auto","stream":true,"messages":[{"role":"system","content":"instructions"},{"role":"assistant","content":null},{"role":"user","content":[{"type":"text","text":"first"},{"type":"input_text","text":"第二段"},{"type":"text","text":"last"}]}]}`), &req)
	if err != nil {
		t.Fatal(err)
	}
	want := "first\n第二段\nlast"
	if req.Messages[0].Content != "instructions" || req.Messages[1].Content != "" || messageTextContent(req.Messages[2]) != want {
		t.Fatalf("unexpected normalized messages: %+v", req.Messages)
	}
	body, err := buildQoderBody(&req, "auto", "test")
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		ChatContext struct {
			Text  struct{ Text string } `json:"text"`
			Extra struct {
				OriginalContent struct{ Text string } `json:"originalContent"`
			} `json:"extra"`
		} `json:"chat_context"`
		Messages []openAIMessage `json:"messages"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ChatContext.Text.Text != want || decoded.ChatContext.Extra.OriginalContent.Text != want {
		t.Fatalf("upstream prompt lost text: %+v", decoded.ChatContext)
	}
	if len(decoded.Messages) < len(req.Messages) {
		t.Fatal("upstream messages missing conversation")
	}
	for i, got := range decoded.Messages[len(decoded.Messages)-len(req.Messages):] {
		if !reflect.DeepEqual(got, req.Messages[i]) {
			t.Fatalf("upstream message %d: got %+v want %+v", i, got, req.Messages[i])
		}
	}
}

func TestOpenAIImageContentSurvivesQoderBody(t *testing.T) {
	for _, stream := range []bool{false, true} {
		var req openAIRequest
		input := `{"model":"qoder/Sonus","messages":[{"role":"assistant","content":null,"tool_calls":[{"id":"call-test","type":"function","function":{"name":"inspect","arguments":"{}"}}]},{"role":"tool","tool_call_id":"call-test","content":"done"},{"role":"user","content":[{"type":"text","text":"Read "},{"type":"text","text":"this image."},{"type":"image_url","image_url":{"url":"data:image/png;base64,aGVsbG8=","detail":"high"}},{"type":"text","text":" Exactly."}]}]}`
		if err := json.Unmarshal([]byte(input), &req); err != nil {
			t.Fatal(err)
		}
		req.Stream = stream
		body, err := buildQoderBody(&req, "smodel", "test")
		if err != nil {
			t.Fatal(err)
		}
		var original, upstream map[string]any
		if err := json.Unmarshal([]byte(input), &original); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(body, &upstream); err != nil {
			t.Fatal(err)
		}
		want := original["messages"].([]any)
		got := upstream["messages"].([]any)
		if !reflect.DeepEqual(got[len(got)-len(want):], want) {
			t.Fatal("image parts, order, null content, or tool fields were lost")
		}
		prompt := upstream["chat_context"].(map[string]any)["text"].(map[string]any)["text"]
		if prompt != "Read \nthis image.\n Exactly." {
			t.Fatalf("text mirror changed: %v", prompt)
		}
	}
}
