package version

import (
	"fmt"
	"sync"
	"time"
)

// UpgradePlan defines a scheduled network upgrade / hard fork
type UpgradePlan struct {
	Name             string    `json:"name"`
	Height           uint64    `json:"height"`
	MinBinaryVersion string    `json:"minBinaryVersion"`
	ActivationTime   time.Time `json:"activationTime"`
	Description      string    `json:"description"`
}

// NetworkUpgradeManager handles node upgrade coordination and version enforcement at hard fork heights
type NetworkUpgradeManager struct {
	mu           sync.RWMutex
	plans        map[string]*UpgradePlan
	currentBlock uint64
}

// NewNetworkUpgradeManager creates a new NetworkUpgradeManager
func NewNetworkUpgradeManager() *NetworkUpgradeManager {
	num := &NetworkUpgradeManager{
		plans: make(map[string]*UpgradePlan),
	}

	// Register default v2.0.0 upgrade plan (ZK Groth16, Block-STM, AI Agents, Passkeys)
	num.RegisterPlan(&UpgradePlan{
		Name:             "v2-zk-blockstm-ai-upgrade",
		Height:           100, // Hard fork height
		MinBinaryVersion: "2.0.0",
		ActivationTime:   time.Now().Add(24 * time.Hour),
		Description:      "Upgrade activating Gnark Groth16 ZK-SNARKs, Block-STM parallel execution, AI Agent Accounts, and WebAuthn Passkeys",
	})

	return num
}

// RegisterPlan adds a scheduled upgrade plan
func (m *NetworkUpgradeManager) RegisterPlan(plan *UpgradePlan) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plans[plan.Name] = plan
}

// CheckBlockUpgrade verifies binary version compatibility at block height.
// If the node binary is out-of-date when an upgrade height is reached, returns a fatal upgrade error.
func (m *NetworkUpgradeManager) CheckBlockUpgrade(blockHeight uint64) error {
	m.mu.Lock()
	m.currentBlock = blockHeight
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, plan := range m.plans {
		if blockHeight >= plan.Height {
			// Compare running version against required min binary version
			if CompareVersions(Version, plan.MinBinaryVersion) < 0 {
				return fmt.Errorf("🚨 FATAL HARD FORK UPGRADE ERROR: Block height %d reached scheduled upgrade '%s' requiring binary version >= %s, but running binary is version %s! Please upgrade your node binary immediately",
					blockHeight, plan.Name, plan.MinBinaryVersion, Version)
			}
		}
	}
	return nil
}

// IsUpgradeActive returns true if an upgrade plan is active at the given height
func (m *NetworkUpgradeManager) IsUpgradeActive(planName string, blockHeight uint64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plan, exists := m.plans[planName]
	if !exists {
		return false
	}

	if blockHeight >= plan.Height && CompareVersions(Version, plan.MinBinaryVersion) >= 0 {
		return true
	}
	return false
}
