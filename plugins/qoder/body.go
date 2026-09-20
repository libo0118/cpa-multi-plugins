// body.go constructs the QoderWork agent_chat_generation request body from
// OpenAI-style chat completion inputs.
//
// The base template lives in baseprompt.json (embedded). Per-request we
// overwrite request/session ids, timestamps, model key, and the user prompt.
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

//go:embed baseprompt.json
var basepromptJSON []byte

// cpaToUpstreamKey maps CPA-facing model names to upstream keys.
// Unknown names pass through unchanged (server silently routes to auto).
func cpaToUpstreamKey(cpaModel string) string {
	switch cpaModel {
	case "qoder-auto", "auto":
		return "auto"
	case "qwen3.8-max-preview", "qwen3.8-max", "qmodel_preview":
		return "qmodel_preview"
	case "qwen3.7-max", "qmodel_latest":
		return "qmodel_latest"
	case "qwen3.7-plus", "qmodel":
		return "qmodel"
	case "qwen3.6-flash", "q36fmodel":
		return "q36fmodel"
	case "deepseek-v4-pro", "dmodel":
		return "dmodel"
	case "deepseek-v4-flash", "dfmodel":
		return "dfmodel"
	case "glm-5.2", "gm51model":
		return "gm51model"
	case "kimi-k2.7-code", "kmodel":
		return "kmodel"
	case "minimax-m2.7", "mmodel":
		return "mmodel"
	}
	return cpaModel
}

// openAIMessage is one message in the OpenAI chat completion format.
type openAIMessage struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	rawContent json.RawMessage
	rawFields  map[string]json.RawMessage
}

func (m *openAIMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	delete(fields, "role")
	delete(fields, "content")
	content := bytes.TrimSpace(raw.Content)
	var text string
	switch {
	case len(content) == 0 || bytes.Equal(content, []byte("null")):
	case content[0] == '"':
		if err := json.Unmarshal(content, &text); err != nil {
			return err
		}
	case content[0] == '[':
		var parts []struct {
			Type     string  `json:"type"`
			Text     *string `json:"text"`
			ImageURL *struct {
				URL string `json:"url"`
			} `json:"image_url"`
		}
		if err := json.Unmarshal(content, &parts); err != nil {
			return fmt.Errorf("unsupported message content array: %w", err)
		}
		var joined strings.Builder
		for i, part := range parts {
			if part.Type == "image_url" {
				if part.ImageURL == nil || strings.TrimSpace(part.ImageURL.URL) == "" {
					return fmt.Errorf("unsupported message content part %d: image_url.url must be a nonempty string", i)
				}
				continue
			}
			if part.Type != "text" && part.Type != "input_text" {
				return fmt.Errorf("unsupported message content part %d type %q", i, part.Type)
			}
			if part.Text == nil {
				return fmt.Errorf("unsupported message content part %d: text must be a string", i)
			}
			joined.WriteString(*part.Text)
		}
		text = joined.String()
	default:
		return fmt.Errorf("unsupported message content: expected string, null, or content array")
	}
	*m = openAIMessage{Role: raw.Role, Content: text, rawFields: fields}
	if len(content) > 0 && content[0] != '"' {
		m.rawContent = append(json.RawMessage(nil), content...)
	}
	return nil
}

func (m openAIMessage) MarshalJSON() ([]byte, error) {
	fields := make(map[string]json.RawMessage, len(m.rawFields)+2)
	for name, value := range m.rawFields {
		fields[name] = value
	}
	fields["role"], _ = json.Marshal(m.Role)
	if len(m.rawContent) > 0 {
		fields["content"] = m.rawContent
	} else {
		fields["content"], _ = json.Marshal(m.Content)
	}
	return json.Marshal(fields)
}

// openAIRequest is the CPA-facing chat completion request.
type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

// extractLatestUserPrompt returns the content of the last user message.
func extractLatestUserPrompt(messages []openAIMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	return ""
}

// buildQoderBody renders the upstream agent_chat_generation body for one request.
// modelKey is the upstream key (already mapped via cpaToUpstreamKey).
func buildQoderBody(req *openAIRequest, modelKey, userType string) ([]byte, error) {
	var base map[string]any
	if err := json.Unmarshal(basepromptJSON, &base); err != nil {
		return nil, fmt.Errorf("baseprompt decode: %w", err)
	}

	prompt := extractLatestUserPrompt(req.Messages)
	if prompt == "" {
		return nil, fmt.Errorf("no user message in request")
	}

	nid := uuid.NewString()
	base["request_id"] = nid
	base["chat_record_id"] = nid
	base["request_set_id"] = uuid.NewString()
	base["session_id"] = uuid.NewString()
	base["stream"] = true
	base["aliyun_user_type"] = userType
	base["agent_id"] = "agent_common"

	// model_config
	if mc, ok := base["model_config"].(map[string]any); ok {
		mc["key"] = modelKey
	}

	// chat_context.text.text + chat_context.extra.originalContent.text
	if cc, ok := base["chat_context"].(map[string]any); ok {
		if txt, ok := cc["text"].(map[string]any); ok {
			txt["text"] = prompt
		}
		if extra, ok := cc["extra"].(map[string]any); ok {
			if oc, ok := extra["originalContent"].(map[string]any); ok {
				oc["text"] = prompt
			}
			if mc, ok := extra["modelConfig"].(map[string]any); ok {
				mc["key"] = modelKey
			}
		}
	}

	// messages: keep system prompt from baseprompt (template has it), replace user/assistant
	var systemMsgs []any
	if msgs, ok := base["messages"].([]any); ok {
		for _, m := range msgs {
			if mm, ok := m.(map[string]any); ok {
				if role, _ := mm["role"].(string); role == "system" {
					systemMsgs = append(systemMsgs, m)
				}
			}
		}
	}
	// Append the actual conversation
	for _, m := range req.Messages {
		systemMsgs = append(systemMsgs, m)
	}
	base["messages"] = systemMsgs

	// business
	if biz, ok := base["business"].(map[string]any); ok {
		biz["id"] = uuid.NewString()
		biz["begin_at"] = time.Now().UnixMilli()
		if len(prompt) > 30 {
			biz["name"] = prompt[:30]
		} else {
			biz["name"] = prompt
		}
	}

	return json.Marshal(base)
}
