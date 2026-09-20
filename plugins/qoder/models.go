// models.go implements the ModelProvider capability: static and per-auth
// model lists, dynamic model discovery via the upstream models API, alias
// reverse resolution (client-facing alias → upstream model id), and the
// host-config oauth-excluded-models filter.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

// wbModels is the static fallback model list for QoderWork CN. Keys mirror
// /root/qoderwork/models_list.json (KNOWLEDGE §6.2). Aliases use the qoder/
// prefix in AuthAttributes; bare IDs work too. Dynamic refresh via
// /algo/api/v2/model/list replaces this at runtime when an account is present.
func wbModels() []pluginapi.ModelInfo {
	return []pluginapi.ModelInfo{
		{ID: "auto", Name: "Auto", ContextLength: 180000, MaxCompletionTokens: 8192, OwnedBy: providerName, SupportedGenerationMethods: []string{"chat"}},
		{ID: "qmodel_preview", Name: "Qwen3.8-Max-Preview", ContextLength: 180000, MaxCompletionTokens: 8192, OwnedBy: providerName, SupportedGenerationMethods: []string{"chat"}},
		{ID: "qfmodel", Name: "Qwen3.8-Flash", ContextLength: 180000, MaxCompletionTokens: 8192, OwnedBy: providerName, SupportedGenerationMethods: []string{"chat"}},
		{ID: "qmodel_latest", Name: "Qwen3.7-Max", ContextLength: 180000, MaxCompletionTokens: 8192, OwnedBy: providerName, SupportedGenerationMethods: []string{"chat"}},
		{ID: "qmodel", Name: "Qwen3.7-Plus", ContextLength: 180000, MaxCompletionTokens: 8192, OwnedBy: providerName, SupportedGenerationMethods: []string{"chat"}},
		{ID: "q36fmodel", Name: "Qwen3.6-Flash", ContextLength: 180000, MaxCompletionTokens: 8192, OwnedBy: providerName, SupportedGenerationMethods: []string{"chat"}},
		{ID: "dmodel", Name: "DeepSeek-V4-Pro", ContextLength: 180000, MaxCompletionTokens: 8192, OwnedBy: providerName, SupportedGenerationMethods: []string{"chat"}},
		{ID: "dfmodel", Name: "DeepSeek-V4-Flash", ContextLength: 180000, MaxCompletionTokens: 8192, OwnedBy: providerName, SupportedGenerationMethods: []string{"chat"}},
		{ID: "gm51model", Name: "GLM-5.2", ContextLength: 180000, MaxCompletionTokens: 8192, OwnedBy: providerName, SupportedGenerationMethods: []string{"chat"}},
		{ID: "kmodel", Name: "Kimi-K2.7-Code", ContextLength: 180000, MaxCompletionTokens: 8192, OwnedBy: providerName, SupportedGenerationMethods: []string{"chat"}},
		{ID: "mmodel", Name: "MiniMax-M2.7", ContextLength: 180000, MaxCompletionTokens: 8192, OwnedBy: providerName, SupportedGenerationMethods: []string{"chat"}},
	}
}

// International fallback IDs supplied for the current Qoder catalog. Successful
// upstream discovery remains authoritative; a fallback is not an entitlement check.
func fallbackModels(region string) []pluginapi.ModelInfo {
	if region != regionIntl {
		return wbModels()
	}
	entries := [][2]string{
		{"auto", "Auto"}, {"ultimate", "Ultimate"},
		{"performance", "Performance"}, {"efficient", "Efficient"},
		{"lite", "Lite"}, {"smodel", "Sonus"}, {"cmodel", "Cantus"},
		{"qmodel_38max", "Qwen3.8-Max"}, {"qfmodel", "Qwen3.8-Flash"},
		{"qmodel_latest", "Qwen3.7-Max"}, {"qmodel", "Qwen3.7-Plus"},
		{"kmodel_latest", "Kimi-K3"}, {"kmodel", "Kimi-K2.8-Preview"},
		{"gmodel", "GLM-5.3"}, {"gfmodel", "GLM-5.3-Flash"},
		{"dmodel", "DeepSeek-V4-Pro"}, {"dfmodel", "DeepSeek-Flash"},
		{"mmodel", "MiniMax-M3"},
	}
	models := make([]pluginapi.ModelInfo, 0, len(entries))
	for _, entry := range entries {
		models = append(models, pluginapi.ModelInfo{ID: entry[0], Name: entry[1], DisplayName: entry[1],
			ContextLength: 180000, MaxCompletionTokens: 8192, OwnedBy: providerName,
			SupportedGenerationMethods: []string{"chat"}})
	}
	return models
}

// Guarded by dynamicModelsCache: account-specific offers must not cross credentials.
var dynamicModelsAccountKey string

func modelCatalogAccountKey(sa *storedAuth) string {
	return fmt.Sprintf("%s:%x", authRegion(sa), sha256.Sum256([]byte(sa.Auth.AccessToken)))
}

func cachedDynamicModels(key string) ([]pluginapi.ModelInfo, bool) {
	dynamicModelsCache.RLock()
	defer dynamicModelsCache.RUnlock()
	if key == dynamicModelsAccountKey && len(dynamicModelsCache.models) > 0 && time.Since(dynamicModelsCache.fetched) < dynamicModelsCacheTTL {
		return dynamicModelsCache.models, true
	}
	return nil, false
}

func storeDynamicModels(key string, models []pluginapi.ModelInfo) {
	dynamicModelsCache.Lock()
	dynamicModelsCache.models = models
	dynamicModelsAccountKey = key
	dynamicModelsCache.fetched = time.Now()
	dynamicModelsCache.Unlock()
}

func fetchDynamicModels() []pluginapi.ModelInfo {
	models := fallbackModels(loadedLoginRegion())
	files, err := hostAuthListFiles()
	if err != nil || len(files) == 0 {
		return models
	}
	// Strict filename-prefix match — same filter as host_auth.go hostAuthList.
	// (Earlier code also matched files containing "codebuddy" anywhere, which
	// would wrongly include workbuddy-*.json auths here and cause us to call
	// the qoderwork models API with a workbuddy token.)
	prefix := providerName + "-"
	for _, f := range files {
		if !strings.HasPrefix(strings.ToLower(f.Name), prefix) {
			continue
		}
		raw, err := hostAuthGetByIndex(f.AuthIndex)
		if err != nil {
			continue
		}
		sa, err := parseStored(raw)
		if err != nil || sa == nil {
			continue
		}
		key := modelCatalogAccountKey(sa)
		if cached, ok := cachedDynamicModels(key); ok {
			return cached
		}
		dyn, err := callModelsAPI(sa)
		if err == nil && len(dyn) > 0 {
			storeDynamicModels(key, dyn)
			return dyn
		}
	}
	return models
}

func fetchDynamicModelsFromStorage(storageJSON []byte) []pluginapi.ModelInfo {
	sa, err := parseStored(storageJSON)
	if err != nil || sa == nil {
		return fetchDynamicModels()
	}
	key := modelCatalogAccountKey(sa)
	if models, ok := cachedDynamicModels(key); ok {
		return models
	}
	if dyn, err := callModelsAPI(sa); err == nil && len(dyn) > 0 {
		storeDynamicModels(key, dyn)
		return dyn
	}
	return fallbackModels(authRegion(sa))
}

// fetchDynamicModels calls the QoderWork API to get the latest model list.
// Falls back to the hardcoded list on any error.
// callModelsAPI GETs /algo/api/v2/model/list from the QoderWork gateway
// with COSY signing (same as inference). Returns plain JSON (not QoderEncoding).
// Falls back to wbModels() on any error.
func callModelsAPI(sa *storedAuth) ([]pluginapi.ModelInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rawURL := endpointModelsFor(sa) // includes ?Encode=1
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	// Sign the bytes actually sent. GET has an empty body; signing encoded {} causes 403.
	if err := applyCosyHeaders(req, sa, "", rawURL, "", false); err != nil {
		return nil, fmt.Errorf("cosy sign: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", clientUA)
	resp, err := hostHTTPDo(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models API status %d", resp.StatusCode)
	}
	return parseQoderModelCatalog(resp.Body, authRegion(sa))
}

func parseQoderModelCatalog(body []byte, region string) ([]pluginapi.ModelInfo, error) {
	// Only use chat-scene pricing; other scenes can have different rates.
	var apiResp map[string]json.RawMessage
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("models parse: %w", err)
	}
	// Prefer the "chat" scene (matches our inference use case).
	chatRaw, ok := apiResp["chat"]
	if !ok {
		return nil, fmt.Errorf("no chat scene in models response")
	}
	var models []struct {
		Key                 string   `json:"key"`
		DisplayName         string   `json:"display_name"`
		Enable              bool     `json:"enable"`
		IsReasoning         bool     `json:"is_reasoning"`
		IsVL                bool     `json:"is_vl"`
		MaxInputTokens      int64    `json:"max_input_tokens"`
		PriceFactor         *float64 `json:"price_factor"`
		OriginalPriceFactor *float64 `json:"original_price_factor"`
		ThinkingConfig      map[string]struct {
			Efforts map[string]json.RawMessage `json:"efforts"`
		} `json:"thinking_config"`
	}
	if err := json.Unmarshal(chatRaw, &models); err != nil {
		return nil, fmt.Errorf("chat scene parse: %w", err)
	}
	var out []pluginapi.ModelInfo
	seen := make(map[string]bool)
	for _, m := range models {
		if m.Key == "" || seen[m.Key] {
			continue
		}
		seen[m.Key] = true
		if !m.Enable {
			continue
		}
		ctx2 := int64(180000)
		if m.MaxInputTokens > 0 {
			ctx2 = m.MaxInputTokens
		}
		name := m.DisplayName
		if name == "" {
			name = m.Key
		}
		display := name
		if m.PriceFactor != nil && *m.PriceFactor >= 0 {
			rate := fmt.Sprintf("%.2f×", *m.PriceFactor)
			if m.OriginalPriceFactor != nil && *m.OriginalPriceFactor > *m.PriceFactor {
				rate = fmt.Sprintf("%.2f× → %s", *m.OriginalPriceFactor, rate)
			}
			display += " · " + rate
		}
		out = append(out, pluginapi.ModelInfo{
			ID:                         m.Key,
			Name:                       name,
			DisplayName:                display,
			ContextLength:              ctx2,
			MaxCompletionTokens:        8192,
			OwnedBy:                    providerName,
			SupportedGenerationMethods: []string{"chat"},
			Thinking:                   qoderThinkingSupport(m.ThinkingConfig),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no enabled chat models")
	}
	if region == regionIntl {
		// Keep the existing compatibility IDs when the catalog omits them. Missing rates
		// remain unknown, while an explicit upstream disable remains authoritative.
		byID := make(map[string]pluginapi.ModelInfo)
		for _, model := range out {
			byID[model.ID] = model
		}
		merged := make([]pluginapi.ModelInfo, 0, len(out))
		for _, model := range fallbackModels(region) {
			if live, ok := byID[model.ID]; ok {
				merged = append(merged, live)
				delete(byID, model.ID)
			} else if !seen[model.ID] {
				merged = append(merged, model)
			}
		}
		for _, model := range out {
			if _, ok := byID[model.ID]; ok {
				merged = append(merged, model)
			}
		}
		out = merged
	}
	return out, nil
}

// The chat catalog is authoritative: model families do not share effort levels.
func qoderThinkingSupport(config map[string]struct {
	Efforts map[string]json.RawMessage `json:"efforts"`
}) *pluginapi.ThinkingSupport {
	if len(config) == 0 {
		return nil
	}
	enabled, ok := config["enabled"]
	if !ok || len(enabled.Efforts) == 0 {
		return nil
	}
	_, canDisable := config["disabled"]
	support := &pluginapi.ThinkingSupport{ZeroAllowed: canDisable}
	for _, effort := range []string{"low", "medium", "high", "xhigh", "max"} {
		if _, ok := enabled.Efforts[effort]; ok {
			support.Levels = append(support.Levels, effort)
		}
	}
	if len(support.Levels) == 0 {
		return nil
	}
	return support
}

func cacheModelAliases(host pluginapi.HostConfigSummary) {
	entries := host.OAuthModelAlias[providerName]
	if len(entries) == 0 {
		// Host may key the channel case-insensitively; fall back to a scan.
		for channel, list := range host.OAuthModelAlias {
			if strings.EqualFold(strings.TrimSpace(channel), providerName) {
				entries = list
				break
			}
		}
	}
	byAlias := make(map[string]string, len(entries))
	for _, e := range entries {
		name := strings.TrimSpace(e.Name)
		alias := strings.TrimSpace(e.Alias)
		if name == "" || alias == "" || strings.EqualFold(name, alias) {
			continue
		}
		byAlias[strings.ToLower(alias)] = name
	}
	modelAliasCache.Lock()
	modelAliasCache.byAlias = byAlias
	modelAliasCache.Unlock()
}

// resolveUpstreamModel maps an aliased requested model back to the real
// upstream model ID. Returns the input unchanged when nothing matches.
func resolveUpstreamModel(model string, attributes map[string]string) string {
	m := strings.TrimSpace(model)
	if m == "" {
		return model
	}
	key := strings.ToLower(m)
	if name, ok := parseModelAliasAttribute(attributes)[key]; ok {
		return name
	}
	modelAliasCache.RLock()
	name, ok := modelAliasCache.byAlias[key]
	modelAliasCache.RUnlock()
	if ok {
		return name
	}
	return m
}

// parseModelAliasAttribute decodes a per-auth alias override from auth
// attributes. Accepts JSON ([{"name":...,"alias":...}] or {alias:name}) or
// comma-separated "alias=name" pairs.
func parseModelAliasAttribute(attributes map[string]string) map[string]string {
	if len(attributes) == 0 {
		return nil
	}
	raw := ""
	for _, k := range []string{"model_alias", "model-alias", "oauth-model-alias"} {
		if v := strings.TrimSpace(attributes[k]); v != "" {
			raw = v
			break
		}
	}
	if raw == "" {
		return nil
	}
	out := make(map[string]string)
	add := func(name, alias string) {
		name, alias = strings.TrimSpace(name), strings.TrimSpace(alias)
		if name != "" && alias != "" && !strings.EqualFold(name, alias) {
			out[strings.ToLower(alias)] = name
		}
	}
	if strings.HasPrefix(raw, "[") {
		var list []struct {
			Name  string `json:"name"`
			Alias string `json:"alias"`
		}
		if json.Unmarshal([]byte(raw), &list) == nil {
			for _, e := range list {
				add(e.Name, e.Alias)
			}
			return out
		}
	}
	if strings.HasPrefix(raw, "{") {
		var m map[string]string
		if json.Unmarshal([]byte(raw), &m) == nil {
			for alias, name := range m {
				add(name, alias)
			}
			return out
		}
	}
	for _, pair := range strings.Split(raw, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			add(kv[1], kv[0])
		}
	}
	return out
}

// filterExcludedModels removes models listed in oauth-excluded-models for
// the qoderwork provider. The host passes this config via HostConfigSummary.
func filterExcludedModels(models []pluginapi.ModelInfo, host pluginapi.HostConfigSummary) []pluginapi.ModelInfo {
	if len(host.ExcludedModels) == 0 {
		return models
	}
	// Try exact provider match, then case-insensitive scan.
	excluded := host.ExcludedModels[providerName]
	if len(excluded) == 0 {
		for channel, list := range host.ExcludedModels {
			if strings.EqualFold(strings.TrimSpace(channel), providerName) {
				excluded = list
				break
			}
		}
	}
	if len(excluded) == 0 {
		return models
	}
	excludeSet := make(map[string]struct{}, len(excluded))
	for _, m := range excluded {
		excludeSet[strings.ToLower(strings.TrimSpace(m))] = struct{}{}
	}
	// Use a fresh slice — models[:0] would alias the input's backing array,
	// which may be the dynamicModelsCache's own slice. Mutating it in place
	// would corrupt the cache for subsequent callers (P0 bug: after one
	// filterExcludedModels call, cache returns the filtered list as the
	// "full" list on the next fetch).
	out := make([]pluginapi.ModelInfo, 0, len(models))
	for _, m := range models {
		if _, skip := excludeSet[strings.ToLower(m.ID)]; skip {
			continue
		}
		out = append(out, m)
	}
	return out
}

// publishUsage reports one upstream attempt into CPAMP request monitoring.
// requestedModel is client-facing (may be alias); upstreamModel is resolved.

func handleModelStatic(raw []byte) ([]byte, error) {
	var req pluginapi.StaticModelRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	cacheModelAliases(req.Host)
	models := fetchDynamicModels()
	models = filterExcludedModels(models, req.Host)
	return okEnvelope(pluginapi.ModelResponse{Provider: providerName, Models: models})
}

func handleModelForAuth(raw []byte) ([]byte, error) {
	var req pluginapi.AuthModelRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	// Always return the plugin's canonical provider key. The host skips any
	// response whose Provider doesn't match the auth's provider, so echoing
	// req.AuthProvider back would silently drop the model list whenever the
	// auth file carries a non-canonical provider string.
	cacheModelAliases(req.Host)
	models := fetchDynamicModelsFromStorage(req.StorageJSON)
	models = filterExcludedModels(models, req.Host)
	return okEnvelope(pluginapi.ModelResponse{Provider: providerName, Models: models})
}
