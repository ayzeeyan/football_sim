package server

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"football_sim/pkg/growth"
)

func TestServerManualTrainingReportsAndStoresCappedOVR(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	p := srv.DataManager.Wonderkids[0]
	bio := srv.GrowthEngine.Biometrics[p.PlayerID]
	attrs := srv.GrowthEngine.Attributes[p.PlayerID]
	if bio == nil || attrs == nil {
		srv.worldMu.Unlock()
		t.Fatal("expected selected wonderkid growth state")
	}
	for _, key := range []string{"pace", "shooting", "passing", "dribbling", "defending", "physicality"} {
		setAttrForServerTest(attrs, key, 99)
	}
	capOVR := bio.SeasonStartOVR + growth.MaxAnnualOVRGain
	if capOVR > bio.Potential {
		capOVR = bio.Potential
	}
	if capOVR > 99 {
		capOVR = 99
	}
	p.OVR = srv.GrowthEngine.EnforceSeasonOVRCap(p.PlayerID, p.Category)
	if p.OVR != capOVR {
		srv.worldMu.Unlock()
		t.Fatalf("setup OVR=%d, want annual cap %d", p.OVR, capOVR)
	}
	attrsBefore := *attrs
	srv.worldMu.Unlock()

	for i := 0; i < 3; i++ {
		resp, err := http.Post(ts.URL+"/api/prodigies/"+p.PlayerID+"/train", "application/json", strings.NewReader(`{"focus":"technical"}`))
		if err != nil {
			t.Fatalf("manual training request %d failed: %v", i, err)
		}
		var result map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || result["status"] != "success" {
			t.Fatalf("manual training request %d response: status=%d body=%v", i, resp.StatusCode, result)
		}
		if got, ok := result["ovr"].(float64); !ok || int(got) != capOVR {
			t.Fatalf("manual training request %d reported OVR=%v, want %d", i, result["ovr"], capOVR)
		}
	}

	srv.worldMu.RLock()
	stored := p.OVR
	attrsAfter := *srv.GrowthEngine.Attributes[p.PlayerID]
	calculated := srv.GrowthEngine.CalculateOVR(p.PlayerID, p.Category)
	srv.worldMu.RUnlock()
	if stored != capOVR || calculated != capOVR {
		t.Fatalf("stored/calculated OVR=%d/%d, want %d", stored, calculated, capOVR)
	}
	if !reflect.DeepEqual(attrsAfter, attrsBefore) {
		t.Fatalf("manual training banked raw attributes at cap: before=%+v after=%+v", attrsBefore, attrsAfter)
	}
}

func setAttrForServerTest(a *growth.TechnicalAttributes, key string, value int) {
	switch key {
	case "pace":
		a.Pace = value
	case "shooting":
		a.Shooting = value
	case "passing":
		a.Passing = value
	case "dribbling":
		a.Dribbling = value
	case "defending":
		a.Defending = value
	case "physicality":
		a.Physicality = value
	}
}
