package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/Hypasis/Litechain/pkg/l1chain"
	"github.com/Hypasis/Litechain/pkg/version"
)

const (
	appName = "Litechain L1"
)

func main() {
	var (
		nodeType      = flag.String("type", "validator", "Node type: validator, full, light")
		showVersion   = flag.Bool("version", false, "Show version information")
		dataDir       = flag.String("data-dir", "./data", "Data directory for blockchain storage")
		logLevel      = flag.String("log-level", "info", "Log level: debug, info, warn, error")
		chainID       = flag.Uint64("chain-id", 1001, "L1 Chain ID (1001=LightChain, 1337=LightBeam testnet)")
		listenAddr    = flag.String("listen", "0.0.0.0:30300", "P2P listen address")
		bootstrap     = flag.String("bootstrap", "", "Comma-separated list of bootstrap peers")
		genesisPath   = flag.String("genesis", "", "Path to genesis file (optional)")
	)
	flag.Parse()

	if *showVersion || (len(os.Args) > 1 && os.Args[1] == "version") {
		fmt.Println(version.FormattedVersion())
		os.Exit(0)
	}

	// Initialize logger
	logger := setupLogger(*logLevel)
	logger.Printf("🌟 Starting %s v%s", appName, version.Version)
	logger.Printf("   • Node Type: %s", *nodeType)
	logger.Printf("   • Chain ID: %d", *chainID)
	logger.Printf("   • Data Directory: %s", *dataDir)
	logger.Printf("   • Listen Address: %s", *listenAddr)

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Generate node address and private key (in production, load from keystore)
	nodeAddr, privateKey := generateNodeIdentity()
	logger.Printf("   • Node Address: %s", nodeAddr.Hex())

	// Parse bootstrap peers
	var bootstrapPeers []string
	if *bootstrap != "" {
		bootstrapPeers = []string{*bootstrap} // Simplified parsing
	}

	// Create L1 configuration
	l1Config := &l1chain.L1Config{
		ChainID:        big.NewInt(int64(*chainID)),
		NodeAddress:    nodeAddr,
		PrivateKey:     privateKey,
		ListenAddr:     *listenAddr,
		BootstrapPeers: bootstrapPeers,
		MaxPeers:       50,
		IsValidator:    *nodeType == "validator",
		GenesisPath:    *genesisPath,
	}

	// Create main context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the L1 blockchain
	logger.Printf("🚀 Initializing LightChain L1 Independent Blockchain...")
	logger.Printf("   • Innovation: HPoS consensus + Performance-weighted validation")
	logger.Printf("   • Features: Native token economics + Dynamic gas model")
	logger.Printf("   • Network: P2P with validator-priority routing")

	l1chain, err := l1chain.NewLightChainL1(l1Config)
	if err != nil {
		log.Fatalf("Failed to create L1 chain: %v", err)
	}

	// Start the L1 blockchain
	if err := l1chain.Start(ctx); err != nil {
		log.Fatalf("Failed to start L1 chain: %v", err)
	}

	// Print status
	status := l1chain.GetStatus()
	logger.Printf("✅ LightChain L1 started successfully!")
	logger.Printf("   🔗 Genesis Hash: %s", l1chain.GetGenesisHash().Hex()[:16]+"...")
	logger.Printf("   ⛓️  Block Height: %v", status["blockHeight"])
	logger.Printf("   👥 Active Validators: %v", status["activeValidators"])
	logger.Printf("   🌐 Network Peers: %v", status["networkPeers"])
	logger.Printf("   💰 Economics: %v", status["economicStatus"].(map[string]interface{})["totalSupply"])

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Printf("🛑 Shutting down LightChain L1...")

	// Graceful shutdown
	if err := l1chain.Stop(); err != nil {
		logger.Printf("❌ Error during shutdown: %v", err)
	}

	logger.Printf("✅ LightChain L1 stopped successfully")
}

func generateNodeIdentity() (common.Address, []byte) {
	// Generate a random private key (32 bytes)
	privateKey := make([]byte, 32)
	rand.Read(privateKey)
	
	// Generate address from private key (simplified)
	address := common.BytesToAddress(privateKey[:20])
	
	return address, privateKey
}

func isValidNodeType(nodeType string) bool {
	validTypes := []string{"validator", "full", "light"}
	for _, validType := range validTypes {
		if nodeType == validType {
			return true
		}
	}
	return false
}

func setupLogger(level string) *log.Logger {
	// For now, use standard logger. In production, use structured logging like zap or logrus
	logger := log.New(os.Stdout, fmt.Sprintf("[%s] ", appName), log.LstdFlags|log.Lshortfile)
	return logger
}

// getRPCPort returns the RPC port based on node type
func getRPCPort(nodeType string) int {
	switch nodeType {
	case "validator":
		return 8545
	case "sequencer":
		return 8555
	case "archive":
		return 8565
	default:
		return 8545
	}
}

// getWSPort returns the WebSocket port based on node type
func getWSPort(nodeType string) int {
	switch nodeType {
	case "validator":
		return 8546
	case "sequencer":
		return 8556
	case "archive":
		return 8566
	default:
		return 8546
	}
}

func init() {
	// Set the working directory to the project root if running from cmd/lightchain
	if wd, err := os.Getwd(); err == nil {
		if filepath.Base(wd) == "lightchain" {
			if err := os.Chdir("../.."); err != nil {
				log.Printf("Warning: Failed to change to project root: %v", err)
			}
		}
	}
}
