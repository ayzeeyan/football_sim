package persistence

import (
	"encoding/json"
	"testing"
)

func TestProfileReportRowsAndEvents(t *testing.T) {
	raw := readExistingSaveBytes(t)
	var full map[string]json.RawMessage
	if err := json.Unmarshal(raw, &full); err != nil {
		t.Fatal(err)
	}
	var world map[string]json.RawMessage
	if err := json.Unmarshal(full["world"], &world); err != nil {
		t.Fatal(err)
	}
	var fixtures []map[string]json.RawMessage
	if err := json.Unmarshal(world["fixtures"], &fixtures); err != nil {
		t.Fatal(err)
	}
	rowAgg := make(map[string]int)
	eventAgg := make(map[string]int)
	rows, events := 0, 0
	for _, f := range fixtures {
		r, ok := f["report"]
		if !ok {
			continue
		}
		var rep map[string]json.RawMessage
		if err := json.Unmarshal(r, &rep); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"home_xi", "away_xi", "home_bench", "away_bench"} {
			rawRows, ok := rep[key]
			if !ok || len(rawRows) == 0 {
				continue
			}
			var rowsRaw []map[string]json.RawMessage
			if err := json.Unmarshal(rawRows, &rowsRaw); err != nil {
				t.Fatal(err)
			}
			rows += len(rowsRaw)
			for _, row := range rowsRaw {
				for k, v := range row {
					rowAgg[k] += len(v)
				}
			}
		}
		var eventsRaw []map[string]json.RawMessage
		if err := json.Unmarshal(rep["events"], &eventsRaw); err != nil {
			t.Fatal(err)
		}
		events += len(eventsRaw)
		for _, ev := range eventsRaw {
			for k, v := range ev {
				eventAgg[k] += len(v)
			}
		}
	}
	t.Logf("total rows: %d, total events: %d", rows, events)
	reportFieldProfile(t, "player rows", rowAgg)
	reportFieldProfile(t, "events", eventAgg)
}
