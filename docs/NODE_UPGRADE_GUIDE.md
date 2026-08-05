# 📖 Node Operator Upgrade & Release Guide

This guide explains how node operators (validators, full nodes, and sequencers) upgrade their running Litechain nodes when a new binary release is published.

---

## 🚦 **1. Understanding Release Types: Hotfix vs. Hard Fork**

Litechain releases are classified into 2 distinct types:

### 🟡 **A. Hotfix / Minor Release (Soft Upgrade)**
- **Examples**: `v2.0.1`, `v2.1.0`.
- **Nature**: Non-breaking performance updates, RPC bug fixes, mempool optimizations.
- **Node Action**: **Optional / Flexible schedule**. Nodes can upgrade at any time without risking chain splits or block rejection.

### 🔴 **B. Hard Fork Release (Height-Bound Protocol Upgrade)**
- **Examples**: `v2.0.0`, `v3.0.0`.
- **Nature**: State-transition breaking changes (e.g. ZK circuits, Block-STM execution rules, precompiles, consensus rules).
- **Node Action**: **Mandatory before Target Block Height ($H_{fork}$)**.
  - Node operators **MUST** update their node binary before the network reaches block height $H_{fork}$.
  - Nodes running an old binary when block $H_{fork}$ is reached will halt block execution safely with a fatal log warning (`ErrBinaryVersionTooOld`) to prevent corrupting state.

---

## 🛠️ **2. Upgrade Methods for Node Operators**

Node operators can upgrade their nodes using any of the following 3 methods:

---

### 🅰️ **Method A: Zero-Downtime Auto-Updater (Recommended)**

Litechain nodes include a built-in `AutoUpdateManager` (`pkg/version/autoupdate.go`) that downloads and stages binaries prior to the hard fork height.

```bash
# 1. Download and stage new binary archive into nodes/dir/binaries/
mkdir -p ./data/binaries
cp dist/v2.0.0/lightchain-v2.0.0-linux-amd64 ./data/binaries/lightchain-2.0.0
chmod +x ./data/binaries/lightchain-2.0.0

# 2. Verify SHA256 checksum
shasum -a 256 ./data/binaries/lightchain-2.0.0

# 3. Apply symlink swap
ln -sfn ./data/binaries/lightchain-2.0.0 ./data/current
```
The node running `AutoUpdateManager` automatically executes the binary swap cleanly upon reaching the hard fork height.

---

### 🅱️ **Method B: Docker Container Upgrade**

If you run nodes via Docker or Docker Compose:

#### **Hotfix Upgrade (Soft Upgrade)**
```bash
# Pull updated docker image and restart container
docker compose pull
docker compose up -d --remove-orphans
```

#### **Hard Fork Upgrade (Height-Bound)**
1. Check scheduled hard fork block height $H_{fork}$ (announced on GitHub Releases / Discord).
2. Prior to $H_{fork}$, update your `docker-compose.yml` to point to the new release tag:
   ```yaml
   image: lightchain-l1:v2.0.0
   ```
3. Restart container before height $H_{fork}$:
   ```bash
   docker compose up -d
   ```

---

### Ⓒ **Method C: Manual Binary Swap (Systemd / Bare-Metal)**

If you run Litechain as a systemd background service:

```bash
# 1. Stop running node service
sudo systemctl stop lightchain

# 2. Download and replace binary
curl -L -O https://github.com/sanketsaagar/Litechain/releases/download/v2.0.0/lightchain-v2.0.0-linux-amd64.tar.gz
tar -xzf lightchain-v2.0.0-linux-amd64.tar.gz
sudo cp lightchain-v2.0.0-linux-amd64/lightchain /usr/local/bin/lightchain

# 3. Verify new version
lightchain --version
# Output: Litechain L1 Node Version: 2.0.0

# 4. Restart service
sudo systemctl start lightchain

# 5. Check logs
journalctl -u lightchain -f
```

---

## 🔐 **3. Verification & Safety Checklist**

Before starting a node after an upgrade:
1. Verify SHA256 hash against official `checksums.txt` published on GitHub Releases.
2. Confirm `./build/lightchain --version` displays the correct release version.
3. Check node logs for `🌟 Starting Litechain L1 v2.0.0`.
