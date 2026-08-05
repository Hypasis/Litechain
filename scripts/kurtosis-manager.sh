#!/bin/bash
# LightBeam Testnet Kurtosis Manager
# Comprehensive management tool for Kurtosis-based development environment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
ENCLAVE_NAME="lightbeam-testnet"
PACKAGE_PATH="github.com/Hypasis/Litechain/deployments/kurtosis"
LOCAL_PACKAGE_PATH="./deployments/kurtosis"
KURTOSIS_VERSION="0.89.0"

# Logging function
log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    case $level in
        "INFO")  echo -e "${GREEN}[${timestamp}] INFO:${NC} $message" ;;
        "WARN")  echo -e "${YELLOW}[${timestamp}] WARN:${NC} $message" ;;
        "ERROR") echo -e "${RED}[${timestamp}] ERROR:${NC} $message" ;;
        "DEBUG") echo -e "${BLUE}[${timestamp}] DEBUG:${NC} $message" ;;
        "SUCCESS") echo -e "${CYAN}[${timestamp}] SUCCESS:${NC} $message" ;;
        *)       echo -e "${PURPLE}[${timestamp}] $level:${NC} $message" ;;
    esac
}

# Check if Kurtosis is installed
check_kurtosis() {
    if ! command -v kurtosis &> /dev/null; then
        log "ERROR" "Kurtosis is not installed. Please install Kurtosis first."
        log "INFO" "Installation: curl -fsSL https://docs.kurtosis.com/install.sh | bash"
        exit 1
    fi
    
    local version=$(kurtosis version | grep "CLI Version" | awk '{print $3}' | sed 's/v//')
    log "INFO" "Kurtosis CLI Version: $version"
}

# Check if Docker is running
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        log "ERROR" "Docker is not running. Please start Docker first."
        exit 1
    fi
    log "INFO" "Docker is running"
}

# Start the devnet
start_devnet() {
    log "INFO" "🚀 Starting LightBeam Testnet..."
    
    # Parse arguments
    local validators=${1:-3}
    local fullnodes=${2:-2}
    local archives=${3:-1}
    local enable_monitoring=${4:-true}
    local enable_tx_generation=${5:-true}
    
    log "INFO" "Configuration:"
    log "INFO" "  • Validators: $validators"
    log "INFO" "  • Full Nodes: $fullnodes"
    log "INFO" "  • Archives: $archives"
    log "INFO" "  • Monitoring: $enable_monitoring"
    log "INFO" "  • Transaction Generation: $enable_tx_generation"
    
    # Create enclave if it doesn't exist
    if ! kurtosis enclave ls | grep -q "$ENCLAVE_NAME"; then
        log "INFO" "Creating new enclave: $ENCLAVE_NAME"
        kurtosis enclave add "$ENCLAVE_NAME"
    fi
    
    # Run the package
    local args_json=$(cat <<EOF
{
    "validators": $validators,
    "fullnodes": $fullnodes,
    "archives": $archives,
    "monitoring": $enable_monitoring,
    "enable_tx_generation": $enable_tx_generation
}
EOF
)
    
    log "INFO" "Deploying LightBeam testnet..."
    if kurtosis run --enclave "$ENCLAVE_NAME" "$LOCAL_PACKAGE_PATH" --args "$args_json"; then
        log "SUCCESS" "LightBeam Testnet started successfully!"
        show_access_points
    else
        log "ERROR" "Failed to start LightChain L2 DevNet"
        exit 1
    fi
}

# Stop the devnet
stop_devnet() {
    log "INFO" "🛑 Stopping LightChain L2 Kurtosis DevNet..."
    
    if kurtosis enclave ls | grep -q "$ENCLAVE_NAME"; then
        kurtosis enclave stop "$ENCLAVE_NAME"
        log "SUCCESS" "DevNet stopped successfully"
    else
        log "WARN" "DevNet enclave '$ENCLAVE_NAME' not found"
    fi
}

# Clean/remove the devnet
clean_devnet() {
    log "INFO" "🧹 Cleaning LightChain L2 Kurtosis DevNet..."
    
    if kurtosis enclave ls | grep -q "$ENCLAVE_NAME"; then
        kurtosis enclave rm "$ENCLAVE_NAME" --force
        log "SUCCESS" "DevNet cleaned successfully"
    else
        log "WARN" "DevNet enclave '$ENCLAVE_NAME' not found"
    fi
}

# Restart the devnet
restart_devnet() {
    log "INFO" "🔄 Restarting LightChain L2 Kurtosis DevNet..."
    stop_devnet
    sleep 5
    start_devnet "$@"
}

# Show status
show_status() {
    log "INFO" "📊 LightChain L2 DevNet Status"
    echo ""
    
    if kurtosis enclave ls | grep -q "$ENCLAVE_NAME"; then
        echo -e "${BLUE}=== Enclave Status ===${NC}"
        kurtosis enclave ls | grep "$ENCLAVE_NAME" || echo "Enclave not found"
        
        echo -e "\n${BLUE}=== Services Status ===${NC}"
        kurtosis service ls "$ENCLAVE_NAME" 2>/dev/null || echo "No services found"
        
        echo -e "\n${BLUE}=== Port Mappings ===${NC}"
        kurtosis port ls "$ENCLAVE_NAME" 2>/dev/null || echo "No ports found"
        
    else
        log "WARN" "DevNet enclave '$ENCLAVE_NAME' not found"
        log "INFO" "Use 'start' command to create and start the devnet"
    fi
}

# Show access points
show_access_points() {
    log "INFO" "🌐 LightChain L2 DevNet Access Points:"
    echo ""
    
    if kurtosis enclave ls | grep -q "$ENCLAVE_NAME"; then
        # Try to get port mappings
        local rpc_port=$(kurtosis port print "$ENCLAVE_NAME" load-balancer 8545 2>/dev/null | grep -o '[0-9]*' | tail -1)
        local ws_port=$(kurtosis port print "$ENCLAVE_NAME" load-balancer 8546 2>/dev/null | grep -o '[0-9]*' | tail -1)
        local grafana_port=$(kurtosis port print "$ENCLAVE_NAME" grafana 3000 2>/dev/null | grep -o '[0-9]*' | tail -1)
        local prometheus_port=$(kurtosis port print "$ENCLAVE_NAME" prometheus 9090 2>/dev/null | grep -o '[0-9]*' | tail -1)
        
        echo -e "${CYAN}🔗 Primary Endpoints:${NC}"
        if [ ! -z "$rpc_port" ]; then
            echo -e "   • RPC: ${GREEN}http://localhost:$rpc_port${NC}"
        fi
        if [ ! -z "$ws_port" ]; then
            echo -e "   • WebSocket: ${GREEN}ws://localhost:$ws_port${NC}"
        fi
        
        echo -e "\n${CYAN}📊 Monitoring:${NC}"
        if [ ! -z "$grafana_port" ]; then
            echo -e "   • Grafana: ${GREEN}http://localhost:$grafana_port${NC} (admin/admin123)"
        fi
        if [ ! -z "$prometheus_port" ]; then
            echo -e "   • Prometheus: ${GREEN}http://localhost:$prometheus_port${NC}"
        fi
        
        echo -e "\n${CYAN}🎮 Management Commands:${NC}"
        echo -e "   • Status: ${YELLOW}./scripts/kurtosis-manager.sh status${NC}"
        echo -e "   • Logs: ${YELLOW}./scripts/kurtosis-manager.sh logs${NC}"
        echo -e "   • Upgrade: ${YELLOW}./scripts/kurtosis-manager.sh upgrade${NC}"
        
    else
        log "WARN" "DevNet not running. Start it first with: ./scripts/kurtosis-manager.sh start"
    fi
}

# Show logs
show_logs() {
    local service=${1:-""}
    
    if [ -z "$service" ]; then
        log "INFO" "📋 Available services:"
        kurtosis service ls "$ENCLAVE_NAME" 2>/dev/null | awk 'NR>1 {print "   • " $1}' || echo "No services found"
        echo ""
        log "INFO" "Usage: ./scripts/kurtosis-manager.sh logs <service_name>"
        log "INFO" "   or: ./scripts/kurtosis-manager.sh logs all"
        return
    fi
    
    if [ "$service" = "all" ]; then
        log "INFO" "📋 Showing logs for all services..."
        kurtosis service logs "$ENCLAVE_NAME" --follow
    else
        log "INFO" "📋 Showing logs for service: $service"
        kurtosis service logs "$ENCLAVE_NAME" "$service" --follow
    fi
}

# Trigger network upgrade
trigger_upgrade() {
    log "INFO" "🔄 Triggering network upgrade..."
    
    if kurtosis service exec "$ENCLAVE_NAME" lifecycle-manager python /app/lifecycle_manager.py trigger_upgrade; then
        log "SUCCESS" "Network upgrade triggered successfully"
        log "INFO" "Monitor the upgrade process with: ./scripts/kurtosis-manager.sh logs lifecycle-manager"
    else
        log "ERROR" "Failed to trigger network upgrade"
        exit 1
    fi
}

# Run shell in service
shell() {
    local service=${1:-"validator-0"}
    
    log "INFO" "🐚 Opening shell in service: $service"
    kurtosis service shell "$ENCLAVE_NAME" "$service"
}

# Test blockchain functionality
test_blockchain() {
    log "INFO" "🧪 Testing blockchain functionality..."
    
    # Get RPC endpoint
    local rpc_port=$(kurtosis port print "$ENCLAVE_NAME" load-balancer 8545 2>/dev/null | grep -o '[0-9]*' | tail -1)
    
    if [ -z "$rpc_port" ]; then
        log "ERROR" "Could not find RPC port. Is the devnet running?"
        exit 1
    fi
    
    local rpc_url="http://localhost:$rpc_port"
    log "INFO" "Testing RPC endpoint: $rpc_url"
    
    # Test basic RPC calls
    echo -e "\n${BLUE}=== Testing Basic RPC Calls ===${NC}"
    
    # Test net_version
    local net_version=$(curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"net_version","params":[],"id":1}' \
        "$rpc_url" | jq -r '.result' 2>/dev/null || echo "failed")
    echo "• Network Version: $net_version"
    
    # Test eth_blockNumber
    local block_number=$(curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
        "$rpc_url" | jq -r '.result' 2>/dev/null || echo "failed")
    if [ "$block_number" != "failed" ]; then
        local block_decimal=$((16#${block_number#0x}))
        echo "• Current Block: #$block_decimal ($block_number)"
    else
        echo "• Current Block: failed"
    fi
    
    # Test web3_clientVersion
    local client_version=$(curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}' \
        "$rpc_url" | jq -r '.result' 2>/dev/null || echo "failed")
    echo "• Client Version: $client_version"
    
    if [ "$net_version" != "failed" ] && [ "$block_number" != "failed" ]; then
        log "SUCCESS" "Blockchain is responding correctly!"
    else
        log "WARN" "Some RPC calls failed. Check logs for more details."
    fi
}

# Compare Docker vs Kurtosis
compare_environments() {
    log "INFO" "🔍 Docker vs Kurtosis Comparison for LightChain L2"
    echo ""
    
    cat << 'EOF'
╔═══════════════════════════════════════════════════════════════╗
║                   DOCKER VS KURTOSIS COMPARISON              ║
╠═══════════════════════════════════════════════════════════════╣
║                                                               ║
║ 🐳 DOCKER COMPOSE                                            ║
║ ✅ Simple configuration (docker-compose.yml)                 ║
║ ✅ Fast startup and shutdown                                  ║
║ ✅ Direct port access (localhost:8545)                       ║
║ ✅ Easy volume mounting                                       ║
║ ✅ Built-in networking                                        ║
║ ❌ Limited multi-environment support                          ║
║ ❌ Less sophisticated orchestration                           ║
║                                                               ║
║ 🎯 KURTOSIS                                                   ║
║ ✅ Advanced multi-network testing                             ║
║ ✅ Sophisticated service orchestration                        ║
║ ✅ Built-in testing framework                                 ║
║ ✅ Dynamic port allocation                                     ║
║ ✅ Better environment isolation                               ║
║ ✅ Professional dev tooling                                   ║
║ ❌ More complex setup                                          ║
║ ❌ Learning curve                                              ║
║                                                               ║
║ 🎪 BOTH SUPPORT:                                              ║
║ • Continuous block mining                                     ║
║ • Automatic transaction generation                            ║
║ • Multi-node networks (validators, sequencers, archives)     ║
║ • Monitoring (Grafana + Prometheus)                           ║
║ • Network lifecycle management                                ║
║ • Graceful upgrades                                           ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
EOF
    
    echo ""
    log "INFO" "🎯 Use Docker for: Quick local development, simple testing"
    log "INFO" "🎯 Use Kurtosis for: Professional testing, complex scenarios, team environments"
}

# Show help
show_help() {
    cat << EOF
${BLUE}LightChain L2 Kurtosis DevNet Manager${NC}

${YELLOW}USAGE:${NC}
    $0 [COMMAND] [OPTIONS]

${YELLOW}COMMANDS:${NC}
    ${GREEN}start [validators] [sequencers] [archives] [monitoring] [tx-gen]${NC}
        Start the LightChain L2 DevNet
        Default: 3 validators, 1 sequencer, 1 archive, monitoring=true, tx-gen=true
        
    ${GREEN}stop${NC}
        Stop the DevNet (preserves data)
        
    ${GREEN}clean${NC}
        Clean/remove the DevNet completely
        
    ${GREEN}restart [options]${NC}
        Restart the DevNet with same or new options
        
    ${GREEN}status${NC}
        Show DevNet status and service health
        
    ${GREEN}access${NC}
        Show access points and port mappings
        
    ${GREEN}logs [service|all]${NC}
        Show logs for specific service or all services
        
    ${GREEN}upgrade${NC}
        Trigger graceful network upgrade
        
    ${GREEN}shell [service]${NC}
        Open shell in a service (default: validator-0)
        
    ${GREEN}test${NC}
        Test blockchain functionality and RPC endpoints
        
    ${GREEN}compare${NC}
        Compare Docker vs Kurtosis environments
        
    ${GREEN}help${NC}
        Show this help message

${YELLOW}EXAMPLES:${NC}
    # Start default devnet
    $0 start
    
    # Start with custom configuration
    $0 start 5 2 1 true true
    
    # View all logs
    $0 logs all
    
    # View specific service logs
    $0 logs validator-0
    
    # Trigger network upgrade
    $0 upgrade
    
    # Test blockchain
    $0 test

${YELLOW}MANAGEMENT:${NC}
    • DevNet runs until manually stopped or upgraded
    • Auto-mining generates blocks every 1-2 seconds
    • Transaction generator creates realistic activity
    • Monitoring available via Grafana dashboard
    • All data persists between restarts (unless cleaned)

${YELLOW}ACCESS POINTS:${NC}
    • RPC: Dynamic port (check with 'access' command)
    • Grafana: Dynamic port (admin/admin123)
    • Prometheus: Dynamic port
    • Use 'kurtosis port print $ENCLAVE_NAME <service> <port>' for specific mappings
EOF
}

# Main command dispatcher
main() {
    check_kurtosis
    check_docker
    
    case "${1:-help}" in
        "start")
            start_devnet "${2:-3}" "${3:-1}" "${4:-1}" "${5:-true}" "${6:-true}"
            ;;
        "stop")
            stop_devnet
            ;;
        "clean")
            clean_devnet
            ;;
        "restart")
            restart_devnet "${2:-3}" "${3:-1}" "${4:-1}" "${5:-true}" "${6:-true}"
            ;;
        "status")
            show_status
            ;;
        "access")
            show_access_points
            ;;
        "logs")
            show_logs "${2:-}"
            ;;
        "upgrade")
            trigger_upgrade
            ;;
        "shell")
            shell "${2:-validator-0}"
            ;;
        "test")
            test_blockchain
            ;;
        "compare")
            compare_environments
            ;;
        "help"|*)
            show_help
            ;;
    esac
}

# Run main function
main "$@"
