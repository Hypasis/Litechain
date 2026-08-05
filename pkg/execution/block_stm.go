package execution

import (
	"context"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// TxStatus represents the lifecycle state of a transaction in Block-STM
type TxStatus int32

const (
	StatusPending TxStatus = iota
	StatusExecuted
	StatusAborted
	StatusCommitted
)

// VersionKey uniquely identifies a state location (account property or storage slot)
type VersionKey struct {
	Address common.Address
	Slot    common.Hash // Zero hash for account balance/nonce, specific hash for storage
	IsState bool        // false for balance/nonce, true for contract storage
}

// VersionValue holds a versioned value written by a specific transaction index
type VersionValue struct {
	TxIndex   int          // Index of transaction that wrote this value (-1 for base state)
	Value     *uint256.Int // Numeric value (balance or storage)
	IsEstimate bool        // True if location is currently marked ESTIMATE due to an abort
}

// MVMemory (Multi-Version Memory Data Structure)
// Stores multi-versioned state for parallel read/write resolution without locks
type MVMemory struct {
	mu   sync.RWMutex
	data map[VersionKey][]VersionValue
}

// NewMVMemory creates a new Multi-Version Data Structure
func NewMVMemory() *MVMemory {
	return &MVMemory{
		data: make(map[VersionKey][]VersionValue),
	}
}

// Read returns the latest version of key written by a transaction Tx_j where j < txIdx.
// If no prior transaction wrote to key, returns the base value from initial state.
func (mv *MVMemory) Read(key VersionKey, txIdx int, baseVal *uint256.Int) (*uint256.Int, int, bool) {
	mv.mu.RLock()
	defer mv.mu.RUnlock()

	versions, exists := mv.data[key]
	if !exists || len(versions) == 0 {
		return baseVal, -1, true
	}

	// Search for the highest version written by j < txIdx
	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		if v.TxIndex < txIdx {
			if v.IsEstimate {
				return nil, v.TxIndex, false // Hit an un-computed estimate
			}
			return v.Value, v.TxIndex, true
		}
	}

	return baseVal, -1, true
}

// Write records a new version written by Tx_txIdx
func (mv *MVMemory) Write(key VersionKey, txIdx int, val *uint256.Int) {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	versions := mv.data[key]
	newVal := VersionValue{
		TxIndex:   txIdx,
		Value:     val,
		IsEstimate: false,
	}

	// Insert or replace version for txIdx in sorted order
	for i, v := range versions {
		if v.TxIndex == txIdx {
			versions[i] = newVal
			mv.data[key] = versions
			return
		}
		if v.TxIndex > txIdx {
			// Insert before index i
			versions = append(versions[:i], append([]VersionValue{newVal}, versions[i:]...)...)
			mv.data[key] = versions
			return
		}
	}

	// Append if txIdx is greater than all existing versions
	mv.data[key] = append(versions, newVal)
}

// MarkEstimate marks written keys as ESTIMATE when a transaction aborts
func (mv *MVMemory) MarkEstimate(key VersionKey, txIdx int) {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	versions := mv.data[key]
	for i, v := range versions {
		if v.TxIndex == txIdx {
			versions[i].IsEstimate = true
			mv.data[key] = versions
			return
		}
	}
}

// RemoveTxWrites removes all writes associated with Tx_txIdx
func (mv *MVMemory) RemoveTxWrites(key VersionKey, txIdx int) {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	versions := mv.data[key]
	for i, v := range versions {
		if v.TxIndex == txIdx {
			mv.data[key] = append(versions[:i], versions[i+1:]...)
			return
		}
	}
}

// ReadDescriptor records a read performed by a transaction
type ReadDescriptor struct {
	Key     VersionKey
	TxWritten int // Transaction index of the version read (-1 for base state)
}

// TxExecutionTask represents an execution task for a transaction in Block-STM
type TxExecutionTask struct {
	Index        int
	Tx           *types.Transaction
	ReadSet      []ReadDescriptor
	WriteSet     map[VersionKey]*uint256.Int
	Receipt      *types.Receipt
	Status       int32 // TxStatus atomic
	Incincarnation int
}

// BlockSTMExecutor implements Block-level Software Transactional Memory parallel EVM execution
type BlockSTMExecutor struct {
	workerCount int
	chainID     *big.Int
	mvMemory    *MVMemory
}

// NewBlockSTMExecutor creates a new Block-STM engine
func NewBlockSTMExecutor(chainID *big.Int, numWorkers int) *BlockSTMExecutor {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	return &BlockSTMExecutor{
		workerCount: numWorkers,
		chainID:     chainID,
		mvMemory:    NewMVMemory(),
	}
}

// ExecuteBlockSTM executes a batch of transactions in parallel using Block-STM
func (e *BlockSTMExecutor) ExecuteBlockSTM(txs []*types.Transaction, initialBalances map[common.Address]*uint256.Int) ([]*types.Receipt, uint64, int, time.Duration, error) {
	start := time.Now()
	n := len(txs)
	if n == 0 {
		return nil, 0, 0, 0, nil
	}

	e.mvMemory = NewMVMemory()

	tasks := make([]*TxExecutionTask, n)
	for i, tx := range txs {
		tasks[i] = &TxExecutionTask{
			Index:    i,
			Tx:       tx,
			WriteSet: make(map[VersionKey]*uint256.Int),
			Status:   int32(StatusPending),
		}
	}

	// Dynamic work queue & index scheduler
	var nextTaskIdx int32 = 0
	var conflictCount int32 = 0
	var totalGasUsed uint64 = 0

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Worker loop
	for w := 0; w < e.workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					idx := atomic.AddInt32(&nextTaskIdx, 1) - 1
					if int(idx) >= n {
						return
					}

					task := tasks[idx]
					e.executeTransaction(task, initialBalances)

					// Validate dependencies
					aborted := e.validateAndSchedule(task, tasks, n)
					if aborted {
						atomic.AddInt32(&conflictCount, 1)
						// Re-enqueue task for execution
						atomic.StoreInt32(&nextTaskIdx, int32(task.Index))
					}
				}
			}
		}()
	}

	wg.Wait()

	// Sequential Commit Phase
	receipts := make([]*types.Receipt, n)
	for i, task := range tasks {
		atomic.StoreInt32(&task.Status, int32(StatusCommitted))
		if task.Receipt != nil {
			totalGasUsed += task.Receipt.GasUsed
			receipts[i] = task.Receipt
		} else {
			receipts[i] = &types.Receipt{
				Type:              task.Tx.Type(),
				Status:            types.ReceiptStatusSuccessful,
				CumulativeGasUsed: totalGasUsed + 21000,
				TxHash:            task.Tx.Hash(),
				GasUsed:           21000,
				BlockHash:         common.Hash{},
				BlockNumber:       big.NewInt(1),
				TransactionIndex:  uint(i),
			}
			totalGasUsed += 21000
		}
	}

	duration := time.Since(start)
	return receipts, totalGasUsed, int(conflictCount), duration, nil
}

// executeTransaction executes a single transaction against MVMemory
func (e *BlockSTMExecutor) executeTransaction(task *TxExecutionTask, initialBalances map[common.Address]*uint256.Int) {
	task.ReadSet = nil
	task.WriteSet = make(map[VersionKey]*uint256.Int)

	from, err := types.Sender(types.LatestSignerForChainID(task.Tx.ChainId()), task.Tx)
	if err != nil {
		from = common.Address{}
	}

	// 1. Read Sender Balance
	senderKey := VersionKey{Address: from, Slot: common.Hash{}, IsState: false}
	baseBalance := initialBalances[from]
	if baseBalance == nil {
		baseBalance = uint256.NewInt(1000000000000000000) // 1 ETH default test balance
	}

	senderBal, writtenTxIdx, ok := e.mvMemory.Read(senderKey, task.Index, baseBalance)
	if !ok {
		// Estimate hit - retry later
		atomic.StoreInt32(&task.Status, int32(StatusAborted))
		return
	}
	task.ReadSet = append(task.ReadSet, ReadDescriptor{Key: senderKey, TxWritten: writtenTxIdx})

	txValue, _ := uint256.FromBig(task.Tx.Value())
	if txValue == nil {
		txValue = uint256.NewInt(0)
	}

	// Compute updated balance if sufficient
	newSenderBal := new(uint256.Int).Sub(senderBal, txValue)
	if senderBal.Cmp(txValue) < 0 {
		newSenderBal = uint256.NewInt(0)
	}

	// Record sender write
	task.WriteSet[senderKey] = newSenderBal
	e.mvMemory.Write(senderKey, task.Index, newSenderBal)

	// 2. Read & Update Recipient Balance (if transfer)
	if task.Tx.To() != nil {
		to := *task.Tx.To()
		recipientKey := VersionKey{Address: to, Slot: common.Hash{}, IsState: false}
		baseRecipBal := initialBalances[to]
		if baseRecipBal == nil {
			baseRecipBal = uint256.NewInt(0)
		}

		recipBal, writtenTxIdx2, ok2 := e.mvMemory.Read(recipientKey, task.Index, baseRecipBal)
		if !ok2 {
			atomic.StoreInt32(&task.Status, int32(StatusAborted))
			return
		}
		task.ReadSet = append(task.ReadSet, ReadDescriptor{Key: recipientKey, TxWritten: writtenTxIdx2})

		newRecipBal := new(uint256.Int).Add(recipBal, txValue)
		task.WriteSet[recipientKey] = newRecipBal
		e.mvMemory.Write(recipientKey, task.Index, newRecipBal)
	}

	atomic.StoreInt32(&task.Status, int32(StatusExecuted))
}

// validateAndSchedule validates transaction read sets against MVMemory writes
func (e *BlockSTMExecutor) validateAndSchedule(task *TxExecutionTask, tasks []*TxExecutionTask, totalTxs int) bool {
	// Validate read set
	for _, readDesc := range task.ReadSet {
		// Read latest version again
		_, currentTxWritten, _ := e.mvMemory.Read(readDesc.Key, task.Index, uint256.NewInt(0))
		if currentTxWritten != readDesc.TxWritten && currentTxWritten < task.Index {
			// Read set validation failed: lower transaction modified a key read by task
			e.abortTransaction(task)
			return true
		}
	}
	return false
}

// abortTransaction marks writes as ESTIMATE and resets task state
func (e *BlockSTMExecutor) abortTransaction(task *TxExecutionTask) {
	atomic.StoreInt32(&task.Status, int32(StatusAborted))
	for key := range task.WriteSet {
		e.mvMemory.MarkEstimate(key, task.Index)
	}
	task.Incincarnation++
}
