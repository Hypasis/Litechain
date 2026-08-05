package version

import (
	"testing"
	"time"
)

func TestSemVerComparison(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"2.0.0", "1.5.0", 1},
		{"2.0.0", "2.0.0", 0},
		{"1.9.9", "2.0.0", -1},
		{"v2.1.0", "v2.0.5", 1},
		{"v2.0.0", "v2.0.0", 0},
	}

	for _, tt := range tests {
		res := CompareVersions(tt.v1, tt.v2)
		if res != tt.expected {
			t.Errorf("CompareVersions(%s, %s) = %d, expected %d", tt.v1, tt.v2, res, tt.expected)
		}
	}
}

func TestNetworkUpgradeManager(t *testing.T) {
	manager := NewNetworkUpgradeManager()

	// Register custom upgrade plan requiring min version 2.5.0 at height 500
	plan := &UpgradePlan{
		Name:             "v2.5-future-fork",
		Height:           500,
		MinBinaryVersion: "2.5.0",
		ActivationTime:   time.Now(),
		Description:      "Future hard fork",
	}
	manager.RegisterPlan(plan)

	// Block 50 should pass (upgrade height not reached)
	err := manager.CheckBlockUpgrade(50)
	if err != nil {
		t.Fatalf("unexpected error at height 50: %v", err)
	}

	// Block 500 should fail because current running Version is 2.0.0 (< 2.5.0)
	err = manager.CheckBlockUpgrade(500)
	if err == nil {
		t.Fatalf("expected hard fork height 500 to reject out-of-date binary version 2.0.0")
	}
}
