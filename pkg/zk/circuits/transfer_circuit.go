package circuits

import (
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
)

// PrivateTransferCircuit defines a zero-knowledge circuit for private transactions.
// It proves knowledge of secret parameters (sender key, amount, salt, recipient)
// such that Nullifier = MiMC(SenderKey, Salt) and Commitment = MiMC(Recipient, Amount, Salt).
type PrivateTransferCircuit struct {
	// Secret inputs (witness)
	SenderPrivateKey frontend.Variable `gnark:",secret"`
	Amount           frontend.Variable `gnark:",secret"`
	Salt             frontend.Variable `gnark:",secret"`
	Recipient        frontend.Variable `gnark:",secret"`

	// Public inputs
	Nullifier  frontend.Variable `gnark:",public"`
	Commitment frontend.Variable `gnark:",public"`
}

// Define sets up constraints for PrivateTransferCircuit
func (c *PrivateTransferCircuit) Define(api frontend.API) error {
	// Nullifier constraint: Nullifier == MiMC(SenderPrivateKey, Salt)
	hNullifier, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}
	hNullifier.Write(c.SenderPrivateKey)
	hNullifier.Write(c.Salt)
	api.AssertIsEqual(c.Nullifier, hNullifier.Sum())

	// Commitment constraint: Commitment == MiMC(Recipient, Amount, Salt)
	hCommitment, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}
	hCommitment.Write(c.Recipient)
	hCommitment.Write(c.Amount)
	hCommitment.Write(c.Salt)
	api.AssertIsEqual(c.Commitment, hCommitment.Sum())

	return nil
}
