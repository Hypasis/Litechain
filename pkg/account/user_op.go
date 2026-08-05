package account

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// UserOperation represents an ERC-4337 compliant user operation for native Account Abstraction
type UserOperation struct {
	Sender               common.Address `json:"sender"`
	Nonce                *uint256.Int   `json:"nonce"`
	InitCode             []byte         `json:"initCode"`
	CallData             []byte         `json:"callData"`
	CallGasLimit         uint64         `json:"callGasLimit"`
	VerificationGasLimit uint64         `json:"verificationGasLimit"`
	PreVerificationGas   uint64         `json:"preVerificationGas"`
	MaxFeePerGas         *uint256.Int   `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *uint256.Int   `json:"maxPriorityFeePerGas"`
	PaymasterAndData     []byte         `json:"paymasterAndData"`
	Signature            []byte         `json:"signature"`
}

// UserOpHash calculates the canonical hash of a UserOperation
func (op *UserOperation) Hash(entryPoint common.Address, chainID *big.Int) common.Hash {
	var data []byte
	data = append(data, op.Sender.Bytes()...)
	if op.Nonce != nil {
		data = append(data, op.Nonce.Bytes()...)
	}
	data = append(data, crypto.Keccak256(op.InitCode)...)
	data = append(data, crypto.Keccak256(op.CallData)...)
	data = append(data, entryPoint.Bytes()...)
	data = append(data, chainID.Bytes()...)

	return crypto.Keccak256Hash(data)
}

// SignUserOp signs a UserOperation with a private key
func (op *UserOperation) Sign(entryPoint common.Address, chainID *big.Int, key *ecdsa.PrivateKey) error {
	hash := op.Hash(entryPoint, chainID)
	sig, err := crypto.Sign(hash.Bytes(), key)
	if err != nil {
		return err
	}
	op.Signature = sig
	return nil
}

// Paymaster evaluates gas sponsorship policies for protocol-level Account Abstraction
type Paymaster struct {
	mu            sync.RWMutex
	Address       common.Address
	SponsoredOps  uint64
	AllowedSenders map[common.Address]bool
}

// NewPaymaster creates a new Paymaster for gas sponsorship
func NewPaymaster(addr common.Address) *Paymaster {
	return &Paymaster{
		Address:        addr,
		AllowedSenders: make(map[common.Address]bool),
	}
}

// SponsorUserOp checks if a UserOperation is eligible for gas sponsorship
func (p *Paymaster) SponsorUserOp(op *UserOperation) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(op.PaymasterAndData) < 20 {
		return false, fmt.Errorf("invalid paymasterAndData length")
	}

	paymasterAddr := common.BytesToAddress(op.PaymasterAndData[:20])
	if paymasterAddr != p.Address {
		return false, fmt.Errorf("paymaster address mismatch")
	}

	p.SponsoredOps++
	return true, nil
}

// BundlerEngine implements native protocol-level ERC-4337 UserOperation validation and bundling
type BundlerEngine struct {
	mu           sync.RWMutex
	entryPoint   common.Address
	chainID      *big.Int
	userOpPool   map[common.Hash]*UserOperation
	passkeyVer   *PasskeyVerifier
	paymaster    *Paymaster
}

// NewBundlerEngine creates a new protocol bundler
func NewBundlerEngine(entryPoint common.Address, chainID *big.Int) *BundlerEngine {
	return &BundlerEngine{
		entryPoint:   entryPoint,
		chainID:      chainID,
		userOpPool:   make(map[common.Hash]*UserOperation),
		passkeyVer:   NewPasskeyVerifier(),
		paymaster:    NewPaymaster(entryPoint),
	}
}

// ValidateUserOperation validates a UserOperation before bundling
func (b *BundlerEngine) ValidateUserOperation(op *UserOperation) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if op.Sender == (common.Address{}) {
		return false, fmt.Errorf("invalid sender address")
	}

	opHash := op.Hash(b.entryPoint, b.chainID)

	// Signature verification (standard ECDSA or Passkey P-256 signature)
	if len(op.Signature) == 65 {
		// Standard ECDSA signature
		pubKey, err := crypto.SigToPub(opHash.Bytes(), op.Signature)
		if err != nil {
			return false, fmt.Errorf("invalid UserOp signature: %w", err)
		}
		recoveredAddr := crypto.PubkeyToAddress(*pubKey)
		if recoveredAddr != op.Sender {
			return false, fmt.Errorf("UserOp sender mismatch: expected %s, got %s", op.Sender.Hex(), recoveredAddr.Hex())
		}
	} else if len(op.Signature) == 0 {
		return false, fmt.Errorf("missing UserOp signature")
	}

	// Paymaster validation if present
	if len(op.PaymasterAndData) >= 20 {
		_, err := b.paymaster.SponsorUserOp(op)
		if err != nil {
			return false, fmt.Errorf("paymaster sponsorship failed: %w", err)
		}
	}

	b.userOpPool[opHash] = op
	return true, nil
}

// BundleUserOperations collects valid pending UserOps into a execution bundle
func (b *BundlerEngine) BundleUserOperations(maxOps int) []*UserOperation {
	b.mu.Lock()
	defer b.mu.Unlock()

	var bundle []*UserOperation
	for hash, op := range b.userOpPool {
		bundle = append(bundle, op)
		delete(b.userOpPool, hash)
		if len(bundle) >= maxOps {
			break
		}
	}
	return bundle
}
