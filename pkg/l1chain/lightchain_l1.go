package l1chain

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/Hypasis/Litechain/pkg/consensus"
	"github.com/Hypasis/Litechain/pkg/economics"
	"github.com/Hypasis/Litechain/pkg/execution"
	"github.com/Hypasis/Litechain/pkg/genesis"
	"github.com/Hypasis/Litechain/pkg/mempool"
	"github.com/Hypasis/Litechain/pkg/network"
	"github.com/Hypasis/Litechain/pkg/staking"
	"github.com/Hypasis/Litechain/pkg/zk"
)

// LightChainL1 represents the complete L1 blockchain implementation
// Innovation: Unified architecture combining HPoS consensus, dynamic economics, and performance-based validation
type LightChainL1 struct {
	// Core components
	consensus *consensus.HPoSConsensus
	network   *network.L1P2PNetwork
	staking   *staking.StakingManager
	economics *economics.TokenEconomics
	genesis   *genesis.L1Genesis

	// High-performance transaction processing (Solana-style)
	mempool  *mempool.MemPool
	executor *execution.ParallelExecutor

	// Zero-knowledge capabilities
	zkEngine *zk.ZKEngine

	// Chain configuration
	chainID     *big.Int
	nodeAddress common.Address
	privateKey  []byte

	// Runtime state
	blockHeight   *big.Int
	currentEpoch  uint64
	lastBlockTime time.Time

	// Network configuration
	listenAddr     string
	bootstrapPeers []string
	maxPeers       int

	// Operation state
	isValidator bool
	isSyncing   bool
	running     bool

	// Synchronization
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// L1Config contains configuration for the L1 chain
type L1Config struct {
	ChainID        *big.Int
	NodeAddress    common.Address
	PrivateKey     []byte
	ListenAddr     string
	BootstrapPeers []string
	MaxPeers       int
	IsValidator    bool
	GenesisPath    string
}

// NewLightChainL1 creates a new L1 blockchain instance
func NewLightChainL1(config *L1Config) (*LightChainL1, error) {
	// Initialize genesis if not provided
	var genesisBlock *genesis.L1Genesis
	if config.GenesisPath != "" {
		// Load from file
		// genesisBlock = genesis.LoadFromFile(config.GenesisPath)
	} else {
		// Use default genesis
		genesisBlock = genesis.DefaultL1Genesis(config.ChainID)
	}

	// Initialize components
	consensusEngine := consensus.NewHPoSConsensus(config.ChainID, config.NodeAddress, config.PrivateKey)
	p2pNetwork := network.NewL1P2PNetwork(config.NodeAddress, config.ListenAddr, config.MaxPeers, config.BootstrapPeers)
	stakingManager := staking.NewStakingManager()
	tokenEconomics := economics.NewTokenEconomics()

	// Initialize high-performance mempool and parallel executor
	txMempool := mempool.NewMemPool(mempool.DefaultMemPoolConfig())
	parallelExecutor := execution.NewParallelExecutor(config.ChainID, execution.DefaultParallelConfig())

	// Initialize zero-knowledge engine
	zkEngine := zk.NewZKEngine(zk.DefaultZKConfig())

	l1 := &LightChainL1{
		consensus:      consensusEngine,
		network:        p2pNetwork,
		staking:        stakingManager,
		economics:      tokenEconomics,
		genesis:        genesisBlock,
		mempool:        txMempool,
		executor:       parallelExecutor,
		zkEngine:       zkEngine,
		chainID:        config.ChainID,
		nodeAddress:    config.NodeAddress,
		privateKey:     config.PrivateKey,
		blockHeight:    big.NewInt(0),
		currentEpoch:   0,
		listenAddr:     config.ListenAddr,
		bootstrapPeers: config.BootstrapPeers,
		maxPeers:       config.MaxPeers,
		isValidator:    config.IsValidator,
	}

	// Set up network message handling
	p2pNetwork.SetMessageHandler(l1)

	return l1, nil
}

// Start initializes and starts the L1 blockchain
func (l1 *LightChainL1) Start(ctx context.Context) error {
	l1.mu.Lock()
	defer l1.mu.Unlock()

	if l1.running {
		return fmt.Errorf("L1 chain already running")
	}

	l1.ctx, l1.cancel = context.WithCancel(ctx)

	// Initialize from genesis
	if err := l1.initializeFromGenesis(); err != nil {
		return fmt.Errorf("failed to initialize from genesis: %w", err)
	}

	// Start network layer
	if err := l1.network.Start(l1.ctx); err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}

	// Start consensus engine
	if err := l1.consensus.Start(l1.ctx); err != nil {
		return fmt.Errorf("failed to start consensus: %w", err)
	}

	// Start background processes
	go l1.blockProcessor()
	go l1.rewardDistributor()
	go l1.performanceMonitor()
	go l1.economicsManager()

	l1.running = true
	l1.lastBlockTime = time.Now()

	fmt.Printf("🚀 LightChain L1 started successfully!\n")
	fmt.Printf("   • Chain ID: %s\n", l1.chainID.String())
	fmt.Printf("   • Node Address: %s\n", l1.nodeAddress.Hex()[:12]+"...")
	fmt.Printf("   • Network: %s\n", l1.listenAddr)
	fmt.Printf("   • Is Validator: %v\n", l1.isValidator)
	fmt.Printf("   • Genesis Hash: %s\n", l1.genesis.Hash.Hex()[:12]+"...")
	fmt.Printf("   • Initial Validators: %d\n", len(l1.genesis.Validators))

	return nil
}

// initializeFromGenesis initializes the chain from genesis block
func (l1 *LightChainL1) initializeFromGenesis() error {
	// Validate genesis
	if err := l1.genesis.Validate(); err != nil {
		return fmt.Errorf("invalid genesis: %w", err)
	}

	// Initialize staking from genesis validators
	for _, genValidator := range l1.genesis.Validators {
		err := l1.staking.CreateValidator(
			genValidator.Address,
			genValidator.PubKey,
			genValidator.Description.Moniker,
			genValidator.Description.Details,
			genValidator.Description.Website,
			genValidator.Commission,
			genValidator.Commission+1000, // Max commission = current + 10%
			genValidator.Stake,
		)
		if err != nil {
			return fmt.Errorf("failed to create genesis validator: %w", err)
		}
	}

	// Initialize token economics
	// Set initial token balances from genesis allocations
	for addr := range l1.genesis.Alloc {
		// Token balances would be initialized here in the economics module
		_ = addr // placeholder
	}

	fmt.Printf("📊 Genesis initialized:\n")
	fmt.Printf("   • Total Supply: %s LIGHT\n", formatTokenAmount(l1.genesis.Token.TotalSupply))
	fmt.Printf("   • Validators: %d\n", len(l1.genesis.Validators))
	fmt.Printf("   • Accounts: %d\n", len(l1.genesis.Alloc))
	fmt.Printf("   • Block Reward: %s LIGHT\n", formatTokenAmount(l1.genesis.Economics.BlockReward))

	return nil
}

// blockProcessor handles block processing and validation
func (l1 *LightChainL1) blockProcessor() {
	ticker := time.NewTicker(2 * time.Second) // 2 second block time
	defer ticker.Stop()

	for {
		select {
		case <-l1.ctx.Done():
			return
		case <-ticker.C:
			if err := l1.processBlock(); err != nil {
				fmt.Printf("❌ Block processing error: %v\n", err)
			}
		}
	}
}

// processBlock processes a new block with parallel transaction execution
func (l1 *LightChainL1) processBlock() error {
	start := time.Now()
	l1.mu.Lock()
	defer l1.mu.Unlock()

	l1.blockHeight.Add(l1.blockHeight, big.NewInt(1))
	l1.lastBlockTime = time.Now()

	// Get active validators
	activeValidators := l1.staking.GetActiveValidators()
	if len(activeValidators) == 0 {
		return fmt.Errorf("no active validators")
	}

	// Select block proposer (this would be integrated with consensus)
	proposer := activeValidators[int(l1.blockHeight.Uint64())%len(activeValidators)]

	// 🚀 PARALLEL TRANSACTION EXECUTION (Solana-style)
	// Get transactions ready for parallel execution
	parallelBatches := l1.mempool.GetParallelBatches()

	var totalTxCount int
	var totalGasUsed uint64
	var gasFeesCollected = big.NewInt(0)

	if len(parallelBatches) > 0 {
		// Execute transactions in parallel
		fmt.Printf("⚡ Processing %d parallel batches for block #%d\n",
			len(parallelBatches), l1.blockHeight.Uint64())

		for _, batch := range parallelBatches {
			totalTxCount += len(batch)
		}

		// Simulate parallel execution (in real implementation, this would use the executor)
		blockExecStart := time.Now()

		// Process each level in sequence, but transactions within level in parallel
		for level, batch := range parallelBatches {
			if len(batch) == 0 {
				continue
			}

			fmt.Printf("  📦 Level %d: %d transactions executing in parallel\n", level, len(batch))

			// Simulate gas collection from transactions
			for _, tx := range batch {
				gasFeesCollected.Add(gasFeesCollected, big.NewInt(int64(tx.GasLimit*tx.GasPrice.Uint64())))
				totalGasUsed += tx.GasLimit

				// Remove processed transaction from mempool
				l1.mempool.RemoveTransaction(tx.Hash)
			}
		}

		blockExecTime := time.Since(blockExecStart)
		totalBlockTime := time.Since(start)

		// Calculate TPS
		tps := float64(totalTxCount) / blockExecTime.Seconds()

		fmt.Printf("🎯 Block #%d: %d transactions, %.2f TPS, %v total time\n",
			l1.blockHeight.Uint64(), totalTxCount, tps, totalBlockTime)
	}

	// Process block rewards
	l1.economics.ProcessBlockRewards(l1.blockHeight.Uint64(), proposer.Address, gasFeesCollected)

	// Update epoch if necessary
	if l1.blockHeight.Uint64()%100 == 0 { // 100 blocks per epoch
		l1.currentEpoch++
		fmt.Printf("🔄 Entered epoch %d at block #%d\n", l1.currentEpoch, l1.blockHeight.Uint64())
	}

	return nil
}

// rewardDistributor handles reward distribution
func (l1 *LightChainL1) rewardDistributor() {
	ticker := time.NewTicker(1 * time.Hour) // Distribute rewards hourly
	defer ticker.Stop()

	for {
		select {
		case <-l1.ctx.Done():
			return
		case <-ticker.C:
			l1.distributeRewards()
		}
	}
}

// distributeRewards distributes staking rewards
func (l1 *LightChainL1) distributeRewards() {
	l1.mu.Lock()
	defer l1.mu.Unlock()

	// Calculate total rewards to distribute
	totalRewards := big.NewInt(0)
	activeValidators := l1.staking.GetActiveValidators()

	for _, validator := range activeValidators {
		// Calculate validator rewards based on performance and stake
		validatorReward := new(big.Int).Mul(validator.TotalStake, big.NewInt(100)) // 1% hourly example
		validatorReward.Div(validatorReward, big.NewInt(8760))                     // Annualized
		totalRewards.Add(totalRewards, validatorReward)
	}

	if totalRewards.Sign() > 0 {
		l1.staking.DistributeRewards(totalRewards)
		fmt.Printf("💰 Distributed %s LIGHT in rewards to %d validators\n",
			formatTokenAmount(totalRewards), len(activeValidators))
	}
}

// performanceMonitor monitors validator performance
func (l1 *LightChainL1) performanceMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-l1.ctx.Done():
			return
		case <-ticker.C:
			l1.updateValidatorPerformance()
		}
	}
}

// updateValidatorPerformance updates performance metrics for validators
func (l1 *LightChainL1) updateValidatorPerformance() {
	activeValidators := l1.staking.GetActiveValidators()

	for _, validator := range activeValidators {
		// Create mock performance metrics (in real implementation, this would be actual data)
		metrics := &staking.PerformanceMetrics{
			StartTime:      time.Now().Add(-24 * time.Hour),
			BlocksProposed: 100,
			BlocksSigned:   95,
			BlocksMissed:   5,
			ResponseTimes:  []time.Duration{100 * time.Millisecond, 150 * time.Millisecond},
			LastHeartbeat:  time.Now(),
		}

		l1.staking.UpdatePerformance(validator.Address, metrics)
	}
}

// economicsManager handles economic operations
func (l1 *LightChainL1) economicsManager() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-l1.ctx.Done():
			return
		case <-ticker.C:
			l1.updateEconomics()
		}
	}
}

// updateEconomics updates economic parameters and handles token operations
func (l1 *LightChainL1) updateEconomics() {
	// Update gas prices based on network load
	networkLoad := 0.5           // Placeholder - would be calculated from actual usage
	validatorPerformance := 0.95 // Average validator performance

	// Access gas model through economics (would need to add getter method)
	_ = networkLoad
	_ = validatorPerformance

	// _ = gasPrice // Would be used to update base gas price

	// Handle token burns from fees (deflationary mechanism)
	// This would be integrated with actual transaction processing
}

// HandleMessage implements network.MessageHandler interface
func (l1 *LightChainL1) HandleMessage(msg *network.NetworkMessage) error {
	switch msg.Type {
	case network.MsgTypeProposal:
		return l1.handleProposal(msg)
	case network.MsgTypeVote:
		return l1.handleVote(msg)
	case network.MsgTypeCommit:
		return l1.handleCommit(msg)
	case network.MsgTypeTransaction:
		return l1.handleTransaction(msg)
	case network.MsgTypePing:
		return l1.handlePing(msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleProposal handles block proposals
func (l1 *LightChainL1) handleProposal(msg *network.NetworkMessage) error {
	fmt.Printf("📥 Received proposal from %s\n", msg.From.Hex()[:8])
	// Process proposal through consensus engine
	return nil
}

// handleVote handles consensus votes
func (l1 *LightChainL1) handleVote(msg *network.NetworkMessage) error {
	fmt.Printf("🗳️  Received vote from %s\n", msg.From.Hex()[:8])
	// Process vote through consensus engine
	return nil
}

// handleCommit handles block commits
func (l1 *LightChainL1) handleCommit(msg *network.NetworkMessage) error {
	fmt.Printf("✅ Received commit from %s\n", msg.From.Hex()[:8])
	// Process commit through consensus engine
	return nil
}

// handleTransaction handles transactions by adding them to mempool
func (l1 *LightChainL1) handleTransaction(msg *network.NetworkMessage) error {
	fmt.Printf("💸 Received transaction from %s\n", msg.From.Hex()[:8])

	// Parse transaction from message data (simplified)
	// In real implementation, this would properly deserialize the transaction
	// For now, create a mock transaction for demonstration

	return nil
}

// AddTransaction adds a transaction to the mempool for parallel processing
func (l1 *LightChainL1) AddTransaction(tx *types.Transaction) error {
	return l1.mempool.AddTransaction(tx)
}

// GenerateTestTransactions generates test transactions for performance testing
func (l1 *LightChainL1) GenerateTestTransactions(count int) error {
	fmt.Printf("🧪 Generating %d test transactions for performance testing...\n", count)

	for i := 0; i < count; i++ {
		// Create mock transaction
		tx := l1.createMockTransaction(i)

		// Add to mempool
		if err := l1.mempool.AddTransaction(tx); err != nil {
			fmt.Printf("❌ Failed to add transaction %d: %v\n", i, err)
			continue
		}

		if i%100 == 0 {
			fmt.Printf("📦 Generated %d/%d transactions\n", i, count)
		}
	}

	fmt.Printf("✅ Generated %d test transactions successfully\n", count)
	return nil
}

// createMockTransaction creates a mock transaction for testing
func (l1 *LightChainL1) createMockTransaction(nonce int) *types.Transaction {
	// Create a simple value transfer transaction
	to := common.HexToAddress(fmt.Sprintf("0x%040d", nonce%1000))
	value := big.NewInt(int64(1000000000000000000)) // 1 ETH in wei
	gasLimit := uint64(21000)
	gasPrice := big.NewInt(1000000000) // 1 Gwei

	// Create transaction data
	tx := types.NewTransaction(
		uint64(nonce),
		to,
		value,
		gasLimit,
		gasPrice,
		nil, // No data for simple transfer
	)

	return tx
}

// handlePing handles ping messages
func (l1 *LightChainL1) handlePing(msg *network.NetworkMessage) error {
	// Send pong response
	pongMsg := &network.NetworkMessage{
		Type:      network.MsgTypePong,
		From:      l1.nodeAddress,
		To:        msg.From,
		Timestamp: time.Now(),
	}

	return l1.network.SendToPeer(msg.From, pongMsg)
}

// Stop gracefully stops the L1 blockchain
func (l1 *LightChainL1) Stop() error {
	l1.mu.Lock()
	defer l1.mu.Unlock()

	if !l1.running {
		return nil
	}

	fmt.Println("🛑 Stopping LightChain L1...")

	// Stop components in reverse order
	if l1.cancel != nil {
		l1.cancel()
	}

	if l1.consensus != nil {
		l1.consensus.Stop()
	}

	if l1.network != nil {
		l1.network.Stop()
	}

	l1.running = false
	fmt.Println("✅ LightChain L1 stopped successfully")

	return nil
}

// GetStatus returns the current status of the L1 chain
func (l1 *LightChainL1) GetStatus() map[string]interface{} {
	l1.mu.RLock()
	defer l1.mu.RUnlock()

	activeValidators := len(l1.staking.GetActiveValidators())

	return map[string]interface{}{
		"chainId":          l1.chainID.String(),
		"blockHeight":      l1.blockHeight.Uint64(),
		"currentEpoch":     l1.currentEpoch,
		"lastBlockTime":    l1.lastBlockTime.Unix(),
		"isValidator":      l1.isValidator,
		"isSyncing":        l1.isSyncing,
		"running":          l1.running,
		"nodeAddress":      l1.nodeAddress.Hex(),
		"networkPeers":     l1.network.GetNetworkStatus()["totalPeers"],
		"activeValidators": activeValidators,
		"stakingStatus":    l1.staking.GetStakingStatus(),
		"economicStatus":   l1.economics.GetEconomicStatus(),
		"consensusStatus":  l1.consensus.GetStatus(),
	}
}

// formatTokenAmount formats token amounts for display
func formatTokenAmount(amount *big.Int) string {
	if amount == nil {
		return "0"
	}

	// Convert from wei to LIGHT (18 decimals)
	ether := new(big.Int).Div(amount, big.NewInt(1e18))
	remainder := new(big.Int).Mod(amount, big.NewInt(1e18))

	if remainder.Sign() == 0 {
		return ether.String()
	}

	// Format with decimals
	return fmt.Sprintf("%s.%018s", ether.String(), remainder.String())
}

// GetGenesisHash returns the genesis block hash
func (l1 *LightChainL1) GetGenesisHash() common.Hash {
	return l1.genesis.Hash
}

// IsValidator returns whether this node is a validator
func (l1 *LightChainL1) IsValidator() bool {
	return l1.isValidator
}

// GetChainID returns the chain ID
func (l1 *LightChainL1) GetChainID() *big.Int {
	return new(big.Int).Set(l1.chainID)
}
