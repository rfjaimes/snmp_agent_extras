package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadData(t *testing.T) {
	data := `sda               0.00     3.07    0.03    4.93     0.13  1505.33   606.23     0.49   99.65   12.00  100.24   4.48   2.23
dm-0               1.23     2.44    5.03    9.93     8.13  15.33   60.28     8.49   22.65   5.12  10.34   0.00   3.03
dm-1               8.23     8.44    8.03    8.93     8.88  18.33   80.28     8.89   32.65   8.12  0.00   0.00   0.00`

	expected_sda := StatsData{"sda", 0.00, 3.07, 0.03, 4.93, 0.13, 1505.33, 606.23, 0.49, 99.65, 12.00, 100.24, 4.48, 2.23}
	expected_dm0 := StatsData{"dm-0", 1.23, 2.44, 5.03, 9.93, 8.13, 15.33, 60.28, 8.49, 22.65, 5.12, 10.34, 0.00, 3.03}
	expected_dm1 := StatsData{"dm-1", 8.23, 8.44, 8.03, 8.93, 8.88, 18.33, 80.28, 8.89, 32.65, 8.12, 0.00, 0.00, 0.00}

	expected_stats := map[string]StatsData{
		"sda":  expected_sda,
		"dm-0": expected_dm0,
		"dm-1": expected_dm1,
	}

	sm := NewStatsManager()
	if err := sm.Load(data); err != nil {
		t.Errorf("Error loading data: %v", err)
	}

	if uint(time.Since(sm.LastUpdated()).Seconds()) > 0 {
		t.Error("Error last updated")
	}

	if !reflect.DeepEqual(expected_stats, sm.Stats()) {
		t.Errorf("Error loading stats. Expected: %v. Got: %v", expected_stats, sm.Stats())
	}

	time.Sleep(2 * time.Second)

	// data now has 13 fields instead of 14
	data = `sda               3.07    0.03    4.93     0.13  1505.33   606.23     0.49   99.65   12.00  100.24   4.48   2.23`

	if err := sm.Load(data); err == nil {
		t.Error("Error was expected")
	}

	if uint(time.Since(sm.LastUpdated()).Seconds()) != 2 {
		t.Error("Error last updated")
	}

	// Stats should remain the same as before
	if !reflect.DeepEqual(expected_stats, sm.Stats()) {
		t.Errorf("Error loading stats. Expected: %v. Got: %v", expected_stats, sm.Stats())
	}

	time.Sleep(1 * time.Second)

	// One of the elements is not a float
	data = `sda               0.0.0     3.07    0.03    4.93     0.13  1505.33   606.23     0.49   99.65   12.00  100.24   4.48   2.23`

	if err := sm.Load(data); err == nil {
		t.Error("Error was expected")
	}

	if uint(time.Since(sm.LastUpdated()).Seconds()) != 3 {
		t.Error("Error last updated")
	}

	// Stats should remain the same as before
	if !reflect.DeepEqual(expected_stats, sm.Stats()) {
		t.Errorf("Error loading stats. Expected: %v. Got: %v", expected_stats, sm.Stats())
	}

	data = `bla2               1.11     2.22    3.33    4.44     5.55  66.66   77.77     88.88   99.99   100.100  200.200   300.300   400.400
bla1               1.00     2.00    3.00    5.00     8.00  13.00   21.00     34.00   55.00   89.00  144.00   233.00   377.00`

	expected_bla2 := StatsData{"bla2", 1.11, 2.22, 3.33, 4.44, 5.55, 66.66, 77.77, 88.88, 99.99, 100.100, 200.200, 300.300, 400.400}
	expected_bla1 := StatsData{"bla1", 1.00, 2.00, 3.00, 5.00, 8.00, 13.00, 21.00, 34.00, 55.00, 89.00, 144.00, 233.00, 377.00}

	expected_stats = map[string]StatsData{
		"bla2": expected_bla2,
		"bla1": expected_bla1,
	}

	if err := sm.Load(data); err != nil {
		t.Errorf("Error loading data: %v", err)
	}

	if uint(time.Since(sm.LastUpdated()).Seconds()) > 0 {
		t.Error("Error last updated")
	}

	if !reflect.DeepEqual(expected_stats, sm.Stats()) {
		t.Errorf("Error loading stats. Expected: %v. Got: %v", expected_stats, sm.Stats())
	}
}
