# Eiyaro Wallet (eiyarowallet)

## Overview

**eiyarowallet** is an HD (Hierarchical Deterministic) wallet for the Eiyaro
blockchain. It generates and manages keys using BIP32 hierarchical derivation
from BIP39 mnemonic seed phrases. The wallet supports both command-line
operations and a long-running daemon mode with a gRPC API, making it suitable
for interactive use, scripting, and integration into backend services.

### Core Features

- **BIP39 Mnemonics** &mdash; 24-word seed phrases (256-bit entropy)
- **BIP32 Hierarchical Derivation** &mdash; derivable key chains for
  receiving and change addresses
- **Schnorr Signatures** &mdash; default; ECDSA available via `--ecdsa` flag
- **Multi-signature (multisig)** &mdash; M-of-N signature schemes via
  `--min-signatures`, `--num-private-keys`, `--num-public-keys`
- **Address Types** &mdash; P2PK (default) and P2PKH
- **Bech32 Address Encoding** &mdash; with the `eiyaro:` prefix
- **Encrypted Key Storage** &mdash; Argon2id key derivation +
  ChaCha20-Poly1305 AEAD encryption
- **Daemon Mode** &mdash; long-running gRPC server for headless operation
- **Offline Signing** &mdash; create unsigned transactions on one machine,
  sign on another

---

## Quick Start

### 1. Create a Wallet

```bash
eiyarowallet create
```

You will be prompted to enter and confirm a wallet password. Upon success, a
24-word mnemonic and the corresponding extended public key are displayed. The
encrypted wallet is saved to the default keys file location.

### 2. Start the Daemon

```bash
eiyarowallet start-daemon
```

This connects to an Eiyaro full node via RPC and exposes a gRPC API on
`localhost:4282` (the default listen address).

### 3. Generate a Receiving Address

```bash
eiyarowallet new-address
```

### 4. Check Your Balance

```bash
eiyarowallet balance
```

### 5. Send Funds

```bash
eiyarowallet send --to-address eiyaro:qpx0... --send-amount 100.12345678 --password "your_password"
```

---

## Wallet Creation

### `create`

Creates a new wallet with one or more mnemonic seed phrases. This is the
first command you should run.

```
eiyarowallet create [flags]
```

**Flags:**

| Flag                 | Shorthand | Type     | Default | Description                                                  |
|----------------------|-----------|----------|---------|--------------------------------------------------------------|
| `--keys-file`        | `-f`      | `string` | auto    | Keys file location                                           |
| `--password`         | `-p`      | `string` | (prompt)  | Wallet password                                              |
| `--yes`              | `-y`      | `bool`   | `false` | Assume "yes" to all prompts                                  |
| `--min-signatures`   | `-m`      | `uint32` | `1`     | Minimum required signatures for multisig                     |
| `--num-private-keys` | `-k`      | `uint32` | `1`     | Number of private keys to generate                           |
| `--num-public-keys`  | `-n`      | `uint32` | `1`     | Total number of keys in the wallet                           |
| `--ecdsa`            |           | `bool`   | `false` | Create an ECDSA wallet instead of Schnorr                    |
| `--import`           | `-i`      | `bool`   | `false` | Import existing mnemonics instead of generating new ones     |
| `--testnet`          |           | `bool`   | `false` | Use the test network                                         |
| `--simnet`           |           | `bool`   | `false` | Use the simulation test network                              |
| `--devnet`           |           | `bool`   | `false` | Use the development test network                             |

**What happens internally:**

1. One or more BIP39 mnemonics are generated (24 words, 256-bit entropy)
   or imported if `--import` is used.
2. Each mnemonic is converted to a BIP32 seed using `bip39.NewSeed(mnemonic, "")`
   (no passphrase).
3. The master key is derived to the path `m/44'/111111'/0'` (single-sig) or
   `m/45'/111111'/0'` (multisig).
4. The resulting extended public key is displayed.
5. The mnemonic is encrypted with Argon2id + ChaCha20-Poly1305 and stored
   alongside the extended public key(s) in the keys file.
6. An exclusive file lock is acquired on the keys file.

**Examples:**

```bash
# Basic wallet creation
eiyarowallet create

# Create with explicit password and keys file
eiyarowallet create --password "secure_pswd" --keys-file /path/to/keys.json

# Create an ECDSA wallet
eiyarowallet create --ecdsa

# Multisig: 2-of-3 wallet
eiyarowallet create --min-signatures 2 --num-private-keys 1 --num-public-keys 3

# Import an existing mnemonic
eiyarowallet create --import
```

---

## Core Commands

### `balance`

Displays the wallet balance. Requires the daemon to be running.

```
eiyarowallet balance [flags]
```

**Flags:**

| Flag              | Shorthand | Type     | Default          | Description                            |
|-------------------|-----------|----------|------------------|----------------------------------------|
| `--daemonaddress` | `-d`      | `string` | `localhost:4282` | Wallet daemon server to connect to     |
| `--verbose`       | `-v`      | `bool`   | `false`          | Show addresses with individual balance |
| `--testnet`       |           | `bool`   | `false`          | Use the test network                   |
| `--simnet`        |           | `bool`   | `false`          | Use the simulation test network        |
| `--devnet`        |           | `bool`   | `false`          | Use the development test network       |

**Output:**

```
Total balance, EY           123.45678900             0.00000000
```

With `--verbose`:

```
Address                                                                       Available
-----------------------------------------------------------------------------------------
eiyaro:qpx0abc...                                                              100.00000000 0.00000000
eiyaro:qpx0def...                                                               23.45678900 0.00000000
-----------------------------------------------------------------------------------------
Total balance, EY           123.45678900             0.00000000
```

---

### `send`

Sends EY to an address. This command combines three operations: creating an
unsigned transaction, signing it, and broadcasting it.

```
eiyarowallet send --to-address <address> [flags]
```

**Flags:**

| Flag                           | Shorthand | Type       | Default          | Description                                                                    |
|--------------------------------|-----------|------------|------------------|--------------------------------------------------------------------------------|
| `--to-address`                 | `-t`      | `string`   | **(required)**   | The public address to send EY to                                               |
| `--from-address`               | `-a`      | `[]string` | (none)           | Specific address(es) to send from. Repeat for multiple                         |
| `--send-amount`                | `-v`      | `string`   | (none)           | Amount in EY (e.g. `1234.12345678`)                                            |
| `--send-all`                   |           | `bool`     | `false`          | Send all EY in the wallet                                                      |
| `--password`                   | `-p`      | `string`   | (prompt)         | Wallet password                                                                |
| `--keys-file`                  | `-f`      | `string`   | auto             | Keys file location                                                             |
| `--daemonaddress`              | `-d`      | `string`   | `localhost:4282` | Wallet daemon server to connect to                                             |
| `--use-existing-change-address`| `-u`      | `bool`     | `false`          | Use an existing change address                                                 |
| `--show-serialized`            | `-s`      | `bool`     | `false`          | Show hex-encoded signed transactions                                           |
| `--limit`                      | `-l`      | `string`   | (none)           | Limit number of UTXOs to fetch (default 10,000; 0 = no limit)                  |
| `--testnet`                    |           | `bool`     | `false`          | Use the test network                                                           |
| `--simnet`                     |           | `bool`     | `false`          | Use the simulation test network                                                |
| `--devnet`                     |           | `bool`     | `false`          | Use the development test network                                               |

> **Note:** Exactly one of `--send-amount` or `--send-all` must be specified.

**Example:**

```bash
eiyarowallet send --to-address eiyaro:qpx0recipient... --send-amount 10.5
```

---

### `sign`

Signs one or more unsigned/partially-signed transactions.

```
eiyarowallet sign [flags]
```

**Flags:**

| Flag                | Shorthand | Type     | Default | Description                                                          |
|---------------------|-----------|----------|---------|----------------------------------------------------------------------|
| `--transaction`     | `-t`      | `string` | (none)  | Unsigned transaction(s) encoded in hex                               |
| `--transaction-file`| `-F`      | `string` | (none)  | File containing the unsigned transaction(s) in hex                   |
| `--password`        | `-p`      | `string` | (prompt)| Wallet password                                                      |
| `--keys-file`       | `-f`      | `string` | auto    | Keys file location                                                   |
| `--testnet`         |           | `bool`   | `false` | Use the test network                                                 |
| `--simnet`          |           | `bool`   | `false` | Use the simulation test network                                      |
| `--devnet`          |           | `bool`   | `false` | Use the development test network                                     |

> **Note:** Exactly one of `--transaction` or `--transaction-file` is required.

**Example:**

```bash
eiyarowallet sign --transaction hex_from_create_unsigned --password "pswd"
```

Upon full signing, the command prints: `The transaction is signed and ready to broadcast`.

---

### `broadcast`

Broadcasts signed transactions to the Eiyaro network.

```
eiyarowallet broadcast [flags]
```

**Flags:**

| Flag                | Shorthand | Type     | Default          | Description                                                |
|---------------------|-----------|----------|------------------|------------------------------------------------------------|
| `--transaction`     | `-t`      | `string` | (none)           | Signed transaction(s) encoded in hex                       |
| `--transaction-file`| `-F`      | `string` | (none)           | File containing the signed transaction(s) in hex           |
| `--daemonaddress`   | `-d`      | `string` | `localhost:4282` | Wallet daemon server to connect to                         |
| `--testnet`         |           | `bool`   | `false`          | Use the test network                                       |
| `--simnet`          |           | `bool`   | `false`          | Use the simulation test network                            |
| `--devnet`          |           | `bool`   | `false`          | Use the development test network                           |

> **Note:** Exactly one of `--transaction` or `--transaction-file` is required.

**Example:**

```bash
eiyarowallet broadcast --transaction signed_hex
```

---

### `new-address`

Generates a new receiving (external) address.

```
eiyarowallet new-address [flags]
```

**Flags:**

| Flag              | Shorthand | Type     | Default          | Description                                                        |
|-------------------|-----------|----------|------------------|--------------------------------------------------------------------|
| `--daemonaddress` | `-d`      | `string` | `localhost:4282` | Wallet daemon server to connect to                                 |
| `--address-type`  |           | `string` | `""` (P2PK)      | Address type: `p2pk` (default) or `p2pkh`                          |
| `--include-both`  |           | `bool`   | `false`          | Also print both P2PK and P2PKH forms (single-sig only)             |
| `--testnet`       |           | `bool`   | `false`          | Use the test network                                               |
| `--simnet`        |           | `bool`   | `false`          | Use the simulation test network                                    |
| `--devnet`        |           | `bool`   | `false`          | Use the development test network                                   |

**Example:**

```bash
eiyarowallet new-address --address-type p2pkh --include-both
```

---

### `show-addresses`

Lists all manually generated addresses of the current wallet.

```
eiyarowallet show-addresses [flags]
```

**Flags:**

| Flag              | Shorthand | Type     | Default          | Description                                                       |
|-------------------|-----------|----------|------------------|-------------------------------------------------------------------|
| `--daemonaddress` | `-d`      | `string` | `localhost:4282` | Wallet daemon server to connect to                                |
| `--address-type`  |           | `string` | `""` (P2PK)      | Address type: `p2pk` (default) or `p2pkh`                         |
| `--include-both`  |           | `bool`   | `false`          | Include both P2PK and P2PKH forms (single-sig only)               |
| `--testnet`       |           | `bool`   | `false`          | Use the test network                                              |
| `--simnet`        |           | `bool`   | `false`          | Use the simulation test network                                   |
| `--devnet`        |           | `bool`   | `false`          | Use the development test network                                  |

**Output:**

```
Addresses (3):
eiyaro:qpx0addr1...
eiyaro:qpx0addr2...
eiyaro:qpx0addr3...

Note: the above are only addresses that were manually created by the 'new-address'
command. If you want to see a list of all addresses, including change addresses,
that have a positive balance, use the command 'balance -v'
```

---

### `create-unsigned-transaction`

Creates unsigned transaction(s) for offline or multisig signing scenarios.

```
eiyarowallet create-unsigned-transaction [flags]
```

**Flags:**

| Flag                            | Shorthand | Type       | Default          | Description                                                      |
|---------------------------------|-----------|------------|------------------|------------------------------------------------------------------|
| `--to-address`                  | `-t`      | `string`   | **(required)**   | The public address to send EY to                                 |
| `--from-address`                | `-a`      | `[]string` | (none)           | Specific address(es) to send from                                |
| `--send-amount`                 | `-v`      | `string`   | (none)           | Amount in EY (e.g. `1234.12345678`)                              |
| `--send-all`                    |           | `bool`     | `false`          | Send all EY                                                      |
| `--use-existing-change-address` | `-u`      | `bool`     | `false`          | Use an existing change address                                   |
| `--daemonaddress`               | `-d`      | `string`   | `localhost:4282` | Wallet daemon server to connect to                               |
| `--testnet`                     |           | `bool`     | `false`          | Use the test network                                             |
| `--simnet`                      |           | `bool`     | `false`          | Use the simulation test network                                  |
| `--devnet`                      |           | `bool`     | `false`          | Use the development test network                                 |

**Example:**

```bash
eiyarowallet create-unsigned-transaction --to-address eiyaro:qpx0... --send-amount 50 > unsigned.hex
```

---

### `auto-compound`

Periodically compounds UTXOs to improve wallet performance. Run as a
background process.

```
eiyarowallet auto-compound [flags]
```

**Flags:**

| Flag                            | Shorthand | Type       | Default          | Description                                                      |
|---------------------------------|-----------|------------|------------------|------------------------------------------------------------------|
| `--to-address`                  | `-t`      | `string`   | **(required)**   | The public address to compound EY to                             |
| `--from-address`                | `-a`      | `[]string` | (none)           | Specific address(es) to compound from                            |
| `--compound-rate`               | `-c`      | `int`      | `60`             | Time between compounds in seconds (minimum 6; forced to 60 if below) |
| `--password`                    | `-p`      | `string`   | (prompt)         | Wallet password                                                  |
| `--keys-file`                   | `-f`      | `string`   | auto             | Keys file location                                               |
| `--daemonaddress`               | `-d`      | `string`   | `localhost:4282` | Wallet daemon server to connect to                               |
| `--use-existing-change-address` | `-u`      | `bool`     | `false`          | Use an existing change address                                   |
| `--show-serialized`             | `-s`      | `bool`     | `false`          | Show hex-encoded sent transactions                               |
| `--limit`                       | `-l`      | `string`   | (none)           | Limit number of UTXOs (default 10,000; 0 = no limit)             |
| `--testnet`                     |           | `bool`     | `false`          | Use the test network                                             |
| `--simnet`                      |           | `bool`     | `false`          | Use the simulation test network                                  |
| `--devnet`                      |           | `bool`     | `false`          | Use the development test network                                 |

**Example:**

```bash
eiyarowallet auto-compound --to-address eiyaro:qpx0... --compound-rate 120
```

> The command runs indefinitely. Press Ctrl+C to stop.

---

### `sweep`

Sweeps all funds from a Schnorr private key to a new address of the
current wallet. Useful for consolidating funds from cold keys or keys
generated offline.

```
eiyarowallet sweep [flags]
```

**Flags:**

| Flag              | Shorthand | Type     | Default          | Description                            |
|-------------------|-----------|----------|------------------|----------------------------------------|
| `--private-key`   | `-k`      | `string` | **(required)**   | Private key in hex format              |
| `--daemonaddress` | `-d`      | `string` | `localhost:4282` | Wallet daemon server to connect to     |
| `--testnet`       |           | `bool`   | `false`          | Use the test network                   |
| `--simnet`        |           | `bool`   | `false`          | Use the simulation test network        |
| `--devnet`        |           | `bool`   | `false`          | Use the development test network       |

**Example:**

```bash
eiyarowallet sweep --private-key abc123hex...
```

---

### `parse`

Parses and displays a transaction in a human-readable form.

```
eiyarowallet parse [flags]
```

**Flags:**

| Flag                | Shorthand | Type     | Default | Description                                                |
|---------------------|-----------|----------|---------|------------------------------------------------------------|
| `--transaction`     | `-t`      | `string` | (none)  | Transaction encoded in hex                                 |
| `--transaction-file`| `-F`      | `string` | (none)  | File containing the transaction in hex                     |
| `--verbose`         | `-v`      | `bool`   | `false` | Show transaction inputs with amounts                       |
| `--testnet`         |           | `bool`   | `false` | Use the test network                                       |
| `--simnet`          |           | `bool`   | `false` | Use the simulation test network                            |
| `--devnet`          |           | `bool`   | `false` | Use the development test network                           |

**Example:**

```bash
eiyarowallet parse --transaction hex_string --verbose
```

**Sample output:**

```
Transaction #1 ID:     abc123...

Input 0:     Outpoint: def456...:0     Amount: 100.00 Eiyaro

Output 0:    Recipient: eiyaro:qpx0...     Amount: 99.90 Eiyaro

Fee:    10000 Sey
```

---

### `vote`

Casts a vote on a governance poll by sending 1 EY with a vote payload.

```
eiyarowallet vote [flags]
```

**Flags:**

| Flag                            | Shorthand | Type       | Default          | Description                                                      |
|---------------------------------|-----------|------------|------------------|------------------------------------------------------------------|
| `--poll-id`                     | `-i`      | `string`   | **(required)**   | The poll ID to vote on                                           |
| `--vote`                        | `-v`      | `[]int`    | **(required)**   | Vote value(s). Repeat for multiple votes                         |
| `--password`                    | `-p`      | `string`   | (prompt)         | Wallet password                                                  |
| `--keys-file`                   | `-f`      | `string`   | auto             | Keys file location                                               |
| `--daemonaddress`               | `-d`      | `string`   | `localhost:4282` | Wallet daemon server to connect to                               |
| `--from-address`                | `-a`      | `[]string` | (none)           | Specific address(es) to send from                                |
| `--use-existing-change-address` | `-u`      | `bool`     | `false`          | Use an existing change address                                   |
| `--show-serialized`             | `-s`      | `bool`     | `false`          | Show hex-encoded sent transactions                               |
| `--testnet`                     |           | `bool`     | `false`          | Use the test network                                             |
| `--simnet`                      |           | `bool`     | `false`          | Use the simulation test network                                  |
| `--devnet`                      |           | `bool`     | `false`          | Use the development test network                                 |

**Example:**

```bash
eiyarowallet vote --poll-id "poll_abc123" --vote 1
```

---

### `dump-unencrypted-data`

Exports wallet data in unencrypted form. **Use with extreme caution.**

```
eiyarowallet dump-unencrypted-data [flags]
```

**Flags:**

| Flag          | Shorthand | Type     | Default | Description                      |
|---------------|-----------|----------|---------|----------------------------------|
| `--password`  | `-p`      | `string` | (prompt)| Wallet password                  |
| `--keys-file` | `-f`      | `string` | auto    | Keys file location               |
| `--yes`       | `-y`      | `bool`   | `false` | Skip confirmation prompt         |
| `--testnet`   |           | `bool`   | `false` | Use the test network             |
| `--simnet`    |           | `bool`   | `false` | Use the simulation test network  |
| `--devnet`    |           | `bool`   | `false` | Use the development test network |

**Example:**

```bash
eiyarowallet dump-unencrypted-data --password "pswd"
```

This prints all mnemonics, extended public keys, and the minimum
signatures value. Anyone who sees this output can steal your funds.

---

### `start-daemon`

Starts the wallet daemon as a long-running gRPC server.

```
eiyarowallet start-daemon [flags]
```

**Flags:**

| Flag            | Shorthand | Type     | Default          | Description                                                            |
|-----------------|-----------|----------|------------------|------------------------------------------------------------------------|
| `--keys-file`   | `-f`      | `string` | auto             | Keys file location                                                     |
| `--password`    | `-p`      | `string` | (prompt)         | Wallet password                                                        |
| `--rpcserver`   | `-s`      | `string` | `localhost`      | RPC server of the Eiyaro full node to connect to                       |
| `--listen`      | `-l`      | `string` | `localhost:4282` | Address for the gRPC daemon to listen on                               |
| `--wait-timeout`| `-w`      | `uint32` | `30`             | Timeout in seconds for RPC calls                                       |
| `--profile`     |           | `string` | (none)           | Enable HTTP profiling on the given port (1024-65536)                   |
| `--testnet`     |           | `bool`   | `false`          | Use the test network                                                   |
| `--simnet`      |           | `bool`   | `false`          | Use the simulation test network                                        |
| `--devnet`      |           | `bool`   | `false`          | Use the development test network                                       |

**Example:**

```bash
eiyarowallet start-daemon --rpcserver 127.0.0.1:42110 --listen 0.0.0.0:4282
```

---

### `version`

Displays the wallet binary version.

```
eiyarowallet version
```

---

### `get-daemon-version`

Queries the running daemon for its version.

```
eiyarowallet get-daemon-version [flags]
```

**Flags:**

| Flag              | Shorthand | Type     | Default          | Description                        |
|-------------------|-----------|----------|------------------|------------------------------------|
| `--daemonaddress` | `-d`      | `string` | `localhost:4282` | Wallet daemon server to connect to |

---

## Daemon Mode

The `start-daemon` command launches a background gRPC server that handles
all wallet operations. All other commands (`balance`, `send`, etc.) connect
to this daemon as clients.

### gRPC API Methods

The daemon exposes the following RPC methods on the `eiyarowalletd` service:

| Method                               | Description                                                       |
|--------------------------------------|-------------------------------------------------------------------|
| `GetBalance`                         | Returns available, pending, and per-address balances              |
| `GetExternalSpendableUTXOs`          | Returns spendable UTXOs for a given external address              |
| `CreateUnsignedTransactions`         | Creates unsigned transactions from wallet UTXOs                   |
| `CreateUnsignedCompoundTransaction`  | Creates an unsigned compound (consolidation) transaction          |
| `ShowAddresses`                      | Returns all manually generated wallet addresses                   |
| `NewAddress`                         | Generates and returns a new receiving address                     |
| `Send`                               | Sends funds with password (use only on trusted connections)       |
| `Sign`                               | Signs transactions with password (use only on trusted connections)|
| `Broadcast`                          | Broadcasts signed transactions to the network                     |
| `Shutdown`                           | Gracefully shuts down the daemon                                  |
| `GetVersion`                         | Returns the daemon version string                                 |

### Protobuf Messages

#### General

- **AddressType** enum: `ADDRESS_TYPE_UNSPECIFIED` (0), `ADDRESS_TYPE_P2PKH` (1), `ADDRESS_TYPE_P2PK` (2)
- **Outpoint**: `transactionId` (string), `index` (uint32)
- **UtxoEntry**: `amount` (uint64), `scriptPublicKey` (ScriptPublicKey), `blockDaaScore` (uint64), `isCoinbase` (bool)
- **ScriptPublicKey**: `version` (uint32), `scriptPublicKey` (string/hex)

#### GetBalance

- **Request**: (empty)
- **Response**: `available` (uint64), `pending` (uint64), `addressBalances` (repeated AddressBalances)

#### GetExternalSpendableUTXOs

- **Request**: `address` (string)
- **Response**: `Entries` (repeated UtxosByAddressesEntry)

#### CreateUnsignedTransactions

- **Request**: `address` (recipient), `amount` (uint64, in Sey), `from` (repeated string),
  `useExistingChangeAddress` (bool), `isSendAll` (bool), `payload` (bytes),
  `limit` (optional string)
- **Response**: `unsignedTransactions` (repeated bytes)

#### CreateUnsignedCompoundTransaction

- **Request**: `address`, `from`, `useExistingChangeAddress`, `limit` (optional string)
- **Response**: `unsignedTransactions` (repeated bytes)

#### Broadcast

- **Request**: `isDomain` (bool), `transactions` (repeated bytes),
  `allowOrphan` (bool), `isHighPriority` (optional bool)
- **Response**: `txIDs` (repeated string)

#### Send

- **Request**: `toAddress` (string), `amount` (uint64), `password` (string),
  `from` (repeated string), `useExistingChangeAddress` (bool),
  `isSendAll` (bool), `limit` (optional string)
- **Response**: `txIDs` (repeated string), `signedTransactions` (repeated bytes)

#### Sign

- **Request**: `unsignedTransactions` (repeated bytes), `password` (string)
- **Response**: `signedTransactions` (repeated bytes)

---

## Key Derivation

### BIP39 Mnemonic Generation

- **Entropy:** 256 bits → 24-word mnemonic
- **Seed derivation:** `bip39.NewSeed(mnemonic, "")` — no BIP39 passphrase

### BIP32 Hierarchical Paths

The wallet uses the following derivation paths:

| Wallet Type   | Purpose | Path                 |
|---------------|---------|----------------------|
| Single-sig    | 44'     | `m/44'/111111'/0'`   |
| Multisig      | 45'     | `m/45'/111111'/0'`   |

The **coin type** is `111111`. From the account level (`0'`), two key chains
are derived:

| Key Chain     | Index | Purpose                   |
|---------------|-------|---------------------------|
| External      | `0`   | Receiving addresses       |
| Internal      | `1`   | Change addresses          |

A full derivation path for a receiving address at index 5 is:
`m/44'/111111'/0'/0/5`

### BIP32 Version Bytes

| Network   | Private Key Version |
|-----------|---------------------|
| Mainnet   | Network-specific     |
| Testnet   | Network-specific     |
| Devnet    | Network-specific     |
| Simnet    | Network-specific     |

The exact version bytes are defined in `bip32` constants (`EiyaroMainnetPrivate`,
`EiyaroTestnetPrivate`, etc.).

### Address Generation

#### Single-Signature

For single-sig wallets (1 extended public key):

- **P2PK (default):** `NewAddressPublicKey(schnorrPubKey, prefix)` or
  `NewAddressPublicKeyECDSA(ecdsaPubKey, prefix)`
- **P2PKH:** `NewAddressPublicKeyHash(schnorrPubKey, prefix)` or
  `NewAddressPublicKeyHashECDSA(ecdsaPubKey, prefix)`

#### Multi-Signature (M-of-N)

For multisig wallets, addresses are Pay-to-Script-Hash (P2SH). The redeem
script encodes:

- Minimum required signatures (M)
- Each compressed public key
- Total number of keys (N)
- Check multisig opcode (`OpCheckMultiSig` for Schnorr, `OpCheckMultiSigECDSA` for ECDSA)

### Address Encoding

All addresses are Bech32-encoded with the `eiyaro:` prefix (the prefix is
configured per network parameters).

---

## Configuration

### Keys File

The wallet stores encrypted mnemonics, extended public keys, and signature
configuration in a JSON file. The default location depends on the operating
system and follows the platform conventions for application data directories.
Within the application data directory, the file is stored under
`<network>/keys.json` (e.g., `mainnet/keys.json`).

Override with the `--keys-file` / `-f` flag.

### Keys File Structure

```json
{
  "version": 1,
  "encryptedMnemonics": [
    {
      "cipher": "<hex>",
      "salt": "<hex>"
    }
  ],
  "publicKeys": ["<extended_public_key>"],
  "minimumSignatures": 1,
  "cosignerIndex": 0,
  "lastUsedExternalIndex": 5,
  "lastUsedInternalIndex": 3,
  "ecdsa": false
}
```

### Network Selection

Use one of the following flags to select the network:

- `--testnet` — test network
- `--simnet` — simulation test network
- `--devnet` — development test network

If none is specified, **mainnet** is used.

### Daemon Address

The `--daemonaddress` / `-d` flag specifies the gRPC address of the running
wallet daemon. Default: `localhost:4282`.

---

## Transaction Flow

The standard transaction flow involves three steps:

### 1. Create Unsigned Transaction

```
eiyarowallet create-unsigned-transaction --to-address <addr> --send-amount <amt>
```

This produces hex-encoded unsigned transaction(s). The daemon selects UTXOs,
creates inputs with empty signature slots, and constructs outputs including
a change output if needed.

### 2. Sign

```
eiyarowallet sign --transaction <hex>
```

This decrypts the wallet mnemonics using the password, derives the
appropriate private keys for each input, computes Schnorr (or ECDSA)
signatures with `SigHashAll`, and populates the signature slots.

### 3. Broadcast

```
eiyarowallet broadcast --transaction <hex>
```

This sends the fully signed transaction to the Eiyaro network via the
daemon's connection to the full node.

### One-Step Send

The `send` command performs all three steps in a single invocation. It also
handles retries (up to 10 attempts with a 2-second delay) for cases like
insufficient spendable UTXOs at the time of creation.

### Hex Encoding for Multiple Transactions

When multiple transactions are involved, they are encoded as a single hex
string with transactions separated by the `_` (underscore) character, which
is outside the hex alphabet. The `encodeTransactionsToHex` and
`decodeTransactionsFromHex` functions handle this format.

---

## Usage Examples

### Example 1: Full Wallet Setup

```bash
# Create a new wallet
eiyarowallet create --password "MyStr0ngP@ss"

# Start the daemon (in a separate terminal or background)
eiyarowallet start-daemon --password "MyStr0ngP@ss"

# Generate a receiving address
eiyarowallet new-address

# Check balance
eiyarowallet balance -v

# Send 100 EY
eiyarowallet send --to-address eiyaro:qpx0recipient... --send-amount 100 --password "MyStr0ngP@ss"
```

### Example 2: Offline Signing (Cold Wallet)

```bash
# On the online machine (with daemon running):
eiyarowallet create-unsigned-transaction --to-address eiyaro:qpx0... --send-amount 100 > tx.hex

# Transfer tx.hex to the offline machine, then:
eiyarowallet sign --transaction-file tx.hex --password "cold_pswd" > tx_signed.hex

# Transfer tx_signed.hex back to the online machine, then:
eiyarowallet broadcast --transaction-file tx_signed.hex
```

### Example 3: Multisig Wallet (2-of-3)

```bash
# Party A creates the wallet with 1 private key and 3 public keys
eiyarowallet create --min-signatures 2 --num-private-keys 1 --num-public-keys 3

# Parties B and C provide their extended public keys when prompted.
# After creation, send the keys file to all parties.
```

### Example 4: Sweeping a Cold Key

```bash
# Sweep funds from a Schnorr private key to the wallet
eiyarowallet sweep --private-key abc123def456...
```

### Example 5: Auto-Compound

```bash
# Run auto-compounder every 5 minutes
eiyarowallet auto-compound --to-address eiyaro:qpx0... --compound-rate 300
```

### Example 6: ECDSA Wallet

```bash
# Create an ECDSA-based wallet
eiyarowallet create --ecdsa --password "pswd"

# Start daemon (ECDSA mode is stored in the keys file, no separate flag needed)
eiyarowallet start-daemon --password "pswd"

# Generate address
eiyarowallet new-address
```

---

## Notes & Caveats

### Security

- **Always back up your mnemonic seed phrase.** It is the only way to
  recover your funds if you lose access to the keys file.
- **Password cannot be recovered.** There is no password reset mechanism.
  If you forget your password, your funds are permanently inaccessible
  unless you have the unencrypted mnemonic.
- **The `Send` and `Sign` gRPC methods transmit the password over the
  network.** Use them only on trusted or secure connections (localhost
  or TLS). For remote setups, prefer the offline signing workflow
  (`create-unsigned-transaction` → `sign` → `broadcast`).
- **`dump-unencrypted-data` prints your private keys in plain text.**
  Anyone who sees this output can steal your funds. Run it only in a
  secure environment and never pipe the output to an untrusted destination.

### Daemon Mode

- The daemon requires a running Eiyaro full node accessible via RPC.
  Specify the node address with `--rpcserver`.
- The daemon performs an initial UTXO sync on startup. Large wallets may
  take time to complete the first sync.
- Only one daemon process can hold the lock on a keys file at a time.
  Attempting to start a second daemon with the same keys file will fail.
- The daemon listens on `localhost:4282` by default. Change this with
  `--listen` to bind to a different interface (e.g., `0.0.0.0:4282` for
  remote access &mdash; only do this with TLS).

### File Locking

The keys file uses advisory file locking (via `flock`). If a keys file is
reported as locked, ensure no other wallet command or daemon is using it.

### Transaction Fees

The `sweep` command uses a fixed fee of 10,000 Sey per input. Other
transactions have fees determined by the daemon's UTXO selection and the
network's minimum relay fee.

### Multisig Limitations

- The `send` and `vote` commands cannot be used with a multisig wallet that
  does not have all private keys available.
- For partial signing in multisig scenarios, use `create-unsigned-transaction`
  followed by `sign` on each cosigner's machine.

### Network Compatibility

- Address prefixes differ between mainnet and test networks. Always use the
  correct network flag (`--testnet`, `--simnet`, `--devnet`) consistently
  across all commands.
- The default coinbase maturity may differ on testnet (e.g., 1000 on
  testnet vs. the mainnet default).

### Performance

- Wallets with many small UTXOs may benefit from the `auto-compound` command,
  which consolidates UTXOs into fewer, larger outputs for better performance.
- Use `--limit` to control the maximum number of UTXOs fetched during
  operations. The default is 10,000; set it to 0 for no limit.

---

## FAQ

### How do I recover my wallet if I lose my password?

You cannot recover an encrypted wallet without the password. The only recovery path is using your 24-word BIP39 mnemonic seed phrase with `eiyarowallet create --import` to recreate the wallet from scratch. Always back up your mnemonic securely before storing any funds.

### How do I check my transaction status?

Use `eiyarowallet parse --transaction <hex>` to inspect any transaction's inputs, outputs, and fee details. For on-chain confirmation status, query the full node via eiyaroctl commands like `getTransaction <txid>` or check whether your balance has been updated with `eiyarowallet balance`.

### Can I use the same wallet on multiple computers?

Yes. Transfer the encrypted keys file to another machine along with the wallet password, or recreate the wallet using the same mnemonics via `eiyarowallet create --import`. Note that only one daemon can hold the file lock on a keys file at a time, so you cannot run the daemon simultaneously on two machines with the same keys file.

### What happens to my funds if the daemon crashes?

No funds are at risk. The daemon only manages keys and relays transactions; the actual blockchain state and UTXOs are stored on the full node. Transactions already broadcast remain in the mempool or on-chain. When you restart the daemon, it will resync UTXOs from the full node and restore full functionality.

### How do I vote on governance polls?

Use the `vote` command: `eiyarowallet vote --poll-id "poll_abc123" --vote 1 --password "your_password"`. This sends 1 EY with a vote payload to the specified poll ID. You can repeat the `--vote` flag to cast multiple votes. The daemon must be running and connected to the full node.