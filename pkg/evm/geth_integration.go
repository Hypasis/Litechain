package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// EVMExecutor provides full Ethereum Virtual Machine compatibility
type EVMExecutor struct {
	blockchain *core.BlockChain
	stateDB    *state.StateDB
	vmConfig   vm.Config
	chainID    *big.Int
}

// NewEVMExecutor creates a new EVM-compatible execution engine
func NewEVMExecutor(chainID *big.Int, stateDB *state.StateDB) *EVMExecutor {
	return &EVMExecutor{
		chainID:  chainID,
		stateDB:  stateDB,
		vmConfig: vm.Config{},
	}
}

// ExecuteTransaction processes a transaction through the EVM
func (e *EVMExecutor) ExecuteTransaction(tx *types.Transaction, header *types.Header) (*types.Receipt, error) {
	// Create EVM context
	blockContext := core.NewEVMBlockContext(header, e.blockchain, nil)
	
	// Initialize EVM
	evm := vm.NewEVM(blockContext, e.stateDB, e.getChainConfig(), e.vmConfig)
	
	// Execute transaction
	gp := new(core.GasPool).AddGas(header.GasLimit)
	result, err := core.ApplyTransaction(
		evm,
		gp,
		e.stateDB,
		header,
		tx,
		&header.GasUsed,
	)
	
	return result, err
}

// DeployContract deploys a smart contract
func (e *EVMExecutor) DeployContract(from common.Address, code []byte, value *big.Int, gas uint64) (common.Address, error) {
	// Create contract creation transaction
	tx := types.NewContractCreation(
		e.stateDB.GetNonce(from),
		value,
		gas,
		e.getGasPrice(),
		code,
	)
	
	// Sign and execute (simplified - in reality needs proper signing)
	header := &types.Header{
		Number:     big.NewInt(1),
		GasLimit:   gas * 2,
		Time:       uint64(1234567890),
		Difficulty: big.NewInt(1),
	}
	
	receipt, err := e.ExecuteTransaction(tx, header)
	if err != nil {
		return common.Address{}, err
	}
	
	return receipt.ContractAddress, nil
}

// CallContract calls a smart contract function
func (e *EVMExecutor) CallContract(to common.Address, data []byte, value *big.Int) ([]byte, error) {
	// Create EVM context
	header := &types.Header{
		Number:     big.NewInt(1),
		GasLimit:   1000000,
		Time:       uint64(1234567890),
		Difficulty: big.NewInt(1),
	}
	
	blockContext := core.NewEVMBlockContext(header, e.blockchain, nil)
	
	// Initialize EVM and execute
	evm := vm.NewEVM(blockContext, e.stateDB, e.getChainConfig(), e.vmConfig)
	val, _ := uint256.FromBig(value)
	if val == nil {
		val = new(uint256.Int)
	}
	result, _, err := evm.Call(common.Address{}, to, data, 1000000, val)
	
	return result, err
}

// GetContractCode returns the bytecode of a deployed contract
func (e *EVMExecutor) GetContractCode(address common.Address) []byte {
	return e.stateDB.GetCode(address)
}

// EstimateGas estimates the gas needed for a transaction
func (e *EVMExecutor) EstimateGas(from, to common.Address, data []byte, value *big.Int) (uint64, error) {
	// Run estimation (simplified)
	header := &types.Header{
		Number:     big.NewInt(1),
		GasLimit:   10000000,
		Time:       uint64(1234567890),
		Difficulty: big.NewInt(1),
	}
	
	blockContext := core.NewEVMBlockContext(header, e.blockchain, nil)
	
	evm := vm.NewEVM(blockContext, e.stateDB, e.getChainConfig(), e.vmConfig)
	val, _ := uint256.FromBig(value)
	if val == nil {
		val = new(uint256.Int)
	}
	
	_, gasLeft, err := evm.Call(from, to, data, 10000000, val)
	if err != nil {
		return 0, err
	}
	
	return 10000000 - gasLeft, nil
}

// getChainConfig returns the chain configuration compatible with Ethereum
func (e *EVMExecutor) getChainConfig() *params.ChainConfig {
	cfg := *params.AllEthashProtocolChanges
	cfg.ChainID = e.chainID
	return &cfg
}

// getGasPrice returns current gas price
func (e *EVMExecutor) getGasPrice() *big.Int {
	// Dynamic gas pricing - will be integrated with economics module
	return big.NewInt(1000000000) // 1 Gwei default
}

// GetBalance returns the balance of an account
func (e *EVMExecutor) GetBalance(address common.Address) *big.Int {
	return e.stateDB.GetBalance(address).ToBig()
}

// GetNonce returns the nonce of an account
func (e *EVMExecutor) GetNonce(address common.Address) uint64 {
	return e.stateDB.GetNonce(address)
}

// GetStorageAt returns the storage value at a specific slot
func (e *EVMExecutor) GetStorageAt(address common.Address, slot common.Hash) common.Hash {
	return e.stateDB.GetState(address, slot)
}