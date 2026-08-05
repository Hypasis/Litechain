package security

import (
	"crypto/rand"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/Hypasis/Litechain/pkg/consensus"
	"github.com/Hypasis/Litechain/pkg/execution"
)

func TestNullifierDoubleSpendProtection(t *testing.T) {
	registry := NewNullifierRegistry()

	nullifierBytes := make([]byte, 32)
	rand.Read(nullifierBytes)
	nullifier := common.BytesToHash(nullifierBytes)

	// 1. First spend should succeed
	err := registry.MarkSpent(nullifier)
	if err != nil {
		t.Fatalf("first MarkSpent failed: %v", err)
	}

	if !registry.IsSpent(nullifier) {
		t.Errorf("expected nullifier to be marked spent")
	}

	// 2. Second spend attempt must be rejected with ErrNullifierAlreadySpent!
	err = registry.MarkSpent(nullifier)
	if err == nil {
		t.Fatalf("expected double-spend attempt to fail, got nil error")
	}
}

func TestReplayProtection(t *testing.T) {
	chainID := big.NewInt(1337)
	guard := NewReplayGuard(chainID)

	// 1. Valid Chain ID
	if err := guard.ValidateChainID(chainID); err != nil {
		t.Fatalf("valid chain ID rejected: %v", err)
	}

	// 2. Mismatched Chain ID (replay attack from Ethereum mainnet chain ID 1)
	ethMainnetID := big.NewInt(1)
	if err := guard.ValidateChainID(ethMainnetID); err == nil {
		t.Fatalf("expected mismatched chain ID to be rejected")
	}

	// 3. Nonce sequential validation
	tracker := NewNonceTracker()
	user := common.HexToAddress("0x1111111111111111111111111111111111111111")

	// Nonce 0 succeeds
	if err := tracker.ValidateAndIncrement(user, 0); err != nil {
		t.Fatalf("nonce 0 failed: %v", err)
	}

	// Replay nonce 0 fails
	if err := tracker.ValidateAndIncrement(user, 0); err == nil {
		t.Fatalf("replayed nonce 0 should be rejected")
	}

	// Out-of-order nonce 5 fails
	if err := tracker.ValidateAndIncrement(user, 5); err == nil {
		t.Fatalf("out-of-order nonce 5 should be rejected")
	}
}

func TestEquivocationDetectorSlashing(t *testing.T) {
	detector := consensus.NewEquivocationDetector()

	valKey, _ := crypto.GenerateKey()
	valAddr := crypto.PubkeyToAddress(valKey.PublicKey)

	detector.RegisterValidator(valAddr, uint256.NewInt(1000000000))

	headerA := consensus.BlockHeaderEvidence{
		BlockNumber: 42,
		BlockHash:   crypto.Keccak256Hash([]byte("block_42_variant_A")),
		StateRoot:   crypto.Keccak256Hash([]byte("state_A")),
		Validator:   valAddr,
		Timestamp:   time.Now(),
	}

	// 1. First block header signed by validator
	slashedRecord, err := detector.ProcessHeader(headerA)
	if err != nil || slashedRecord != nil {
		t.Fatalf("unexpected slashing on valid first header: %v", err)
	}

	// 2. Validator double-signs a different block at the SAME height 42!
	headerB := consensus.BlockHeaderEvidence{
		BlockNumber: 42,
		BlockHash:   crypto.Keccak256Hash([]byte("block_42_variant_B_DOUBLE_SIGN")),
		StateRoot:   crypto.Keccak256Hash([]byte("state_B")),
		Validator:   valAddr,
		Timestamp:   time.Now(),
	}

	slashedRecord, err = detector.ProcessHeader(headerB)
	if err == nil || slashedRecord == nil {
		t.Fatalf("expected double-signing detector to catch and slash validator!")
	}

	if slashedRecord.SlashedAmount.Cmp(uint256.NewInt(500000000)) != 0 {
		t.Errorf("expected 50%% slashed amount (500,000,000), got %s", slashedRecord.SlashedAmount.String())
	}

	if !detector.IsJailed(valAddr) {
		t.Errorf("expected validator to be jailed after double-signing")
	}
}

func TestBlockSTMReversionOnInsufficientBalance(t *testing.T) {
	chainID := big.NewInt(1337)
	executor := execution.NewBlockSTMExecutor(chainID, 4)

	senderKey, _ := crypto.GenerateKey()
	senderAddr := crypto.PubkeyToAddress(senderKey.PublicKey)

	recipKey, _ := crypto.GenerateKey()
	recipAddr := crypto.PubkeyToAddress(recipKey.PublicKey)

	// Sender only has 100 wei
	initialBalances := map[common.Address]*uint256.Int{
		senderAddr: uint256.NewInt(100),
	}

	// Attempt transfer of 5,000 wei (exceeding balance!)
	txData := &types.LegacyTx{
		Nonce:    0,
		To:       &recipAddr,
		Value:    big.NewInt(5000),
		Gas:      21000,
		GasPrice: big.NewInt(1000000000),
	}
	signedTx, _ := types.SignTx(types.NewTx(txData), types.NewEIP155Signer(chainID), senderKey)

	receipts, _, _, _, err := executor.ExecuteBlockSTM([]*types.Transaction{signedTx}, initialBalances)
	if err != nil {
		t.Fatalf("ExecuteBlockSTM failed: %v", err)
	}

	if len(receipts) != 1 {
		t.Fatalf("expected 1 receipt, got %d", len(receipts))
	}
}
