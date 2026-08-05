package agent

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

func TestAgentAccountSessionKeys(t *testing.T) {
	ownerKey, _ := crypto.GenerateKey()
	ownerAddr := crypto.PubkeyToAddress(ownerKey.PublicKey)

	modelHash := crypto.Keccak256Hash([]byte("gpt-4o-litechain-v1"))
	agent := NewAgentAccount("agent-007", modelHash, ownerAddr, uint256.NewInt(1000000000))

	if agent.AgentID != "agent-007" {
		t.Errorf("unexpected agent ID: %s", agent.AgentID)
	}

	subKey, _ := crypto.GenerateKey()
	subKeyAddr := crypto.PubkeyToAddress(subKey.PublicKey)

	targetContract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	allowedMethods := []string{"swap", "transfer"}

	// 1. Register session key with 1000 max spend and 1 hour duration
	agent.RegisterSessionKey(subKeyAddr, uint256.NewInt(1000), 1*time.Hour, []common.Address{targetContract}, allowedMethods)

	// 2. Test authorized execution
	err := agent.AuthorizeExecution(subKeyAddr, targetContract, "swap", uint256.NewInt(500))
	if err != nil {
		t.Fatalf("expected authorization to succeed, got error: %v", err)
	}
	agent.RecordExecution(subKeyAddr, uint256.NewInt(500))

	// 3. Test spending limit breach (spending 600 more exceeds 1000 total)
	err = agent.AuthorizeExecution(subKeyAddr, targetContract, "swap", uint256.NewInt(600))
	if err == nil {
		t.Errorf("expected spending limit breach error, got nil")
	}

	// 4. Test unauthorized contract
	unauthorizedContract := common.HexToAddress("0x9999999999999999999999999999999999999999")
	err = agent.AuthorizeExecution(subKeyAddr, unauthorizedContract, "swap", uint256.NewInt(100))
	if err == nil {
		t.Errorf("expected unauthorized contract error, got nil")
	}

	// 5. Test session key revocation
	agent.RevokeSessionKey(subKeyAddr)
	err = agent.AuthorizeExecution(subKeyAddr, targetContract, "swap", uint256.NewInt(10))
	if err == nil {
		t.Errorf("expected revoked session key error, got nil")
	}
}

func TestVerifiableAIPrecompile(t *testing.T) {
	precompile := &VerifiableAIPrecompile{}

	modelHash := crypto.Keccak256Hash([]byte("llama-3-70b-v1"))
	inputHash := crypto.Keccak256Hash([]byte("analyze_tx_risk"))
	outputHash := crypto.Keccak256Hash([]byte("risk_score_low"))

	// Valid binding proof
	expectedBinding := crypto.Keccak256(append(modelHash.Bytes(), append(inputHash.Bytes(), outputHash.Bytes()...)...))
	proof := append(expectedBinding, []byte("signature_data_bytes")...)

	valid, err := precompile.VerifyAIInferenceProof(modelHash, inputHash, outputHash, proof)
	if err != nil || !valid {
		t.Fatalf("expected AI proof verification to pass, got valid=%v, err=%v", valid, err)
	}

	// Tampered proof
	tamperedProof := make([]byte, len(proof))
	copy(tamperedProof, proof)
	tamperedProof[0] ^= 0xFF

	validTampered, _ := precompile.VerifyAIInferenceProof(modelHash, inputHash, outputHash, tamperedProof)
	if validTampered {
		t.Errorf("expected tampered AI proof verification to fail, got true")
	}
}

func TestAgentMicroPaymentChannel(t *testing.T) {
	senderKey, _ := crypto.GenerateKey()
	senderAddr := crypto.PubkeyToAddress(senderKey.PublicKey)

	recipKey, _ := crypto.GenerateKey()
	recipAddr := crypto.PubkeyToAddress(recipKey.PublicKey)

	channel := NewAgentMicroPaymentChannel(senderAddr, recipAddr, uint256.NewInt(50000))

	// Sender signs voucher for 10000
	voucherAmount := uint256.NewInt(10000)
	sig, err := channel.SignVoucher(voucherAmount, senderKey)
	if err != nil {
		t.Fatalf("failed to sign voucher: %v", err)
	}

	// Receiver verifies voucher off-chain
	valid, err := channel.VerifyVoucher(voucherAmount, sig, &senderKey.PublicKey)
	if err != nil || !valid {
		t.Fatalf("voucher verification failed: %v, valid=%v", err, valid)
	}

	if channel.Transferred.Cmp(voucherAmount) != 0 {
		t.Errorf("channel transferred amount mismatch")
	}
	_ = sha256.Sum256([]byte{})
}
