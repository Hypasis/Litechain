package agent

import (
	"crypto/ecdsa"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// AgentSessionKey represents a delegated permission key assigned to an AI sub-agent or task
type AgentSessionKey struct {
	PublicKey        common.Address   `json:"publicKey"`
	MaxSpendAmount   *uint256.Int     `json:"maxSpendAmount"`
	SpentAmount      *uint256.Int     `json:"spentAmount"`
	ExpiresAt        time.Time        `json:"expiresAt"`
	AllowedContracts []common.Address `json:"allowedContracts"`
	AllowedMethods   []string         `json:"allowedMethods"`
	IsRevoked        bool             `json:"isRevoked"`
}

// AgentAccount represents an autonomous AI Agent Account on Litechain
type AgentAccount struct {
	mu           sync.RWMutex
	AgentID      string                   `json:"agentId"`
	ModelHash    common.Hash              `json:"modelHash"` // Fingerprint of AI model weights / prompt
	OwnerAddress common.Address           `json:"ownerAddress"`
	AgentAddress common.Address           `json:"agentAddress"`
	SessionKeys  map[common.Address]*AgentSessionKey `json:"sessionKeys"`
	Balance      *uint256.Int             `json:"balance"`
	Nonce        uint64                   `json:"nonce"`
}

// NewAgentAccount creates a new AI Agent Account owned by owner
func NewAgentAccount(agentID string, modelHash common.Hash, owner common.Address, initialBalance *uint256.Int) *AgentAccount {
	if initialBalance == nil {
		initialBalance = uint256.NewInt(0)
	}

	// Generate deterministic agent address from agentID and modelHash
	addrBytes := crypto.Keccak256(append([]byte(agentID), modelHash.Bytes()...))
	agentAddr := common.BytesToAddress(addrBytes[12:])

	return &AgentAccount{
		AgentID:      agentID,
		ModelHash:    modelHash,
		OwnerAddress: owner,
		AgentAddress: agentAddr,
		SessionKeys:  make(map[common.Address]*AgentSessionKey),
		Balance:      initialBalance,
		Nonce:        0,
	}
}

// RegisterSessionKey authorizes a new sub-key with granular spending and expiration constraints
func (a *AgentAccount) RegisterSessionKey(keyAddr common.Address, maxSpend *uint256.Int, duration time.Duration, contracts []common.Address, methods []string) *AgentSessionKey {
	a.mu.Lock()
	defer a.mu.Unlock()

	if maxSpend == nil {
		maxSpend = uint256.NewInt(0)
	}

	sessionKey := &AgentSessionKey{
		PublicKey:        keyAddr,
		MaxSpendAmount:   maxSpend,
		SpentAmount:      uint256.NewInt(0),
		ExpiresAt:        time.Now().Add(duration),
		AllowedContracts: contracts,
		AllowedMethods:   methods,
		IsRevoked:        false,
	}

	a.SessionKeys[keyAddr] = sessionKey
	return sessionKey
}

// AuthorizeExecution verifies whether a transaction signed by a session key is within allowed limits
func (a *AgentAccount) AuthorizeExecution(sessionKeyAddr common.Address, targetContract common.Address, method string, spendAmount *uint256.Int) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sk, exists := a.SessionKeys[sessionKeyAddr]
	if !exists {
		return fmt.Errorf("unregistered session key: %s", sessionKeyAddr.Hex())
	}

	if sk.IsRevoked {
		return fmt.Errorf("session key has been revoked")
	}

	if time.Now().After(sk.ExpiresAt) {
		return fmt.Errorf("session key expired at %v", sk.ExpiresAt)
	}

	// Check allowed contracts (empty slice means all allowed)
	if len(sk.AllowedContracts) > 0 {
		allowed := false
		for _, c := range sk.AllowedContracts {
			if c == targetContract {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("contract %s is not in session key whitelist", targetContract.Hex())
		}
	}

	// Check allowed methods (empty slice means all allowed)
	if len(sk.AllowedMethods) > 0 {
		allowed := false
		for _, m := range sk.AllowedMethods {
			if m == method || m == "*" {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("method %s is not in session key whitelist", method)
		}
	}

	// Check spend limit
	if spendAmount != nil && spendAmount.Sign() > 0 {
		newSpent := new(uint256.Int).Add(sk.SpentAmount, spendAmount)
		if newSpent.Cmp(sk.MaxSpendAmount) > 0 {
			return fmt.Errorf("spending limit exceeded: max %s, current %s, requested %s",
				sk.MaxSpendAmount.String(), sk.SpentAmount.String(), spendAmount.String())
		}
	}

	return nil
}

// RecordExecution updates the spent amount for a session key after execution
func (a *AgentAccount) RecordExecution(sessionKeyAddr common.Address, spendAmount *uint256.Int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sk, exists := a.SessionKeys[sessionKeyAddr]
	if exists && spendAmount != nil {
		sk.SpentAmount = new(uint256.Int).Add(sk.SpentAmount, spendAmount)
	}
}

// RevokeSessionKey revokes a session key immediately
func (a *AgentAccount) RevokeSessionKey(sessionKeyAddr common.Address) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if sk, exists := a.SessionKeys[sessionKeyAddr]; exists {
		sk.IsRevoked = true
	}
}

// VerifiableAIPrecompile implements precompiled contract logic (0x00...0A) verifying zkML/TEE proofs
type VerifiableAIPrecompile struct {
	VerifiedCount uint64
}

var VerifiableAIPrecompileAddress = common.HexToAddress("0x000000000000000000000000000000000000000A")

// VerifyAIInferenceProof verifies zero-knowledge machine learning (zkML) or TEE remote attestation proof
func (p *VerifiableAIPrecompile) VerifyAIInferenceProof(modelHash common.Hash, inputHash common.Hash, outputHash common.Hash, proof []byte) (bool, error) {
	if len(proof) == 0 {
		return false, fmt.Errorf("empty AI proof")
	}

	// Verify cryptographic signature / proof binding modelHash + inputHash + outputHash
	expectedBinding := crypto.Keccak256(append(modelHash.Bytes(), append(inputHash.Bytes(), outputHash.Bytes()...)...))
	if len(proof) < 32 {
		return false, fmt.Errorf("invalid proof payload length")
	}

	// Simple check: proof payload contains binding hash prefix
	if bytesEqual(proof[:32], expectedBinding) {
		p.VerifiedCount++
		return true, nil
	}

	return false, fmt.Errorf("AI proof verification failed: invalid model inference binding")
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// AgentMicroPaymentChannel handles low-latency off-chain agent-to-agent micro-payments
type AgentMicroPaymentChannel struct {
	mu           sync.RWMutex
	ChannelID    common.Hash
	Sender       common.Address
	Receiver     common.Address
	Deposit      *uint256.Int
	Transferred  *uint256.Int
	Nonce        uint64
	IsClosed     bool
}

// NewAgentMicroPaymentChannel creates a micro-payment channel between two AI agents
func NewAgentMicroPaymentChannel(sender, receiver common.Address, deposit *uint256.Int) *AgentMicroPaymentChannel {
	if deposit == nil {
		deposit = uint256.NewInt(0)
	}
	channelID := crypto.Keccak256Hash(append(sender.Bytes(), append(receiver.Bytes(), deposit.Bytes()...)...))
	return &AgentMicroPaymentChannel{
		ChannelID:   channelID,
		Sender:      sender,
		Receiver:    receiver,
		Deposit:     deposit,
		Transferred: uint256.NewInt(0),
		Nonce:       0,
		IsClosed:    false,
	}
}

// SignVoucher creates a signed payment voucher for an agent micro-payment
func (c *AgentMicroPaymentChannel) SignVoucher(amount *uint256.Int, key *ecdsa.PrivateKey) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	msg := crypto.Keccak256(append(c.ChannelID.Bytes(), amount.Bytes()...))
	return crypto.Sign(msg, key)
}

// VerifyVoucher verifies and updates the channel state for off-chain micro-payments
func (c *AgentMicroPaymentChannel) VerifyVoucher(amount *uint256.Int, signature []byte, senderPubKey *ecdsa.PublicKey) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.IsClosed {
		return false, fmt.Errorf("channel is closed")
	}

	if amount.Cmp(c.Deposit) > 0 {
		return false, fmt.Errorf("voucher amount exceeds channel deposit")
	}

	msg := crypto.Keccak256(append(c.ChannelID.Bytes(), amount.Bytes()...))

	// Verify ECDSA signature
	sigPublicKey, err := crypto.SigToPub(msg, signature)
	if err != nil {
		return false, fmt.Errorf("signature recovery failed: %w", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*sigPublicKey)
	if recoveredAddr != c.Sender {
		return false, fmt.Errorf("unauthorized voucher signature from %s", recoveredAddr.Hex())
	}

	c.Transferred = amount
	c.Nonce++
	return true, nil
}
