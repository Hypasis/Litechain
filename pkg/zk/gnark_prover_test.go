package zk

import (
	"math/big"
	"testing"
)

func TestGnarkProverInitialization(t *testing.T) {
	prover, err := NewGnarkProver()
	if err != nil {
		t.Fatalf("NewGnarkProver failed: %v", err)
	}
	if prover == nil {
		t.Fatalf("expected non-nil GnarkProver")
	}
	if prover.transferPK == nil || prover.transferVK == nil {
		t.Errorf("transfer keys not initialized")
	}
	if prover.rollupPK == nil || prover.rollupVK == nil {
		t.Errorf("rollup keys not initialized")
	}
}

func TestPrivateTransferCircuit(t *testing.T) {
	prover, err := NewGnarkProver()
	if err != nil {
		t.Fatalf("NewGnarkProver failed: %v", err)
	}

	senderKey := big.NewInt(123456789)
	recipient := big.NewInt(987654321)
	amount := big.NewInt(5000)
	salt := big.NewInt(42)

	// 1. Generate real ZK-SNARK proof
	proofBytes, pubWitnessBytes, err := prover.GenerateTransferProof(senderKey, recipient, amount, salt)
	if err != nil {
		t.Fatalf("GenerateTransferProof failed: %v", err)
	}

	if len(proofBytes) == 0 || len(pubWitnessBytes) == 0 {
		t.Fatalf("expected non-empty proof and public witness bytes")
	}

	// 2. Verify valid proof
	valid, err := prover.VerifyTransferProof(proofBytes, pubWitnessBytes)
	if err != nil {
		t.Fatalf("VerifyTransferProof returned error: %v", err)
	}
	if !valid {
		t.Errorf("expected proof verification to succeed, got false")
	}

	// 3. Verify tampered proof fails
	tamperedProof := make([]byte, len(proofBytes))
	copy(tamperedProof, proofBytes)
	tamperedProof[0] ^= 0xFF

	tamperedValid, _ := prover.VerifyTransferProof(tamperedProof, pubWitnessBytes)
	if tamperedValid {
		t.Errorf("expected tampered proof verification to fail, got true")
	}
}

func TestRollupBatchCircuit(t *testing.T) {
	prover, err := NewGnarkProver()
	if err != nil {
		t.Fatalf("NewGnarkProver failed: %v", err)
	}

	oldStateRoot := big.NewInt(1001)
	batchHash := big.NewInt(2002)

	// 1. Generate ZK-SNARK batch proof
	proofBytes, pubWitnessBytes, err := prover.GenerateRollupProof(oldStateRoot, batchHash)
	if err != nil {
		t.Fatalf("GenerateRollupProof failed: %v", err)
	}

	// 2. Verify valid batch proof
	valid, err := prover.VerifyRollupProof(proofBytes, pubWitnessBytes)
	if err != nil {
		t.Fatalf("VerifyRollupProof returned error: %v", err)
	}
	if !valid {
		t.Errorf("expected rollup batch proof verification to succeed, got false")
	}
}

func TestZKEngineGnarkIntegration(t *testing.T) {
	engine := NewZKEngine(nil)
	if engine == nil {
		t.Fatalf("expected non-nil ZKEngine")
	}

	gnark := engine.GetGnarkProver()
	if gnark == nil {
		t.Fatalf("expected non-nil GnarkProver from engine")
	}
}
