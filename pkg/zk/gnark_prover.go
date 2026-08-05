package zk

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/sanketsaagar/lightchain-l1/pkg/zk/circuits"
)

// GnarkProver manages real Groth16 ZK-SNARK proving and verification over BN254.
type GnarkProver struct {
	mu sync.RWMutex

	// Transfer circuit keys
	transferR1CS constraint.ConstraintSystem
	transferPK   groth16.ProvingKey
	transferVK   groth16.VerifyingKey

	// Rollup circuit keys
	rollupR1CS constraint.ConstraintSystem
	rollupPK   groth16.ProvingKey
	rollupVK   groth16.VerifyingKey

	// Metrics
	proofsGenerated  uint64
	proofsVerified   uint64
	totalProvingTime time.Duration
}

// NewGnarkProver initializes and performs trusted setup for ZK circuits.
func NewGnarkProver() (*GnarkProver, error) {
	gp := &GnarkProver{}
	if err := gp.setupCircuits(); err != nil {
		return nil, fmt.Errorf("failed to setup gnark circuits: %w", err)
	}
	return gp, nil
}

func (gp *GnarkProver) setupCircuits() error {
	gp.mu.Lock()
	defer gp.mu.Unlock()

	// 1. Setup Private Transfer Circuit
	var transferCircuit circuits.PrivateTransferCircuit
	ccsTransfer, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &transferCircuit)
	if err != nil {
		return fmt.Errorf("failed to compile transfer circuit: %w", err)
	}
	pkTransfer, vkTransfer, err := groth16.Setup(ccsTransfer)
	if err != nil {
		return fmt.Errorf("failed setup for transfer circuit: %w", err)
	}
	gp.transferR1CS = ccsTransfer
	gp.transferPK = pkTransfer
	gp.transferVK = vkTransfer

	// 2. Setup Rollup Batch Circuit
	var rollupCircuit circuits.RollupBatchCircuit
	ccsRollup, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &rollupCircuit)
	if err != nil {
		return fmt.Errorf("failed to compile rollup circuit: %w", err)
	}
	pkRollup, vkRollup, err := groth16.Setup(ccsRollup)
	if err != nil {
		return fmt.Errorf("failed setup for rollup circuit: %w", err)
	}
	gp.rollupR1CS = ccsRollup
	gp.rollupPK = pkRollup
	gp.rollupVK = vkRollup

	return nil
}

// ComputeMiMCHash calculates off-circuit MiMC hash for BN254 (matching gnark inside-circuit mimc).
func ComputeMiMCHash(data ...*big.Int) *big.Int {
	h := mimc.NewMiMC()
	for _, d := range data {
		if d != nil {
			h.Write(d.Bytes())
		}
	}
	sum := h.Sum(nil)
	return new(big.Int).SetBytes(sum)
}

// GenerateTransferProof generates a real Groth16 ZK proof for a private transfer.
func (gp *GnarkProver) GenerateTransferProof(senderKey, recipient, amount, salt *big.Int) ([]byte, []byte, error) {
	start := time.Now()
	gp.mu.RLock()
	defer gp.mu.RUnlock()

	// Calculate expected public nullifier & commitment using MiMC
	nullifierInt := ComputeMiMCHash(senderKey, salt)
	commitmentInt := ComputeMiMCHash(recipient, amount, salt)

	assignment := circuits.PrivateTransferCircuit{
		SenderPrivateKey: senderKey,
		Amount:           amount,
		Salt:             salt,
		Recipient:        recipient,
		Nullifier:        nullifierInt,
		Commitment:       commitmentInt,
	}

	witnessObj, err := frontend.NewWitness(&assignment, ecc.BN254.ScalarField())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create transfer witness: %w", err)
	}

	proof, err := groth16.Prove(gp.transferR1CS, gp.transferPK, witnessObj)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prove transfer circuit: %w", err)
	}

	var proofBuf bytes.Buffer
	if _, err := proof.WriteTo(&proofBuf); err != nil {
		return nil, nil, fmt.Errorf("failed to serialize proof: %w", err)
	}

	pubWitness, err := witnessObj.Public()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract public witness: %w", err)
	}

	var pubBuf bytes.Buffer
	if _, err := pubWitness.WriteTo(&pubBuf); err != nil {
		return nil, nil, fmt.Errorf("failed to serialize public witness: %w", err)
	}

	gp.proofsGenerated++
	gp.totalProvingTime += time.Since(start)

	return proofBuf.Bytes(), pubBuf.Bytes(), nil
}

// VerifyTransferProof verifies a Groth16 ZK proof for a private transfer.
func (gp *GnarkProver) VerifyTransferProof(proofBytes, publicWitnessBytes []byte) (bool, error) {
	gp.mu.RLock()
	defer gp.mu.RUnlock()

	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(proofBytes)); err != nil {
		return false, fmt.Errorf("failed to deserialize proof: %w", err)
	}

	pubWitness, err := witness.New(ecc.BN254.ScalarField())
	if err != nil {
		return false, fmt.Errorf("failed to initialize public witness: %w", err)
	}
	if _, err := pubWitness.ReadFrom(bytes.NewReader(publicWitnessBytes)); err != nil {
		return false, fmt.Errorf("failed to deserialize public witness: %w", err)
	}

	err = groth16.Verify(proof, gp.transferVK, pubWitness)
	if err != nil {
		return false, nil // Invalid proof
	}

	gp.proofsVerified++
	return true, nil
}

// GenerateRollupProof generates a real Groth16 ZK proof for a rollup batch state transition.
func (gp *GnarkProver) GenerateRollupProof(oldStateRoot, batchHash *big.Int) ([]byte, []byte, error) {
	gp.mu.RLock()
	defer gp.mu.RUnlock()

	newStateRoot := ComputeMiMCHash(oldStateRoot, batchHash)

	assignment := circuits.RollupBatchCircuit{
		BatchHash:    batchHash,
		OldStateRoot: oldStateRoot,
		NewStateRoot: newStateRoot,
	}

	witnessObj, err := frontend.NewWitness(&assignment, ecc.BN254.ScalarField())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create rollup witness: %w", err)
	}

	proof, err := groth16.Prove(gp.rollupR1CS, gp.rollupPK, witnessObj)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prove rollup batch: %w", err)
	}

	var proofBuf bytes.Buffer
	if _, err := proof.WriteTo(&proofBuf); err != nil {
		return nil, nil, fmt.Errorf("failed to serialize rollup proof: %w", err)
	}

	pubWitness, err := witnessObj.Public()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract public rollup witness: %w", err)
	}

	var pubBuf bytes.Buffer
	if _, err := pubWitness.WriteTo(&pubBuf); err != nil {
		return nil, nil, fmt.Errorf("failed to serialize public rollup witness: %w", err)
	}

	return proofBuf.Bytes(), pubBuf.Bytes(), nil
}

// VerifyRollupProof verifies a Groth16 ZK proof for a rollup batch.
func (gp *GnarkProver) VerifyRollupProof(proofBytes, publicWitnessBytes []byte) (bool, error) {
	gp.mu.RLock()
	defer gp.mu.RUnlock()

	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(proofBytes)); err != nil {
		return false, fmt.Errorf("failed to deserialize rollup proof: %w", err)
	}

	pubWitness, err := witness.New(ecc.BN254.ScalarField())
	if err != nil {
		return false, fmt.Errorf("failed to initialize rollup public witness: %w", err)
	}
	if _, err := pubWitness.ReadFrom(bytes.NewReader(publicWitnessBytes)); err != nil {
		return false, fmt.Errorf("failed to deserialize rollup public witness: %w", err)
	}

	err = groth16.Verify(proof, gp.rollupVK, pubWitness)
	if err != nil {
		return false, nil
	}

	return true, nil
}
