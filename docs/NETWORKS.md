# 🌐 LightChain Network Reference

LightChain L1 operates two distinct networks for different purposes.

## 📋 **Network Overview**

| **Network** | **Name** | **Chain ID** | **Purpose** | **Status** | **RPC URL** |
|-------------|----------|--------------|-------------|------------|-------------|
| **Mainnet** | **LightChain** | 1001 | Production | 🚀 Live | `https://rpc.lightchain.network` |
| **Testnet** | **LightBeam** | 1337 | Testing & Development | ⚡ Active | `https://testnet-rpc.lightchain.network` |

---

## 🎯 **LightChain Mainnet (Chain ID: 1001)**

### **Network Details**
- **Official Name**: `LightChain`
- **Chain ID**: `1001` 
- **Network ID**: `1001`
- **Currency**: `LIGHT`
- **Block Time**: `2 seconds`
- **Consensus**: `HPoS (Hybrid Proof-of-Stake)`

### **RPC Endpoints**
```bash
# Primary RPC
https://rpc.lightchain.network

# WebSocket
wss://ws.lightchain.network

# Archive Node (for historical queries)
https://archive-rpc.lightchain.network
```

### **Explorer**
```bash
# Blockchain Explorer
https://explorer.lightchain.network

# API Endpoints
https://api.lightchain.network
```

### **Connect to LightChain Mainnet**

#### **MetaMask Configuration**
```
Network Name: LightChain
RPC URL: https://rpc.lightchain.network
Chain ID: 1001
Symbol: LIGHT
Block Explorer: https://explorer.lightchain.network
```

#### **CLI Connection**
```bash
# Connect to mainnet
lightchain-cli --chain-id 1001 --node https://rpc.lightchain.network

# Check balance on mainnet
lightchain-cli balance --address 0xYourAddress --chain-id 1001

# Send transaction on mainnet
lightchain-cli tx send --to 0xRecipient --amount 100 --chain-id 1001
```

#### **SDK Configuration**
```javascript
import { LightChainSDK } from '@lightchain/sdk';

const sdk = new LightChainSDK({
  nodeUrl: 'https://rpc.lightchain.network',
  chainId: 1001,
  privateKey: process.env.PRIVATE_KEY // Your mainnet private key
});
```

---

## ⚡ **LightBeam Testnet (Chain ID: 1337)**

### **Network Details**
- **Official Name**: `LightBeam`
- **Chain ID**: `1337`
- **Network ID**: `1337`  
- **Currency**: `LIGHT` (test tokens)
- **Block Time**: `2 seconds`
- **Consensus**: `HPoS (Hybrid Proof-of-Stake)`

### **RPC Endpoints**
```bash
# Primary RPC
https://testnet-rpc.lightchain.network

# WebSocket
wss://testnet-ws.lightchain.network

# Local Development
http://localhost:8545 (when running locally)
```

### **Explorer & Tools**
```bash
# Testnet Explorer
https://testnet-explorer.lightchain.network

# Faucet for Test Tokens
https://faucet.lightchain.network

# Test API
https://testnet-api.lightchain.network
```

### **Get Test Tokens**
```bash
# Using CLI faucet
lightchain-cli faucet --address 0xYourAddress --chain-id 1337

# Using web faucet
curl -X POST https://faucet.lightchain.network/request \
  -H "Content-Type: application/json" \
  -d '{"address": "0xYourAddress"}'
```

### **Connect to LightBeam Testnet**

#### **MetaMask Configuration**
```
Network Name: LightBeam Testnet
RPC URL: https://testnet-rpc.lightchain.network
Chain ID: 1337
Symbol: LIGHT
Block Explorer: https://testnet-explorer.lightchain.network
```

#### **CLI Connection**
```bash
# Connect to testnet
lightchain-cli --chain-id 1337 --node https://testnet-rpc.lightchain.network

# Get test tokens from faucet
lightchain-cli faucet --address 0xYourAddress

# Deploy contract to testnet
lightchain-cli contract deploy MyContract.sol --chain-id 1337
```

#### **SDK Configuration**
```javascript
import { LightChainSDK } from '@lightchain/sdk';

const sdk = new LightChainSDK({
  nodeUrl: 'https://testnet-rpc.lightchain.network',
  chainId: 1337,
  privateKey: process.env.TESTNET_PRIVATE_KEY // Your testnet private key
});
```

---

## 🏗️ **Local Development**

### **Start Local LightBeam Network**
```bash
# Using Docker
./scripts/network-lifecycle.sh start

# Using Kurtosis  
./scripts/kurtosis-manager.sh start 3 2 1

# Manual start
./lightchain --chain-id 1337 --genesis configs/genesis.yaml
```

### **Local Network Details**
- **Chain ID**: `1337` (same as LightBeam testnet)
- **RPC**: `http://localhost:8545`
- **WebSocket**: `ws://localhost:8546`
- **Grafana Dashboard**: `http://localhost:3000`

---

## 🌉 **Cross-Chain Bridges**

### **Supported Networks**
LightChain L1 has native bridges to:

| **Network** | **Bridge Fee** | **Time** | **Status** |
|-------------|----------------|----------|------------|
| **Ethereum** | 0.3% | ~15 min | ✅ Active |
| **Polygon** | 0.2% | ~5 min | ✅ Active |
| **Arbitrum** | 0.25% | ~2 min | ✅ Active |
| **Optimism** | 0.25% | ~2 min | ✅ Active |
| **BSC** | 0.2% | ~3 min | ✅ Active |
| **Avalanche** | 0.2% | ~1 min | ✅ Active |

### **Bridge Usage**
```bash
# Bridge ETH to LightChain
lightchain-cli bridge deposit --from ethereum --amount 1.0 --token ETH

# Bridge USDC from Polygon
lightchain-cli bridge deposit --from polygon --amount 1000 --token USDC

# Check bridge status
lightchain-cli bridge status --tx-hash 0xYourTxHash
```

---

## 🔧 **Network Configuration Files**

### **Genesis Files**
```bash
configs/
├── genesis.yaml          # LightBeam testnet genesis
└── genesis-mainnet.yaml  # LightChain mainnet genesis
```

### **Node Configuration**
```bash
configs/
├── validator.yaml        # Validator node config
├── fullnode.yaml        # Full node config
└── archive.yaml         # Archive node config (auto-generated)
```

---

## 📊 **Network Statistics**

### **LightChain Mainnet**
- **Total Supply**: 1,000,000,000 LIGHT
- **Circulating Supply**: ~500,000,000 LIGHT
- **Active Validators**: 21 (maximum)
- **Block Height**: [Live Counter](https://explorer.lightchain.network)
- **TPS**: ~500 transactions per second
- **Average Gas Price**: ~1 Gwei

### **LightBeam Testnet**
- **Test Token Supply**: Unlimited (via faucet)
- **Active Validators**: 3-5 (testing)
- **Block Height**: [Testnet Counter](https://testnet-explorer.lightchain.network)
- **Reset Schedule**: Monthly (for testing)

---

## 🚨 **Emergency Contacts**

### **Network Issues**
- **Discord**: [discord.gg/lightchain](https://discord.gg/lightchain)
- **Twitter**: [@LightChainL1](https://twitter.com/LightChainL1)
- **Email**: support@lightchain.network

### **Developer Support**
- **GitHub Issues**: [github.com/Hypasis/Litechain/issues](https://github.com/Hypasis/Litechain/issues)
- **Documentation**: [docs.lightchain.network](https://docs.lightchain.network)
- **Developer Discord**: #dev-support channel

---

## 🔄 **Network Upgrades**

### **Upcoming Upgrades**
- **Developer Incentives v2**: Enhanced reward system
- **Universal Bridge v2**: More chains, lower fees  
- **EVM Compatibility++**: Advanced Solidity features

### **Upgrade Schedule**
Network upgrades are announced 30 days in advance via:
- Official Discord announcements
- GitHub releases
- Explorer notifications
- Node operator emails

---

**Choose your network and start building on LightChain L1!** 🚀

- **Want to test?** → Use **LightBeam Testnet** (Chain ID 1337)
- **Ready for production?** → Deploy on **LightChain Mainnet** (Chain ID 1001)