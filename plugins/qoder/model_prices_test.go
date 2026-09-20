package main

import (
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestQoderCatalogMultipliers(t *testing.T) {
	raw := []byte(`{"qwork":[{"key":"auto","enable":true,"price_factor":9}],"chat":[{"key":"auto","display_name":"Auto","enable":true,"price_factor":1},{"key":"efficient","display_name":"Efficient","enable":true,"price_factor":0,"original_price_factor":0.3},{"key":"qmodel","display_name":"Qwen","enable":true},{"key":"performance","enable":false,"price_factor":1.1}]}`)
	models, err := parseQoderModelCatalog(raw, regionIntl)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]pluginapi.ModelInfo{}
	for _, m := range models {
		byID[m.ID] = m
	}
	if byID["auto"].DisplayName != "Auto · 1.00×" || byID["efficient"].DisplayName != "Efficient · 0.30× → 0.00×" {
		t.Fatal("incorrect scene/discount rate")
	}
	if byID["lite"].DisplayName != "Lite" || byID["smodel"].DisplayName != "Sonus" || strings.Contains(byID["qmodel"].DisplayName, "×") {
		t.Fatal("missing multiplier was invented or compatibility model lost")
	}
	if _, ok := byID["performance"]; ok {
		t.Fatal("disabled model exposed")
	}
	storeDynamicModels("account-a", models)
	t.Cleanup(func() {
		dynamicModelsCache.Lock()
		dynamicModelsCache.models = nil
		dynamicModelsAccountKey = ""
		dynamicModelsCache.Unlock()
	})
	if _, ok := cachedDynamicModels("account-a"); !ok {
		t.Fatal("account cache miss")
	}
	if _, ok := cachedDynamicModels("account-b"); ok {
		t.Fatal("account-specific rate leaked across credentials")
	}
}
