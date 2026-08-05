package consensus

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// BlockHeaderEvidence represents a signed block header for double-signing detection
type BlockHeaderEvidence struct {
	BlockNumber uint64         `json:"blockNumber"`
	BlockHash   common.Hash    `json:"blockHash"`
	StateRoot   common.Hash    `json:"stateRoot"`
	Validator   common.Address `json:"validator"`
	Signature   []byte         `json:"signature"`
	Timestamp   time.Time      `json:"timestamp"`
}

// EquivocationEvidence contains proof that a validator signed two distinct blocks at the same height
type EquivocationEvidence struct {
	Validator common.Address      `json:"validator"`
	Height    uint64              `json:"height"`
	HeaderA   BlockHeaderEvidence `json:"headerA"`
	HeaderB   BlockHeaderEvidence `json:"headerB"`
}

// SlashingRecord documents a validator slashing penalty
type SlashingRecord struct {
	Validator      common.Address `json:"validator"`
	Reason         string         `json:"reason"`
	SlashedAmount  *uint256.Int   `json:"slashedAmount"`
	SlashedAt      time.Time      `json:"slashedAt"`
	JailedUntil    time.Time      `json:"jailedUntil"`
}

// EquivocationDetector monitors validator block signatures to detect double-signing
type EquivocationDetector struct {
	mu            sync.RWMutex
	signedHeaders map[uint64]map[common.Address]BlockHeaderEvidence
	slashed       map[common.Address]*SlashingRecord
	validatorSet  map[common.Address]*uint256.Int // Validator -> Staked amount
}

// NewEquivocationDetector creates a new EquivocationDetector
func NewEquivocationDetector() *EquivocationDetector {
	return &EquivocationDetector{
		signedHeaders: make(map[uint64]map[common.Address]BlockHeaderEvidence),
		slashed:       make(map[common.Address]*SlashingRecord),
		validatorSet:  make(map[common.Address]*uint256.Int),
	}
}

// RegisterValidator registers a validator with staked amount
func (d *EquivocationDetector) RegisterValidator(val common.Address, stake *uint256.Int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if stake == nil {
		stake = uint256.NewInt(0)
	}
	d.validatorSet[val] = stake
}

// ProcessHeader inspects a proposed block header. Returns slashing record if double-signing is detected.
func (d *EquivocationDetector) ProcessHeader(header BlockHeaderEvidence) (*SlashingRecord, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if already slashed/jailed
	if record, exists := d.slashed[header.Validator]; exists {
		return record, fmt.Errorf("validator %s is jailed until %v", header.Validator.Hex(), record.JailedUntil)
	}

	heightMap, exists := d.signedHeaders[header.BlockNumber]
	if !exists {
		heightMap = make(map[common.Address]BlockHeaderEvidence)
		d.signedHeaders[header.BlockNumber] = heightMap
	}

	existingHeader, alreadySigned := heightMap[header.Validator]
	if alreadySigned {
		if existingHeader.BlockHash != header.BlockHash {
			// DOUBLE-SIGNING DETECTED! Validator signed 2 different blocks at same height.
			stake := d.validatorSet[header.Validator]
			if stake == nil {
				stake = uint256.NewInt(1000000000)
			}

			// Slash 50% of stake
			slashedAmount := new(uint256.Int).Div(stake, uint256.NewInt(2))
			record := &SlashingRecord{
				Validator:     header.Validator,
				Reason:        fmt.Sprintf("Equivocation (double-signing block #%d)", header.BlockNumber),
				SlashedAmount: slashedAmount,
				SlashedAt:     time.Now(),
				JailedUntil:   time.Now().Add(30 * 24 * time.Hour), // Jailed for 30 days
			}

			d.slashed[header.Validator] = record
			delete(d.validatorSet, header.Validator)

			return record, fmt.Errorf("🚨 SECURITY ALERT: Double-signing detected for validator %s at height %d! Validator slashed %s tokens and jailed",
				header.Validator.Hex(), header.BlockNumber, slashedAmount.String())
		}
		return nil, nil // Same block header, harmless duplicate broadcast
	}

	heightMap[header.Validator] = header
	return nil, nil
}

// IsJailed returns true if validator is currently jailed for equivocation
func (d *EquivocationDetector) IsJailed(val common.Address) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	record, exists := d.slashed[val]
	if !exists {
		return false
	}
	return time.Now().Before(record.JailedUntil)
}

// SignHeader creates a valid header signature using validator private key
func SignHeader(header *BlockHeaderEvidence, key *big.Int) {
	msg := crypto.Keccak256(append(header.BlockHash.Bytes(), header.StateRoot.Bytes()...))
	privKey, _ := crypto.ToECDSA(key.Bytes())
	sig, _ := crypto.Sign(msg, privKey)
	header.Signature = sig
}
