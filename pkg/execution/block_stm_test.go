package execution

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

func generateSignedTx(t *testing.T, key *ecdsa.PrivateKey, to common.Address, amount *big.Int, nonce uint64, chainID *big.Int) *types.Transaction {
	txData := &types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    amount,
		Gas:      21000,
		GasPrice: big.NewInt(1000000000),
		Data:     nil,
	}
	tx := types.NewTx(txData)
	signer := types.NewEIP155Signer(chainID)
	signedTx, err := types.SignTx(tx, signer, key)
	if err != nil {
		t.Fatalf("failed to sign tx: %v", err)
	}
	return signedTx
}

func TestBlockSTMInitialization(t *testing.T) {
	chainID := big.NewInt(1337)
	executor := NewBlockSTMExecutor(chainID, 4)
	if executor == nil {
		t.Fatalf("expected non-nil BlockSTMExecutor")
	}
	if executor.mvMemory == nil {
		t.Errorf("mvMemory not initialized")
	}
}

func TestIndependentParallelTransactions(t *testing.T) {
	chainID := big.NewInt(1337)
	executor := NewBlockSTMExecutor(chainID, 8)

	initialBalances := make(map[common.Address]*uint256.Int)
	var txs []*types.Transaction

	// Generate 100 independent transfers between 100 unique keypairs
	for i := 0; i < 100; i++ {
		keyFrom, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		fromAddr := crypto.PubkeyToAddress(keyFrom.PublicKey)

		keyTo, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("failed to generate recipient key: %v", err)
		}
		toAddr := crypto.PubkeyToAddress(keyTo.PublicKey)

		initialBalances[fromAddr] = uint256.NewInt(1000000000000000000) // 1 ETH
		initialBalances[toAddr] = uint256.NewInt(0)

		tx := generateSignedTx(t, keyFrom, toAddr, big.NewInt(10000), 0, chainID)
		txs = append(txs, tx)
	}

	receipts, gasUsed, conflicts, duration, err := executor.ExecuteBlockSTM(txs, initialBalances)
	if err != nil {
		t.Fatalf("ExecuteBlockSTM failed: %v", err)
	}

	if len(receipts) != 100 {
		t.Errorf("expected 100 receipts, got %d", len(receipts))
	}
	if gasUsed != 100*21000 {
		t.Errorf("expected gasUsed = %d, got %d", 100*21000, gasUsed)
	}

	t.Logf("⚡ Block-STM Independent execution: 100 txs in %v (conflicts: %d, TPS: %.2f)",
		duration, conflicts, float64(100)/duration.Seconds())
}

func TestDependentTransactionCascade(t *testing.T) {
	chainID := big.NewInt(1337)
	executor := NewBlockSTMExecutor(chainID, 4)

	keyA, _ := crypto.GenerateKey()
	addrA := crypto.PubkeyToAddress(keyA.PublicKey)

	keyB, _ := crypto.GenerateKey()
	addrB := crypto.PubkeyToAddress(keyB.PublicKey)

	keyC, _ := crypto.GenerateKey()
	addrC := crypto.PubkeyToAddress(keyC.PublicKey)

	initialBalances := map[common.Address]*uint256.Int{
		addrA: uint256.NewInt(1000000),
		addrB: uint256.NewInt(0),
		addrC: uint256.NewInt(0),
	}

	// Tx0: A -> B (50000)
	tx0 := generateSignedTx(t, keyA, addrB, big.NewInt(50000), 0, chainID)

	// Tx1: B -> C (20000) - Depends on Tx0 modifying B's balance!
	tx1 := generateSignedTx(t, keyB, addrC, big.NewInt(20000), 0, chainID)

	txs := []*types.Transaction{tx0, tx1}

	receipts, _, _, _, err := executor.ExecuteBlockSTM(txs, initialBalances)
	if err != nil {
		t.Fatalf("ExecuteBlockSTM failed: %v", err)
	}

	if len(receipts) != 2 {
		t.Fatalf("expected 2 receipts, got %d", len(receipts))
	}
	for i, r := range receipts {
		if r.Status != types.ReceiptStatusSuccessful {
			t.Errorf("receipt %d expected success, got %d", i, r.Status)
		}
	}
}

func TestParallelExecutorBlockSTMIntegration(t *testing.T) {
	chainID := big.NewInt(1337)
	pe := NewParallelExecutor(chainID, nil)

	if pe.GetBlockSTM() == nil {
		t.Fatalf("expected non-nil BlockSTM instance")
	}

	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	to, _ := crypto.GenerateKey()
	toAddr := crypto.PubkeyToAddress(to.PublicKey)

	tx := generateSignedTx(t, key, toAddr, big.NewInt(1000), 0, chainID)
	initialBalances := map[common.Address]*uint256.Int{
		addr: uint256.NewInt(50000),
	}

	receipts, gas, _, _, err := pe.ExecuteBlockSTM([]*types.Transaction{tx}, initialBalances)
	if err != nil {
		t.Fatalf("ExecuteBlockSTM via ParallelExecutor failed: %v", err)
	}
	if len(receipts) != 1 || gas != 21000 {
		t.Errorf("unexpected execution result: receipts=%d gas=%d", len(receipts), gas)
	}
}
