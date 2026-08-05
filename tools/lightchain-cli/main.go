package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sanketsaagar/lightchain-l1/pkg/sdk"
	"github.com/sanketsaagar/lightchain-l1/pkg/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version and build metadata",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.FormattedVersion())
	},
}

var (
	nodeURL    string
	privateKey string
	chainID    int64
	cliSDK     *sdk.LightChainSDK
)

var rootCmd = &cobra.Command{
	Use:   "lightchain-cli",
	Short: "LightChain L1 Command Line Interface",
	Long:  "A developer-friendly CLI for interacting with LightChain L1 blockchain",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize SDK
		config := sdk.SDKConfig{
			NodeURL:    nodeURL,
			PrivateKey: privateKey,
			ChainID:    chainID,
		}
		
		var err error
		cliSDK, err = sdk.NewSDK(config)
		if err != nil {
			log.Fatalf("Failed to initialize SDK: %v", err)
		}
	},
}

// Account commands
var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Account management commands",
}

var createAccountCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new account",
	Run: func(cmd *cobra.Command, args []string) {
		account, err := cliSDK.CreateAccount()
		if err != nil {
			log.Fatalf("Failed to create account: %v", err)
		}
		
		fmt.Printf("✅ Account created successfully!\n")
		fmt.Printf("Address: %s\n", account.Address.Hex())
		fmt.Printf("Private Key: %x\n", account.PrivateKey.D.Bytes())
		fmt.Printf("⚠️  Save your private key securely!\n")
	},
}

var balanceCmd = &cobra.Command{
	Use:   "balance <address>",
	Short: "Get account balance",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		address := common.HexToAddress(args[0])
		balance, err := cliSDK.GetBalance(address)
		if err != nil {
			log.Fatalf("Failed to get balance: %v", err)
		}
		
		balanceEther := cliSDK.FromWei(balance)
		fmt.Printf("Balance: %.6f LIGHT (%s wei)\n", balanceEther, balance.String())
	},
}

// Transaction commands
var txCmd = &cobra.Command{
	Use:   "tx",
	Short: "Transaction commands",
}

var sendCmd = &cobra.Command{
	Use:   "send <to> <amount>",
	Short: "Send tokens to an address",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		to := common.HexToAddress(args[0])
		amountStr := args[1]
		
		amount, ok := new(big.Float).SetString(amountStr)
		if !ok {
			log.Fatalf("Invalid amount: %s", amountStr)
		}
		
		amountFloat, _ := amount.Float64()
		amountWei := cliSDK.ToWei(amountFloat)
		
		tx, err := cliSDK.SendTransaction(to, amountWei, nil)
		if err != nil {
			log.Fatalf("Failed to send transaction: %v", err)
		}
		
		fmt.Printf("✅ Transaction sent!\n")
		fmt.Printf("Hash: %s\n", tx.Hash().Hex())
		fmt.Printf("To: %s\n", to.Hex())
		fmt.Printf("Amount: %s LIGHT\n", amountStr)
		
		// Wait for confirmation
		fmt.Println("⏳ Waiting for confirmation...")
		receipt, err := cliSDK.WaitForTransaction(tx.Hash())
		if err != nil {
			log.Printf("❌ Failed to get confirmation: %v", err)
			return
		}
		
		if receipt.Status == 1 {
			fmt.Printf("✅ Transaction confirmed in block %d\n", receipt.BlockNumber.Uint64())
		} else {
			fmt.Printf("❌ Transaction failed\n")
		}
	},
}

var receiptCmd = &cobra.Command{
	Use:   "receipt <txhash>",
	Short: "Get transaction receipt",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		txHash := common.HexToHash(args[0])
		receipt, err := cliSDK.GetTransactionReceipt(txHash)
		if err != nil {
			log.Fatalf("Failed to get receipt: %v", err)
		}
		
		fmt.Printf("Transaction Receipt:\n")
		fmt.Printf("  Hash: %s\n", receipt.TxHash.Hex())
		fmt.Printf("  Block: %d\n", receipt.BlockNumber.Uint64())
		fmt.Printf("  Status: %d\n", receipt.Status)
		fmt.Printf("  Gas Used: %d\n", receipt.GasUsed)
		
		if receipt.ContractAddress != (common.Address{}) {
			fmt.Printf("  Contract Address: %s\n", receipt.ContractAddress.Hex())
		}
	},
}

// Contract commands
var contractCmd = &cobra.Command{
	Use:   "contract",
	Short: "Smart contract commands",
}

var deployCmd = &cobra.Command{
	Use:   "deploy <bytecode>",
	Short: "Deploy a smart contract",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		bytecode := args[0]
		if !strings.HasPrefix(bytecode, "0x") {
			bytecode = "0x" + bytecode
		}
		
		// Simplified deployment
		deployment, err := cliSDK.DeployContract("", bytecode)
		if err != nil {
			log.Fatalf("Failed to deploy contract: %v", err)
		}
		
		fmt.Printf("✅ Contract deployed!\n")
		fmt.Printf("Transaction: %s\n", deployment.Transaction.Hash().Hex())
		fmt.Printf("Address: %s\n", deployment.Address.Hex())
	},
}

var callCmd = &cobra.Command{
	Use:   "call <address> <method> [params...]",
	Short: "Call a smart contract function",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		contractAddress := common.HexToAddress(args[0])
		methodName := args[1]
		params := args[2:]
		
		var callParams []interface{}
		for _, param := range params {
			callParams = append(callParams, param)
		}
		
		result, err := cliSDK.CallContract(contractAddress, methodName, callParams...)
		if err != nil {
			log.Fatalf("Failed to call contract: %v", err)
		}
		
		fmt.Printf("Result: 0x%x\n", result)
	},
}

// Staking commands
var stakeCmd = &cobra.Command{
	Use:   "stake",
	Short: "Staking commands",
}

var stakeTokensCmd = &cobra.Command{
	Use:   "deposit <amount>",
	Short: "Stake tokens to become a validator",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		amountStr := args[0]
		amount, ok := new(big.Float).SetString(amountStr)
		if !ok {
			log.Fatalf("Invalid amount: %s", amountStr)
		}
		
		amountFloat, _ := amount.Float64()
		amountWei := cliSDK.ToWei(amountFloat)
		
		tx, err := cliSDK.StakeTokens(amountWei)
		if err != nil {
			log.Fatalf("Failed to stake tokens: %v", err)
		}
		
		fmt.Printf("✅ Staking transaction sent!\n")
		fmt.Printf("Hash: %s\n", tx.Hash().Hex())
		fmt.Printf("Amount: %s LIGHT\n", amountStr)
	},
}

var stakingInfoCmd = &cobra.Command{
	Use:   "info <validator>",
	Short: "Get staking information for a validator",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		validator := common.HexToAddress(args[0])
		info, err := cliSDK.GetStakingInfo(validator)
		if err != nil {
			log.Fatalf("Failed to get staking info: %v", err)
		}
		
		fmt.Printf("Staking Information:\n")
		fmt.Printf("  Validator: %s\n", info.Validator.Hex())
		fmt.Printf("  Staked Amount: %.6f LIGHT\n", cliSDK.FromWei(info.StakedAmount))
		fmt.Printf("  Rewards: %.6f LIGHT\n", cliSDK.FromWei(info.Rewards))
		fmt.Printf("  Performance: %.2f%%\n", info.Performance*100)
		fmt.Printf("  Active: %v\n", info.IsActive)
	},
}

// Network commands
var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Network information commands",
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get network status",
	Run: func(cmd *cobra.Command, args []string) {
		// Get basic network info
		fmt.Printf("🌐 LightChain L1 Network Status\n")
		fmt.Printf("Node URL: %s\n", nodeURL)
		fmt.Printf("Chain ID: %d\n", chainID)
		
		// Could add more network stats here
		fmt.Printf("Status: ✅ Connected\n")
	},
}

// Utility commands
var utilCmd = &cobra.Command{
	Use:   "util",
	Short: "Utility commands",
}

var convertCmd = &cobra.Command{
	Use:   "convert <amount> <from> <to>",
	Short: "Convert between units (wei, gwei, ether)",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		amountStr := args[0]
		from := strings.ToLower(args[1])
		to := strings.ToLower(args[2])
		
		amount, ok := new(big.Float).SetString(amountStr)
		if !ok {
			log.Fatalf("Invalid amount: %s", amountStr)
		}
		
		var result *big.Float
		
		// Convert to wei first
		switch from {
		case "ether", "light":
			result = new(big.Float).Mul(amount, big.NewFloat(1e18))
		case "gwei":
			result = new(big.Float).Mul(amount, big.NewFloat(1e9))
		case "wei":
			result = amount
		default:
			log.Fatalf("Unknown unit: %s", from)
		}
		
		// Convert from wei to target
		switch to {
		case "ether", "light":
			result = new(big.Float).Quo(result, big.NewFloat(1e18))
		case "gwei":
			result = new(big.Float).Quo(result, big.NewFloat(1e9))
		case "wei":
			// Already in wei
		default:
			log.Fatalf("Unknown unit: %s", to)
		}
		
		fmt.Printf("%s %s = %s %s\n", amountStr, from, result.Text('f', -1), to)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration commands",
}

var initConfigCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		config := map[string]interface{}{
			"nodeUrl": "http://localhost:8545",
			"chainId": 1337,
		}
		
		configBytes, _ := json.MarshalIndent(config, "", "  ")
		
		configFile := "lightchain-config.json"
		err := os.WriteFile(configFile, configBytes, 0644)
		if err != nil {
			log.Fatalf("Failed to write config file: %v", err)
		}
		
		fmt.Printf("✅ Configuration file created: %s\n", configFile)
		fmt.Printf("Edit this file to customize your settings.\n")
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&nodeURL, "node", "http://localhost:8545", "Node URL")
	rootCmd.PersistentFlags().StringVar(&privateKey, "key", "", "Private key (optional)")
	rootCmd.PersistentFlags().Int64Var(&chainID, "chain-id", 1001, "Chain ID (1001=LightChain, 1337=LightBeam testnet)")
	
	// Add all command groups
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(txCmd)
	rootCmd.AddCommand(contractCmd)
	rootCmd.AddCommand(stakeCmd)
	rootCmd.AddCommand(networkCmd)
	rootCmd.AddCommand(utilCmd)
	rootCmd.AddCommand(configCmd)
	
	// Account commands
	accountCmd.AddCommand(createAccountCmd)
	accountCmd.AddCommand(balanceCmd)
	
	// Transaction commands
	txCmd.AddCommand(sendCmd)
	txCmd.AddCommand(receiptCmd)
	
	// Contract commands
	contractCmd.AddCommand(deployCmd)
	contractCmd.AddCommand(callCmd)
	
	// Staking commands
	stakeCmd.AddCommand(stakeTokensCmd)
	stakeCmd.AddCommand(stakingInfoCmd)
	
	// Network commands
	networkCmd.AddCommand(statusCmd)
	
	// Utility commands
	utilCmd.AddCommand(convertCmd)
	
	// Config commands
	configCmd.AddCommand(initConfigCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}