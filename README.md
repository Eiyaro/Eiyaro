# Eiyaro (エイヤロ)

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25.7-00ADD8.svg?logo=go)](https://go.dev)


**Eiyaro** is a high-performance decentralized blockchain node implementation featuring the innovative **NogoPow** Proof-of-Work algorithm.

## Features

- **Fast Block Time**: 20-second block interval for smooth transaction experience
- **NogoPow Algorithm**: ASIC-resistant PoW using 256x256 matrix computation
- **Mild Deflation**: Initial reward 8 EY, 10% annual reduction, 0.1 EY floor
- **Dynamic Difficulty**: Per-block difficulty adjustment for network stability
- **Production Ready**: Comprehensive test suite and CI/CD pipeline
- **GhostDAG Consensus**: PHANTOM-based directed acyclic graph protocol

## Installation

### Prerequisites

- Go 1.25.7 or later
- Git
- Windows / Linux / macOS

### Building from Source

```bash
# Windows
cd Eiyaro
build.bat

# Linux/macOS
cd Eiyaro
./build.sh

# Or use Make
make build
```

Compiled binaries:
- `eyarod` / `eyarod.exe` — Full node daemon
- `eiyaroctl` / `eiyaroctl.exe` — CLI RPC client
- `eiyarowallet` / `eiyarowallet.exe` — HD wallet
- `miner` / `miner.exe` — CPU miner
- `genkeypair` / `genkeypair.exe` — Key pair generator

### Docker

```bash
docker pull ghcr.io/hoosat-oy/eyarod:latest
```

## Quick Start

### Start Mainnet Node

```bash
./eyarod --utxoindex
```

### Start Testnet Node

```bash
./eyarod --testnet
```

### Generate a Wallet Address

```bash
./eiyarowallet create
```

### Start Mining

```bash
./miner --rpcserver 127.0.0.1:42420 --miningaddr eiyaro:<your_address>
```

## Deployment

### Systemd Service (Linux)

Create `/etc/systemd/system/eyarod.service`:

```ini
[Unit]
Description=Eiyaro Full Node
After=network.target

[Service]
Type=simple
User=eyaro
ExecStart=/usr/local/bin/eyarod --utxoindex
Restart=on-failure
RestartSec=30
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable eyarod
sudo systemctl start eyarod
journalctl -u eyarod -f
```

### Docker Deployment

```bash
# Run full node with UTXO index
docker run -d \
  --name eyarod \
  --restart unless-stopped \
  -p 42420:42420 \
  -p 42421:42421 \
  -v eyarod-data:/root/.eyaro \
  ghcr.io/hoosat-oy/eyarod:latest

# View logs
docker logs -f eyarod

# Run miner alongside node
docker run -d \
  --name eiyaro-miner \
  --network host \
  -v miner-data:/root/.eyaro \
  ghcr.io/hoosat-oy/eyarod:latest \
  /app/miner --rpcserver 127.0.0.1:42420 --miningaddr eiyaro:<your_address>
```

### Docker Compose Stack

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  eyarod:
    image: ghcr.io/hoosat-oy/eyarod:latest
    container_name: eyarod
    restart: unless-stopped
    ports:
      - "42420:42420"   # RPC
      - "42421:42421"   # P2P
    volumes:
      - eyarod-data:/root/.eyaro
    command: --utxoindex

  miner:
    image: ghcr.io/hoosat-oy/eyarod:latest
    container_name: eiyaro-miner
    restart: unless-stopped
    network_mode: host
    volumes:
      - miner-data:/root/.eyaro
    command: /app/miner --rpcserver 127.0.0.1:42420 --miningaddr eiyaro:<your_address>
    depends_on:
      - eyarod

volumes:
  eyarod-data:
  miner-data:
```

```bash
docker-compose up -d
```

### Manual Deployment (Linux)

```bash
# Install binary
sudo cp eyarod /usr/local/bin/
sudo cp eiyaroctl /usr/local/bin/

# Create data directory
mkdir -p /var/lib/eyaro

# Create service user
sudo useradd -r -s /bin/false -d /var/lib/eyaro eyaro
sudo chown -R eyaro:eyaro /var/lib/eyaro

# Install systemd service (use the unit file above)
sudo systemctl enable --now eyarod
```

### Windows Service

```bash
# Install as Windows service
eyarodsvc install

# Start service
eyarodsvc start

# Stop service
eyarodsvc stop

# Remove service
eyarodsvc remove

# Or run directly
eyarod.exe --utxoindex
```

### High-Availability Production Deployment

```bash
# Dedicated bare-metal server or cloud VM (recommended specs)
# CPU: 8+ cores
# RAM: 16 GB+
# Disk: 500 GB+ SSD (NVMe preferred)
# Network: 100 Mbps+ symmetric, static IP

# System tuning
sudo sysctl -w net.core.rmem_max=134217728
sudo sysctl -w net.core.wmem_max=134217728

# Run multiple nodes behind a load balancer for high availability
# Use --externalip to advertise correct peer addresses
./eyarod --utxoindex --externalip=<public_ip> --rpclisten=0.0.0.0:42420
```

## Documentation

Comprehensive English-language manuals for every Eiyaro component.

### Getting Started

| Manual | Description |
|--------|-------------|
| [System Architecture](docs/architecture.md) | High-level architecture, layer design, data flow, and component relationships |
| [Building & Deployment](docs/building.md) | Build scripts, Makefile targets, Docker, cross-compilation, CI/CD |
| [Configuration Guide](docs/configuration.md) | Full config reference — config file, CLI flags, environment variables |
| [Network Parameters](docs/network-params.md) | Mainnet/testnet/devnet/simnet parameters, genesis blocks, economic model |

### Node & Tools

| Manual | Description |
|--------|-------------|
| [Eiyaro Full Node (eyarod)](docs/eyarod.md) | Node startup, CLI flags, data directory, signal handling, Windows service |
| [CLI Tool (eiyaroctl)](docs/eiyaroctl.md) | All RPC commands, parameter syntax, usage examples |
| [Eiyaro Wallet](docs/wallet.md) | HD wallet, BIP32/BIP39 key derivation, transactions, daemon mode |
| [Miner](docs/miner.md) | CPU mining, NogoPow integration, block template flow, configuration |

### Core Protocol

| Manual | Description |
|--------|-------------|
| [Consensus Engine](docs/consensus.md) | GhostDAG algorithm, block validation pipeline, finality, pruning |
| [NogoPow Algorithm](docs/nogopow.md) | 256x256 matrix PoW, Blake3 hashing, PI controller, performance |
| [JSON-RPC API Reference](docs/rpc-api.md) | All RPC methods, request/response schemas, notifications, error codes |

### Infrastructure

| Manual | Description |
|--------|-------------|
| [P2P Network](docs/p2p-network.md) | Protocol version 7, handshake, message types, DNS seeding, connection management |
| [Database Layer](docs/database.md) | LevelDB/PebbleDB backends, key layout, transactions, serialization |
| [Transaction Script Engine](docs/txscript.md) | Stack-based VM, opcodes reference, Schnorr/ECDSA signing, script building |
| [Bech32 Addressing](docs/bech32.md) | Address types (P2PK/P2PKH/P2SH), encoding/decoding, prefix system |

## Development

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests
cd testing/integration
go test -v

# Race detection
go test -race ./...
```

### Code Quality

```bash
# Format
gofmt -w .

# Static analysis
go vet ./...

# Lint
golangci-lint run
```

## Architecture

Eiyaro follows a modular clean architecture design:

```
Eiyaro/
├── app/                  # Application layer (RPC, protocol management)
├── cmd/                  # Command-line programs
│   ├── eyarod/          # Full node
│   ├── eiyaroctl/       # CLI RPC client
│   ├── eiyarowallet/    # HD wallet
│   └── miner/           # CPU miner
├── domain/               # Core business logic
│   ├── consensus/       # Consensus engine (GhostDAG)
│   ├── mining/          # Mining logic & mempool
│   └── dagconfig/       # Network parameters
├── infrastructure/       # Infrastructure layer
│   ├── db/              # Database (LevelDB/PebbleDB)
│   ├── network/         # P2P networking
│   └── config/          # Configuration management
└── util/                 # Utility functions
    ├── bech32/           # Address encoding
    └── txscript/         # Transaction script VM
```

## Network Parameters

| Parameter | Mainnet | Testnet |
|-----------|---------|---------|
| Block Time | 20 sec | 20 sec |
| Initial Reward | 8 EY | 8 EY |
| Annual Reduction | 10% | 10% |
| Floor Reward | 0.1 EY | 0.1 EY |
| P2P Port | 42421 | 42423 |
| RPC Port | 42420 | 42422 |
| Address Prefix | eiyaro | eiyarotest |

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md).

### How to Contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License — see [LICENSE](LICENSE) for details.

## Contact

- Website: https://eiyaro.org
- Documentation: https://docs.eiyaro.org
- Discord: [Join Community](https://discord.gg/eiyaro)
- Twitter: [@EiyaroChain](https://twitter.com/EiyaroChain)

## Acknowledgments

Eiyaro is built upon the KAS architecture. Thanks to the original team for their open-source contributions.

---

**Eiyaro** — Building the next generation of decentralized financial infrastructure