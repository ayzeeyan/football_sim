package growth

import (
	"sync"
	"testing"
)

func TestGetProdigyDataConcurrentAdultReadsKeepPubertyStageConsistent(t *testing.T) {
	ge := NewGrowthEngine(44)
	ge.RegisterProdigy("adult-read", "Concurrent Adult", 20, 181, 75, "FWD", 80, 90, 19)
	ge.Biometrics["adult-read"].PubertyStage = "Late-puberty"

	const readers = 32
	var wg sync.WaitGroup
	errs := make(chan string, readers)
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, ok := ge.GetProdigyData("adult-read", "FWD")
			if !ok {
				errs <- "adult profile was missing"
				return
			}
			if data["puberty_stage"] != "Adult frame" {
				errs <- "puberty stage was not updated in the snapshot"
				return
			}
			if growing, ok := data["still_growing"].(bool); !ok || growing {
				errs <- "adult profile was reported as still growing"
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
