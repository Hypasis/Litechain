package circuits

import (
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
)

// RollupBatchCircuit defines a ZK circuit verifying state transition integrity.
// It proves NewStateRoot == MiMC(OldStateRoot, BatchHash).
type RollupBatchCircuit struct {
	// Secret inputs (witness)
	BatchHash frontend.Variable `gnark:",secret"`

	// Public inputs
	OldStateRoot frontend.Variable `gnark:",public"`
	NewStateRoot frontend.Variable `gnark:",public"`
}

// Define sets up constraints for RollupBatchCircuit
func (c *RollupBatchCircuit) Define(api frontend.API) error {
	hState, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}
	hState.Write(c.OldStateRoot)
	hState.Write(c.BatchHash)
	api.AssertIsEqual(c.NewStateRoot, hState.Sum())

	return nil
}
