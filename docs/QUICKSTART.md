# 🚀 LightChain L2 Quick Start Guide

**Get your unified blockchain running in under 5 minutes!**

This guide will help you get started with LightChain L2 in under 10 minutes.

## Prerequisites

Before you begin, ensure you have:
- **Go 1.21+** installed ([Download](https://golang.org/dl/))
- **Docker** (optional, for containerized deployment)
- **Git** for version control
- **4 GB RAM** minimum
- **10 GB disk space** minimum

## Quick Setup

### 1. Clone and Setup

```bash
git clone https://github.com/Hypasis/Litechain.git
cd Litechain

# Run the automated setup script
./scripts/setup-dev.sh
```

The setup script will:
- Check system requirements
- Install Go dependencies
- Create necessary directories
- Generate example keys
- Build the project
- Run initial tests

### 2. Start a Single Node

Start a validator node for development:

```bash
make run-validator
```

This will start a validator node with:
- **RPC**: http://localhost:8545
- **WebSocket**: ws://localhost:8546
- **Metrics**: http://localhost:9090/metrics

### 3. Verify Installation

Test the node is working:

```bash
# Check node version
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}' \
  http://localhost:8545

# Check network ID
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"net_version","params":[],"id":1}' \
  http://localhost:8545
```

## Development Network Options

### Option 1: Docker Compose (Recommended)

Start a complete multi-node network:

```bash
docker-compose up -d
```

This provides:
- 2 Validator nodes
- 1 Sequencer node  
- 1 Archive node
- PostgreSQL database
- Prometheus monitoring
- Grafana dashboard
- Load balancer

Access points:
- **RPC Load Balancer**: http://localhost:8545
- **Grafana Dashboard**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090

### Option 2: Kurtosis (Advanced)

For a more realistic devnet environment:

```bash
# Install Kurtosis first
make dev-network
```

### Option 3: Manual Multi-Node Setup

Start individual nodes manually:

```bash
# Terminal 1 - Validator
make run-validator

# Terminal 2 - Sequencer
make run-sequencer

# Terminal 3 - Archive
make run-archive
```

## Basic Operations

### Create a Transaction

```bash
# Example using curl
curl -X POST -H "Content-Type: application/json" \
  --data '{
    "jsonrpc":"2.0",
    "method":"eth_sendTransaction",
    "params":[{
      "from": "0x742A4D1A0Ac05A73A48F10C2E2d6b0E3f1b2e3F4",
      "to": "0x8B3A4D1A0Ac05A73A48F10C2E2d6b0E3f1b2e3F5",
      "value": "0x9184e72a000"
    }],
    "id":1
  }' \
  http://localhost:8545
```

### Check Balance

```bash
curl -X POST -H "Content-Type: application/json" \
  --data '{
    "jsonrpc":"2.0",
    "method":"eth_getBalance",
    "params":["0x742A4D1A0Ac05A73A48F10C2E2d6b0E3f1b2e3F4", "latest"],
    "id":1
  }' \
  http://localhost:8545
```

### View Blocks

```bash
# Latest block number
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:8545

# Get block details
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", true],"id":1}' \
  http://localhost:8545
```

## Configuration

### Environment Variables

Create a `.env` file:

```bash
cp .env.example .env
```

Key configurations:
```env
NODE_TYPE=validator
DATA_DIR=./data
LOG_LEVEL=info
RPC_LISTEN_ADDR=127.0.0.1:8545
AGGLAYER_ENABLED=true
```

### Custom Configuration

Edit configuration files in `configs/`:
- `validator.yaml` - Validator node settings
- `sequencer.yaml` - Sequencer node settings  
- `archive.yaml` - Archive node settings
- `genesis.yaml` - Genesis block configuration

## Common Commands

```bash
# Build project
make build

# Run tests
make test

# Clean and rebuild
make clean build

# Format code
make format

# Run linter
make lint

# Generate coverage report
make test-coverage

# Build Docker image
make docker-build

# View logs
make logs-validator
make logs-sequencer
make logs-archive

# Reset database
make db-reset

# Generate new keys
make generate-keys
```

## Troubleshooting

### Port Already in Use

If you get port conflicts:

```bash
# Check what's using the port
lsof -i :8545

# Use different ports in config
sed -i 's/8545/8555/g' configs/validator.yaml
```

### Database Issues

```bash
# Reset the database
make db-reset

# Check database status
ls -la data/*/state/
```

### Build Failures

```bash
# Clean and retry
make clean
go mod tidy
make build
```

### Network Connectivity

```bash
# Check peer connections
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}' \
  http://localhost:8545
```

## Next Steps

1. **Read the Architecture**: See `docs/ARCHITECTURE.md` for detailed system design
2. **Explore Examples**: Check `examples/` directory for sample applications
3. **Join the Community**: Connect with other developers
4. **Contribute**: Help improve LightChain L2

## API Reference

### JSON-RPC Methods

LightChain L2 supports standard Ethereum JSON-RPC methods plus custom extensions:

#### Standard Methods
- `eth_*` - Ethereum compatibility
- `net_*` - Network information  
- `web3_*` - Web3 utilities

#### LightChain Extensions
- `lightchain_getValidators` - Get validator set
- `lightchain_getStake` - Get staking information
- `lightchain_getBridgeState` - Get bridge status

### WebSocket Events

Subscribe to real-time events:

```javascript
const ws = new WebSocket('ws://localhost:8546');

// Subscribe to new blocks
ws.send(JSON.stringify({
  id: 1,
  method: "eth_subscribe",
  params: ["newHeads"]
}));
```

## Performance Tips

1. **Use Archive Nodes**: For historical queries
2. **Enable Caching**: Set appropriate cache sizes
3. **Optimize Networking**: Configure peer limits
4. **Monitor Resources**: Use Grafana dashboards
5. **Batch Requests**: Use batch JSON-RPC calls

## Security Notes

⚠️ **Development Only**: The provided keys and configuration are for development only. Never use them in production.

For production deployment:
- Generate secure keys
- Use proper authentication
- Enable TLS encryption
- Configure firewalls
- Regular security audits

## Support

- **Documentation**: `docs/` directory
- **Issues**: GitHub Issues
- **Community**: Discord/Telegram
- **Email**: support@lightchain.network

Happy building with LightChain L2! 🚀
