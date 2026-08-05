package security

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrNullifierAlreadySpent = fmt.Errorf("ZK privacy nullifier has already been spent")
	ErrInvalidChainID        = fmt.Errorf("invalid transaction chain ID replay attempt")
	ErrInvalidNonce          = fmt.Errorf("invalid account transaction nonce")
)

// NullifierRegistry tracks all spent ZK privacy nullifiers persistently to prevent double-spending
type NullifierRegistry struct {
	mu         sync.RWMutex
	nullifiers map[common.Hash]time.Time
}

// NewNullifierRegistry creates a new NullifierRegistry
func NewNullifierRegistry() *NullifierRegistry {
	return &NullifierRegistry{
		nullifiers: make(map[common.Hash]time.Time),
	}
}

// MarkSpent records a ZK nullifier as spent. Returns error if nullifier was previously used.
func (r *NullifierRegistry) MarkSpent(nullifier common.Hash) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if spentAt, exists := r.nullifiers[nullifier]; exists {
		return fmt.Errorf("%w: spent at %v (nullifier: %s)", ErrNullifierAlreadySpent, spentAt, nullifier.Hex())
	}

	r.nullifiers[nullifier] = time.Now()
	return nil
}

// IsSpent returns true if the ZK nullifier has already been recorded as spent
func (r *NullifierRegistry) IsSpent(nullifier common.Hash) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.nullifiers[nullifier]
	return exists
}

// NonceTracker maintains strict account sequence nonces preventing replay and out-of-order attacks
type NonceTracker struct {
	mu     sync.RWMutex
	nonces map[common.Address]uint64
}

// NewNonceTracker creates a new NonceTracker
func NewNonceTracker() *NonceTracker {
	return &NonceTracker{
		nonces: make(map[common.Address]uint64),
	}
}

// ValidateAndIncrement verifies expected nonce matches and increments upon success
func (t *NonceTracker) ValidateAndIncrement(account common.Address, txNonce uint64) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	currentNonce := t.nonces[account]
	if txNonce != currentNonce {
		return fmt.Errorf("%w: expected %d, got %d for account %s", ErrInvalidNonce, currentNonce, txNonce, account.Hex())
	}

	t.nonces[account] = currentNonce + 1
	return nil
}

// GetNonce returns current nonce for an account
func (t *NonceTracker) GetNonce(account common.Address) uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.nonces[account]
}

// ReplayGuard enforces EIP-155 Chain-ID verification across all operations
type ReplayGuard struct {
	expectedChainID *big.Int
}

// NewReplayGuard creates a ReplayGuard for the chain
func NewReplayGuard(chainID *big.Int) *ReplayGuard {
	return &ReplayGuard{
		expectedChainID: chainID,
	}
}

// ValidateChainID verifies that the operation's Chain ID matches expected network chain ID
func (g *ReplayGuard) ValidateChainID(txChainID *big.Int) error {
	if txChainID == nil || g.expectedChainID == nil {
		return ErrInvalidChainID
	}
	if txChainID.Cmp(g.expectedChainID) != 0 {
		return fmt.Errorf("%w: expected chain ID %s, got %s", ErrInvalidChainID, g.expectedChainID.String(), txChainID.String())
	}
	return nil
}

// HashOperation constructs a domain-separated hash incorporating Chain ID
func (g *ReplayGuard) HashOperation(opData []byte) common.Hash {
	payload := append(g.expectedChainID.Bytes(), opData...)
	return crypto.Keccak256Hash(payload)
}
