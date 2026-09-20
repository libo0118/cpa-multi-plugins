package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type unavailableModelTransport struct{}
func (unavailableModelTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("model discovery unavailable in regression check")
}

func TestInternationalModelFallback(t *testing.T) {
	client := sharedHTTPClient()
	previousTransport := client.Transport
	previousRegion := loadedLoginRegion()
	client.Transport = unavailableModelTransport{}
	t.Cleanup(func() { client.Transport = previousTransport; setLoginRegion(previousRegion) })
	dynamicModelsCache.Lock()
	dynamicModelsCache.models = nil
	dynamicModelsCache.Unlock()
	want := strings.Fields("auto ultimate performance efficient lite smodel cmodel qmodel_38max qfmodel qmodel_latest qmodel kmodel_latest kmodel gmodel gfmodel dmodel dfmodel mmodel")
	check := func(models []pluginapi.ModelInfo) {
		t.Helper()
		if len(models) != len(want) { t.Fatalf("model count: got %d want %d", len(models),len(want)) }
		for i,id := range want {
			if models[i].ID != id || models[i].OwnedBy != "qoder" || models[i].DisplayName == "" { t.Fatalf("wrong model at %d: %+v",i,models[i]) }
			body, err := buildQoderBody(&openAIRequest{Messages: []openAIMessage{{Role:"user",Content:"test"}}},cpaToUpstreamKey(id),"test")
			if err != nil { t.Fatal(err) }
			var decoded map[string]any
			if err := json.Unmarshal(body,&decoded); err != nil { t.Fatal(err) }
			if decoded["model_config"].(map[string]any)["key"] != id { t.Fatalf("upstream model key changed: %s",id) }
		}
	}
	setLoginRegion(regionIntl)
	check(fetchDynamicModels())
	// Stored account region takes precedence over the region selected for new logins.
	setLoginRegion(regionCN)
	check(fetchDynamicModelsFromStorage([]byte(`{"auth":{"accessToken":"test-only","region":"intl","domain":"qoder.com"}}`)))
	setLoginRegion(regionIntl)
	cn := fetchDynamicModelsFromStorage([]byte(`{"auth":{"accessToken":"test-only","region":"cn","domain":"qoder.com.cn"}}`))
	if len(cn) != 10 { t.Fatalf("CN fallback changed: %d",len(cn)) }
}

func TestLiveQoderModelDiscovery(t *testing.T) {
	dir := os.Getenv("QODER_LIVE_AUTH_DIR")
	if dir == "" { t.Skip("live read-only probe not requested") }
	paths,err := filepath.Glob(filepath.Join(dir,"qoder-*.json"))
	if err != nil || len(paths)==0 { t.Fatal("No Qoder auth file available for probe") }
	data,err := os.ReadFile(paths[0]); if err != nil { t.Fatal("Cannot read probe auth file") }
	account,err := parseStored(data); if err != nil { t.Fatal("Cannot parse probe auth storage") }
	models,err := callModelsAPI(account)
	if err != nil { t.Logf("Upstream model discovery: %v; account region: %s",err,authRegion(account)); return }
	ids:=make([]string,0,len(models)); for _,model := range models { ids=append(ids,model.ID) }
	t.Logf("Upstream enabled model IDs: %v",ids)
}
