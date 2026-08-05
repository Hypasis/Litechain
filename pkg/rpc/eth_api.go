package rpc

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/Hypasis/Litechain/pkg/evm"
)

// EthAPI provides Ethereum-compatible JSON-RPC API
type EthAPI struct {
	evm       *evm.EVMExecutor
	backend   Backend
	chainID   *big.Int
}

// Backend interface for blockchain data access
type Backend interface {
	GetBlock(hash common.Hash) *types.Block
	GetBlockByNumber(number *big.Int) *types.Block
	GetTransaction(hash common.Hash) *types.Transaction
	GetReceipt(hash common.Hash) *types.Receipt
	SendTransaction(tx *types.Transaction) error
	GetNonce(address common.Address) uint64
}

// NewEthAPI creates a new Ethereum-compatible API
func NewEthAPI(evmExecutor *evm.EVMExecutor, backend Backend, chainID *big.Int) *EthAPI {
	return &EthAPI{
		evm:     evmExecutor,
		backend: backend,
		chainID: chainID,
	}
}

// ChainId returns the current chain ID
func (api *EthAPI) ChainId() *hexutil.Big {
	return (*hexutil.Big)(api.chainID)
}

// GetBalance returns the balance of an account
func (api *EthAPI) GetBalance(ctx context.Context, address common.Address, blockNumber *big.Int) *hexutil.Big {
	balance := api.evm.GetBalance(address)
	return (*hexutil.Big)(balance)
}

// GetTransactionCount returns the nonce/transaction count of an account
func (api *EthAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNumber *big.Int) *hexutil.Uint64 {
	nonce := api.evm.GetNonce(address)
	return (*hexutil.Uint64)(&nonce)
}

// GetCode returns the contract code at the given address
func (api *EthAPI) GetCode(ctx context.Context, address common.Address, blockNumber *big.Int) hexutil.Bytes {
	code := api.evm.GetContractCode(address)
	return code
}

// GetStorageAt returns the storage value at the specified address and slot
func (api *EthAPI) GetStorageAt(ctx context.Context, address common.Address, slot common.Hash, blockNumber *big.Int) hexutil.Bytes {
	value := api.evm.GetStorageAt(address, slot)
	return value.Bytes()
}

// Call executes a contract call without creating a transaction
func (api *EthAPI) Call(ctx context.Context, args CallArgs, blockNumber *big.Int) (hexutil.Bytes, error) {
	to := args.To
	if to == nil {
		return nil, nil // Contract deployment call
	}
	
	result, err := api.evm.CallContract(*to, args.Data, args.Value.ToInt())
	if err != nil {
		return nil, err
	}
	
	return result, nil
}

// EstimateGas estimates the gas needed for a transaction
func (api *EthAPI) EstimateGas(ctx context.Context, args CallArgs) (hexutil.Uint64, error) {
	from := args.From
	if from == nil {
		from = &common.Address{} // Default address
	}
	
	to := args.To
	if to == nil {
		to = &common.Address{} // Contract creation
	}
	
	gas, err := api.evm.EstimateGas(*from, *to, args.Data, args.Value.ToInt())
	if err != nil {
		return 0, err
	}
	
	return hexutil.Uint64(gas), nil
}

// SendTransaction submits a signed transaction to the network
func (api *EthAPI) SendTransaction(ctx context.Context, tx *types.Transaction) (common.Hash, error) {
	err := api.backend.SendTransaction(tx)
	if err != nil {
		return common.Hash{}, err
	}
	
	return tx.Hash(), nil
}

// SendRawTransaction submits a raw signed transaction
func (api *EthAPI) SendRawTransaction(ctx context.Context, data hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(data); err != nil {
		return common.Hash{}, err
	}
	
	return api.SendTransaction(ctx, tx)
}

// GetTransactionByHash returns transaction details by hash
func (api *EthAPI) GetTransactionByHash(ctx context.Context, hash common.Hash) *Transaction {
	tx := api.backend.GetTransaction(hash)
	if tx == nil {
		return nil
	}
	
	return newTransaction(tx)
}

// GetTransactionReceipt returns the receipt of a transaction
func (api *EthAPI) GetTransactionReceipt(ctx context.Context, hash common.Hash) *Receipt {
	receipt := api.backend.GetReceipt(hash)
	if receipt == nil {
		return nil
	}
	
	return newReceipt(receipt)
}

// GetBlockByNumber returns block details by number
func (api *EthAPI) GetBlockByNumber(ctx context.Context, number *big.Int, fullTx bool) *Block {
	block := api.backend.GetBlockByNumber(number)
	if block == nil {
		return nil
	}
	
	return newBlock(block, fullTx)
}

// GetBlockByHash returns block details by hash
func (api *EthAPI) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) *Block {
	block := api.backend.GetBlock(hash)
	if block == nil {
		return nil
	}
	
	return newBlock(block, fullTx)
}

// BlockNumber returns the current block number
func (api *EthAPI) BlockNumber(ctx context.Context) hexutil.Uint64 {
	// Get from backend - simplified
	return hexutil.Uint64(1) 
}

// GasPrice returns the current gas price
func (api *EthAPI) GasPrice(ctx context.Context) *hexutil.Big {
	// Dynamic gas pricing - integrate with economics
	return (*hexutil.Big)(big.NewInt(1000000000)) // 1 Gwei
}

// CallArgs represents call arguments
type CallArgs struct {
	From     *common.Address `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
}

// Transaction represents an Ethereum transaction
type Transaction struct {
	Hash             common.Hash     `json:"hash"`
	From             common.Address  `json:"from"`
	To               *common.Address `json:"to"`
	Value            *hexutil.Big    `json:"value"`
	Gas              hexutil.Uint64  `json:"gas"`
	GasPrice         *hexutil.Big    `json:"gasPrice"`
	Nonce            hexutil.Uint64  `json:"nonce"`
	Data             hexutil.Bytes   `json:"input"`
	BlockHash        *common.Hash    `json:"blockHash"`
	BlockNumber      *hexutil.Big    `json:"blockNumber"`
	TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
}

// Receipt represents a transaction receipt
type Receipt struct {
	TransactionHash   common.Hash     `json:"transactionHash"`
	TransactionIndex  hexutil.Uint64  `json:"transactionIndex"`
	BlockHash         common.Hash     `json:"blockHash"`
	BlockNumber       *hexutil.Big    `json:"blockNumber"`
	From              common.Address  `json:"from"`
	To                *common.Address `json:"to"`
	CumulativeGasUsed hexutil.Uint64  `json:"cumulativeGasUsed"`
	GasUsed           hexutil.Uint64  `json:"gasUsed"`
	ContractAddress   *common.Address `json:"contractAddress"`
	Logs              []*Log          `json:"logs"`
	Status            hexutil.Uint64  `json:"status"`
}

// Block represents a block
type Block struct {
	Number           *hexutil.Big    `json:"number"`
	Hash             common.Hash     `json:"hash"`
	ParentHash       common.Hash     `json:"parentHash"`
	Timestamp        hexutil.Uint64  `json:"timestamp"`
	Size             hexutil.Uint64  `json:"size"`
	GasLimit         hexutil.Uint64  `json:"gasLimit"`
	GasUsed          hexutil.Uint64  `json:"gasUsed"`
	Transactions     interface{}     `json:"transactions"`
	TransactionsRoot common.Hash     `json:"transactionsRoot"`
	StateRoot        common.Hash     `json:"stateRoot"`
	ReceiptsRoot     common.Hash     `json:"receiptsRoot"`
}

// Log represents a contract log
type Log struct {
	Address     common.Address `json:"address"`
	Topics      []common.Hash  `json:"topics"`
	Data        hexutil.Bytes  `json:"data"`
	BlockNumber *hexutil.Big   `json:"blockNumber"`
	BlockHash   common.Hash    `json:"blockHash"`
	TxHash      common.Hash    `json:"transactionHash"`
	TxIndex     hexutil.Uint64 `json:"transactionIndex"`
	LogIndex    hexutil.Uint64 `json:"logIndex"`
}

// Helper functions to convert types
func newTransaction(tx *types.Transaction) *Transaction {
	from, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	
	return &Transaction{
		Hash:     tx.Hash(),
		From:     from,
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		Data:     tx.Data(),
	}
}

func newReceipt(receipt *types.Receipt) *Receipt {
	var contractAddr *common.Address
	if receipt.ContractAddress != (common.Address{}) {
		contractAddr = &receipt.ContractAddress
	}
	return &Receipt{
		TransactionHash:   receipt.TxHash,
		TransactionIndex:  hexutil.Uint64(receipt.TransactionIndex),
		BlockHash:         receipt.BlockHash,
		BlockNumber:       (*hexutil.Big)(receipt.BlockNumber),
		CumulativeGasUsed: hexutil.Uint64(receipt.CumulativeGasUsed),
		GasUsed:           hexutil.Uint64(receipt.GasUsed),
		ContractAddress:   contractAddr,
		Status:            hexutil.Uint64(receipt.Status),
	}
}

func newBlock(block *types.Block, fullTx bool) *Block {
	b := &Block{
		Number:           (*hexutil.Big)(block.Number()),
		Hash:             block.Hash(),
		ParentHash:       block.ParentHash(),
		Timestamp:        hexutil.Uint64(block.Time()),
		Size:             hexutil.Uint64(block.Size()),
		GasLimit:         hexutil.Uint64(block.GasLimit()),
		GasUsed:          hexutil.Uint64(block.GasUsed()),
		TransactionsRoot: block.TxHash(),
		StateRoot:        block.Root(),
		ReceiptsRoot:     block.ReceiptHash(),
	}
	
	if fullTx {
		txs := make([]*Transaction, len(block.Transactions()))
		for i, tx := range block.Transactions() {
			txs[i] = newTransaction(tx)
		}
		b.Transactions = txs
	} else {
		hashes := make([]common.Hash, len(block.Transactions()))
		for i, tx := range block.Transactions() {
			hashes[i] = tx.Hash()
		}
		b.Transactions = hashes
	}
	
	return b
}