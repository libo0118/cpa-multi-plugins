package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSummarizeQuota(t *testing.T) {
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	var quota quotaUsageResponse
	err := json.Unmarshal([]byte(`{
        "userType":"teams","expiresAt":1893542400000,
        "userQuota":{"total":100,"used":20.25,"remaining":79.75},
        "addOnQuota":{},
        "dedicatedResourcePackages":[
            {"name":"opaque","total":50,"used":10.5,"remaining":39.5,"expiresAt":1893628800000,"available":true,"status":"QUOTA_DETAIL_STATUS_ACTIVE","displayLabels":[{"dimension":"title","value":"title","valueI18n":{"zh-CN":"SOTA 专属积分","en-US":"SOTA Exclusive Credits"}}]},
            {"name":"expired","total":100,"used":10,"remaining":90,"expiresAt":1},
            {"name":"disabled","total":100,"remaining":100,"available":false}
        ],
        "orgResourcePackage":{"used":0.25,"remaining":5.75,"cap":-1,"available":true}
    }`), &quota)
	if err != nil {
		t.Fatal(err)
	}
	got := summarizeQuota(quota, now)
	if got.TotalRemain != 125 || got.TotalUsed != 31 || got.TotalSize != 150 || got.SizeKnown || got.PackCount != 3 {
		t.Fatalf("incorrect usable totals: %+v", got)
	}
	if len(got.Packages) != 5 || got.Packages[0].Name != "套餐内 Credits (Teams)" || got.Packages[1].Name != "SOTA 专属积分" {
		t.Fatalf("incorrect packages: %+v", got.Packages)
	}
	if got.Packages[1].CycleEnd != "2030-01-03T00:00:00Z" || got.Packages[2].Available || got.Packages[3].Available {
		t.Fatalf("incorrect expiry/availability: %+v", got.Packages)
	}
	shared := got.Packages[4]
	if shared.Kind != "shared" || shared.SizeKnown || shared.Size != 0 || shared.Remain != 5.75 {
		t.Fatalf("unknown shared capacity must preserve finite balance: %+v", shared)
	}
	var domestic quotaUsageResponse
	if err := json.Unmarshal([]byte(`{"userQuota":{"total":100,"used":10.5,"remaining":89.5},"addOnQuota":{"total":50,"used":0.25,"remaining":49.75}}`), &domestic); err != nil {
		t.Fatal(err)
	}
	got = summarizeQuota(domestic, now)
	if got.TotalRemain != 139.25 || got.TotalUsed != 10.75 || got.TotalSize != 150 || !got.SizeKnown || got.PackCount != 2 || got.Packages[0].Name != "基础额度" {
		t.Fatalf("CN base and add-on regression: %+v", got)
	}
}
