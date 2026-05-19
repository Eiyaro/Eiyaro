# Eiyaro Network Parameters

## Overview

Eiyaro defines six distinct network configurations, each serving a different purpose in the development and deployment lifecycle:

| # | Network | `Name` Field | Purpose |
|---|---------|-------------|---------|
| 1 | **Mainnet** | `eiyaro-mainnet` | Production network for real monetary value transfer |
| 2 | **Testnet** | `eiyaro-testnet` | Public testing network, mirrors mainnet parameters |
| 3 | **Testnet B5** | `eiyaro-testnet-b5` | Testnet variant with 5× merge set size limit (for stress-testing high-throughput scenarios) |
| 4 | **Testnet B10** | `eiyaro-testnet-b10` | Testnet variant with 10× merge set size limit (for extreme-throughput testing) |
| 5 | **Devnet** | `eiyaro-devnet` | Private development network; no DNS seeds, not intended for public use |
| 6 | **Simnet** | `eiyaro-simnet` | Simulation network for private group testing; difficulty adjustment disabled, no DNS seeds |

All networks are mutually incompatible — each has a distinct genesis block. Software must reject inputs intended for one network when running on another.

---

## Network Types Comparison

### Core Network Identity

| Parameter | Mainnet | Testnet | Testnet B5 | Testnet B10 | Devnet | Simnet |
|-----------|---------|---------|------------|-------------|--------|--------|
| `Name` | `eiyaro-mainnet` | `eiyaro-testnet` | `eiyaro-testnet-b5` | `eiyaro-testnet-b10` | `eiyaro-devnet` | `eiyaro-simnet` |
| `Net` (enum) | `Mainnet` | `Testnet` | `Testnet` | `Testnet` | `Devnet` | `Simnet` |
| P2P Port (`DefaultPort`) | `42421` | `42423` | `42423` | `42423` | `42427` | `42425` |
| RPC Port (`RPCPort`) | `42420` | `42422` | `42422` | `42422` | `42426` | `42424` |

### Address Encoding

| Parameter | Mainnet | Testnet | Testnet B5 | Testnet B10 | Devnet | Simnet |
|-----------|---------|---------|------------|-------------|--------|--------|
| Bech32 HRP (`Prefix`) | `eiyaro` | `eiyarotest` | `eiyarotest` | `eiyarotest` | `eiyaro` | `eiyarosim` |
| WIF Private Key (`PrivateKeyID`) | `0x80` | `0x80` | `0xef` | `0xef` | `0x80` | `0x64` |
| WIF Prefix (uncompressed) | `5` | `5` | `9` | `9` | `5` | `4` |
| WIF Prefix (compressed) | `K` | `K` | `c` | `c` | `K` | `F` |

### Block Time & Difficulty

| Parameter | Mainnet | Testnet | Testnet B5 | Testnet B10 | Devnet | Simnet |
|-----------|---------|---------|------------|-------------|--------|--------|
| `TargetTimePerBlock` | `20s` | `20s` | `20s` | `20s` | `20s` | `20s` |
| `DifficultyAdjustmentWindowSize` | `2641` | `2641` | `2641` | `2641` | `2641` | `2641` |
| `DisableDifficultyAdjustment` | `false` | `false` | `false` | `false` | `false` | `true` |
| `TimestampDeviationTolerance` | `132` | `132` | `132` | `132` | `132` | `132` |
| `POWScores` | `MaxUint64` | `MaxUint64` | `MaxUint64` | `MaxUint64` | `MaxUint64` | `MaxUint64` |

### Subsidy & Economic Model

| Parameter | Mainnet | Testnet | Testnet B5 | Testnet B10 | Devnet | Simnet |
|-----------|---------|---------|------------|-------------|--------|--------|
| `SubsidyGenesisReward` | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) |
| `PreDeflationaryPhaseBaseSubsidy` | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) |
| `DeflationaryPhaseBaseSubsidy` | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) | `800,000,000` Sey (8 EY) |
| `DeflationaryPhaseCurveFactor` | `1.0` | `1.0` | `1.0` (default) | `1.0` (default) | `1.0` | `1.0` (default) |
| `DeflationaryPhaseDaaScore` | `360` | `360` | `360` | `360` | `360` | `360` |
| `BlockCoinbaseMaturity` | `100` | `100` | `100` | `100` | `100` | `100` |
| Minimum Subsidy Floor | `0.1 EY` | `0.1 EY` | `0.1 EY` | `0.1 EY` | `0.1 EY` | `0.1 EY` |

### Block Structure & Mass

| Parameter | Mainnet | Testnet | Testnet B5 | Testnet B10 | Devnet | Simnet |
|-----------|---------|---------|------------|-------------|--------|--------|
| `MaxBlockMass` | `500,000` | `500,000` | `500,000` | `500,000` | `500,000` | `500,000` |
| `MaxBlockParents` | `10` | `10` | `10` | `10` | `10` | `10` |
| `MaxBlockLevel` | `225` | `225` | `225` | `250` | `225` | `250` |
| `MassPerTxByte` | `1` | `1` | `1` | `1` | `1` | `1` |
| `MassPerScriptPubKeyByte` | `10` | `10` | `10` | `10` | `10` | `10` |
| `MassPerSigOp` | `1,000` | `1,000` | `1,000` | `1,000` | `1,000` | `1,000` |
| `MaxCoinbasePayloadLength` | `204` | `204` | `204` | `204` | `204` | `204` |
| `CoinbasePayloadScriptPublicKeyMaxLength` | `150` | `150` | `150` | `150` | `150` | `150` |

### Consensus & Governance

| Parameter | Mainnet | Testnet | Testnet B5 | Testnet B10 | Devnet | Simnet |
|-----------|---------|---------|------------|-------------|--------|--------|
| `RuleChangeActivationThreshold` | `1,916` (95%) | `1,916` (95%) | `1,512` (75%) | `1,512` (75%) | `1,916` (95%) | `75` (75%) |
| `MinerConfirmationWindow` | `2,016` | `2,016` | `2,016` | `2,016` | `2,016` | `100` |
| `DisallowDirectBlocksOnTopOfGenesis` | `true` | `true` | *(not set)* | *(not set)* | `true` | *(not set)* |

### Network Routing

| Parameter | Mainnet | Testnet | Testnet B5 | Testnet B10 | Devnet | Simnet |
|-----------|---------|---------|------------|-------------|--------|--------|
| `AcceptUnroutable` | `false` | `false` | `false` | `false` | `false` | `false` |
| `RelayNonStdTxs` | `false` | `false` | `false` | `false` | `false` | `false` |
| `EnableNonNativeSubnetworks` | `false` | `false` | `false` | `false` | `false` | `false` |

### GHOSTDAG & Finality

| Parameter | Mainnet | Testnet | Testnet B5 | Testnet B10 | Devnet | Simnet |
|-----------|---------|---------|------------|-------------|--------|--------|
| `K` (GHOSTDAG) | `18` | `18` | `18` | `18` | `18` | `18` |
| `FinalityDuration` | `24h` | `24h` | `24h` | `24h` | `24h` | `24h` |
| FinalityDepth (computed) | `4,320` blocks | `4,320` blocks | `4,320` blocks | `4,320` blocks | `4,320` blocks | `4,320` blocks |
| `MergeSetSizeLimit` | `180` | `180` | `900` | `1,800` | `180` | `180` |
| `MergeDepth` | `360` | `360` | `360` | `360` | `360` | `360` |
| `PruningMultiplier` | `1` | `1` | `1` | `1` | `1` | `1` |
| `PruningProofM` | `1,000` | `1,000` | `1,000` | `1,000` | `1,000` | `1,000` |
| PruningDepth (computed) | `21,638` | `21,638` | `73,478` | `138,278` | `21,638` | `21,638` |

### Proof of Work

| Parameter | Mainnet | Testnet | Testnet B5 | Testnet B10 | Devnet | Simnet |
|-----------|---------|---------|------------|-------------|--------|--------|
| `PowMax` | 2²⁵⁵ − 1 | 2²⁵⁵ − 1 | 2²⁵⁵ − 1 | 2²⁵⁵ − 1 | 2²⁵⁵ − 1 | 2²⁵⁵ − 1 |
| `SkipProofOfWork` | *(not set)* | *(not set)* | *(not set)* | *(not set)* | *(not set)* | *(not set)* |

---

## Consensus Parameters (from `consensus_defaults.go`)

These constants are shared across all networks unless explicitly overridden:

| Constant | Value | Description |
|----------|-------|-------------|
| `defaultGHOSTDAGK` | `18` | GHOSTDAG K parameter — maximum number of blue blocks in the anticone of a blue block. Calculated for d = 5 seconds and δ = 0.05 per the PHANTOM paper (Section 4.2, Formula 1). |
| `defaultTargetTimePerBlock` | `20s` | Target time between consecutive block creations |
| `defaultFinalityDuration` | `24h` | Duration of the finality window |
| `defaultDifficultyAdjustmentWindowSize` | `2,641` | Number of blocks in a block's past used to calculate difficulty target. The DAA takes the median of 2,640 blocks, requiring a window of 2,641. |
| `defaultTimestampDeviationTolerance` | `132` | Allowed deviation of an incoming block's timestamp, measured in block delays |
| `defaultMaxBlockParents` | `10` | Maximum number of parent blocks a block may reference |
| `defaultMaxBlockMass` | `500,000` | Maximum mass (in grams) a block is allowed |
| `defaultMassPerTxByte` | `1` | Grams per transaction byte |
| `defaultMassPerScriptPubKeyByte` | `10` | Grams per script public key byte |
| `defaultMassPerSigOp` | `1,000` | Grams per signature operation |
| `defaultMergeSetSizeLimit` | `180` (= K × 10) | Maximum number of blocks in a block's merge set |
| `defaultMergeDepth` | `360` | Merge depth for anti-past computation |
| `defaultMaxCoinbasePayloadLength` | `204` | Maximum length (bytes) of a block's coinbase payload |
| `defaultCoinbasePayloadScriptPublicKeyMaxLength` | `150` | Maximum allowed script public key in the coinbase payload |
| `defaultPruningProofM` | `1,000` | The 'm' constant in the pruning proof |
| `defaultPruningMultiplier` | `1` | Multiplier used in pruning depth calculation |
| `BlockCoinbaseMaturity` | `100` | Number of blocks before newly mined coins become spendable |

### FinalityDepth (Computed)

```
FinalityDepth = FinalityDuration / TargetTimePerBlock
              = 86,400s / 20s
              = 4,320 blocks
```

### PruningDepth (Computed)

```
PruningDepth = 2 × FinalityDepth × PruningMultiplier
             + 4 × MergeSetSizeLimit × K
             + 2 × K
             + 2
```

Mainnet/Testnet/Devnet/Simnet:
```
PruningDepth = 2 × 4,320 × 1 + 4 × 180 × 18 + 2 × 18 + 2
             = 8,640 + 12,960 + 36 + 2
             = 21,638 blocks
```

Testnet B5:
```
PruningDepth = 8,640 + 4 × 900 × 18 + 36 + 2
             = 8,640 + 64,800 + 38
             = 73,478 blocks
```

Testnet B10:
```
PruningDepth = 8,640 + 4 × 1,800 × 18 + 36 + 2
             = 8,640 + 129,600 + 38
             = 138,278 blocks
```

---

## Genesis Block

### Mainnet Genesis

| Field | Value |
|-------|-------|
| **Timestamp** | `1,777,608,000,000` ms (2026-05-01 12:00:00 CST / 2026-05-01 04:00:00 UTC) |
| **Bits** | `100,000,000` |
| **Nonce** | `83,330` |
| **Version** | `0` |
| **Hash** | `c3003a4836c39a7965fc92241a2a2e9504399fe2694ff25276f7ea47dde7a783` |
| **Merkle Root** | `6a1331d7b89bab3d895862e7129de245640248ec3ab1257001a3782e65abf2ff` |
| **Coinbase Payload Message** | `Hoosat is there for you, decentralized open low entry crypto in the world, you are there for Hoosat.` |

### Testnet Genesis

| Field | Value |
|-------|-------|
| **Timestamp** | `0x17c5f62fbb6` ms (≈ 1,637,675,732,966 ms) |
| **Bits** | `0x1f0346dc` (≈ 520,388,316) |
| **Nonce** | `0x14582` (83,330) |
| **Hash** | `8ec2316345a5f563943ac1d362ac7e6c272dd6c7bb8dc857d390a6a9c0c320f5` |
| **Merkle Root** | `db85d757761c4877898d862e17c471a12fceed07cf995f696913200737860e16` |
| **Coinbase Payload Message** | `hoosat-testnet` |

**Note:** Testnet B5 and Testnet B10 share the same genesis block as the standard Testnet.

### Devnet Genesis

| Field | Value |
|-------|-------|
| **Timestamp** | `0x11e9db49828` ms (≈ 1,231,781,303,200 ms) |
| **Bits** | `525,264,379` |
| **Nonce** | `0x48e5e` (298,590) |
| **Hash** | `8777be530646c64538007915edbb3e6f9662dda9c541ea8af0576f8445155d07` |
| **Merkle Root** | `81e51eeb2ffdd99ac9525420b54ff2ae753f2153fb28ebce397cec5d7bc3449d` |
| **Coinbase Payload Message** | `eiyaro-devnet` |

### Simnet Genesis

| Field | Value |
|-------|-------|
| **Timestamp** | `1,777,608,000,000` ms (2026-05-01 12:00:00 CST) |
| **Bits** | `511,705,087` |
| **Nonce** | `83,330` |
| **Hash** | `679ac6964c234a85367b6119d7032b58f64356b51a79d8fecce5613582b77686` |
| **Merkle Root** | `6adef11f5f4b0cf6a6431ca97b970f4c1d75a66b793a020ff32ef9b7b9b0e9d` |
| **Coinbase Payload Message** | `eiyaro-simnet` |

### Genesis Coinbase Transaction Structure

All genesis blocks share the same coinbase transaction structure:

- **Subnetwork**: `SubnetworkIDCoinbase` (native coinbase subnetwork)
- **Transaction inputs**: None (empty array)
- **Transaction outputs**: None (empty array — no spendable outputs in genesis)
- **Payload format**:

| Bytes | Field | Mainnet Value | Testnet Value |
|-------|-------|---------------|---------------|
| 0–7 | Blue score | `0x0000000000000000` | `0x0000000000000000` |
| 8–15 | Subsidy (little-endian) | `0x00E1F50500000000` (= 100,000,000 Sey = 1 EY) | `0x00E1F50500000000` |
| 16–17 | Script version | `0x0000` | `0x0000` |
| 18 | Varint (payload length) | `0x01` | `0x01` |
| 19 | OP_FALSE | `0x00` | `0x00` |
| 20+ | Script public key + message | See per-network messages above | See per-network messages above |

---

## Subsidy & Economic Model

### Unit Definitions

Eiyaro uses the following denominations:

| Unit | Value in Sey | Value in EY |
|------|-------------|-------------|
| 1 Sey (smallest unit) | 1 | 0.00000001 |
| 1 EY | 100,000,000 Sey | 1 |

Defined in `constants.go`:
```
SeyPerEY = 100,000,000
MaxSey   = 17,100,000,000 × SeyPerEY = 1,710,000,000,000,000,000 Sey
```

### Block Subsidy Schedule

#### Phase 1: Genesis Reward (Pre-Deflationary Phase)

The initial (pre-deflationary) phase applies from block 0 until DAA score reaches `DeflationaryPhaseDaaScore`:

| Parameter | Value |
|-----------|-------|
| `SubsidyGenesisReward` | 800,000,000 Sey = **8 EY** |
| `PreDeflationaryPhaseBaseSubsidy` | 800,000,000 Sey = **8 EY** |
| `DeflationaryPhaseDaaScore` | **360** |

During this phase, each block rewards 8 EY to the miner.

#### Phase 2: Deflationary Phase

After DAA score 360, the network enters the deflationary phase:

| Parameter | Value |
|-----------|-------|
| `DeflationaryPhaseBaseSubsidy` | 800,000,000 Sey = **8 EY** |
| `DeflationaryPhaseCurveFactor` | **1.0** |

The subsidy decreases by **10% annually** (gentler curve factor of 1.0). The reward continues to decrease each year until it reaches the minimum subsidy floor.

#### Minimum Subsidy Floor

| Parameter | Value |
|-----------|-------|
| `defaultMinimumSubsidy` | 10,000,000 Sey = **0.1 EY** |

The block subsidy will never fall below 0.1 EY, ensuring perpetual miner incentive.

### Coinbase Maturity

| Parameter | Value |
|-----------|-------|
| `BlockCoinbaseMaturity` | **100 blocks** |

Newly mined coins require 100 confirmations before they become spendable. At 20 seconds per block, this is approximately **33 minutes and 20 seconds**.

---

## Address Prefixes

Bech32 addresses in Eiyaro use human-readable prefixes (HRPs) to distinguish networks:

| Network | Bech32 HRP | Go Constant |
|---------|------------|-------------|
| Mainnet | `eiyaro` | `util.Bech32PrefixEiyaro` |
| Devnet | `eiyaro` | `util.Bech32PrefixEiyaro` |
| Testnet | `eiyarotest` | `util.Bech32PrefixEiyaroTest` |
| Testnet B5 | `eiyarotest` | `util.Bech32PrefixEiyaroTest` |
| Testnet B10 | `eiyarotest` | `util.Bech32PrefixEiyaroTest` |
| Simnet | `eiyarosim` | `util.Bech32PrefixEiyaroSim` |

The `Bech32Prefix` constants are defined in [address.go](file:///d:/Eiyaro/Eiyaro/util/address.go#L44-L64):

```go
Bech32PrefixUnknown Bech32Prefix = iota  // 0
Bech32PrefixEiyaro                        // 1 → "eiyaro"
Bech32PrefixEiyaroDev                     // 2 → "eydev"
Bech32PrefixEiyaroTest                    // 3 → "eiyarotest"
Bech32PrefixEiyaroSim                     // 4 → "eiyarosim"
```

### WIF Private Key Prefixes

Wallet Import Format (WIF) private keys are prefixed with a network-specific byte:

| Network | `PrivateKeyID` | Uncompressed WIF Prefix | Compressed WIF Prefix |
|---------|---------------|------------------------|----------------------|
| Mainnet | `0x80` | `5...` | `K...` |
| Testnet | `0x80` | `5...` | `K...` |
| Testnet B5 | `0xef` | `9...` | `c...` |
| Testnet B10 | `0xef` | `9...` | `c...` |
| Devnet | `0x80` | `5...` | `K...` |
| Simnet | `0x64` | `4...` | `F...` |

---

## DNS Seeds

DNS seed nodes are used for initial peer discovery:

### Mainnet DNS Seeds

```
seed.eiyaro.org
seed1.eiyaro.org
seed2.eiyaro.org
seed3.eiyaro.org
```

### Testnet / Testnet B5 / Testnet B10 DNS Seeds

(Same as Mainnet)

```
seed.eiyaro.org
seed1.eiyaro.org
seed2.eiyaro.org
seed3.eiyaro.org
```

### Devnet DNS Seeds

```
(none — private network, no public seeds)
```

### Simnet DNS Seeds

```
(none — simulation network, peer discovery is manual)
```

---

## Consensus Governance (BIP0009)

All networks support BIP0009-style rule change voting:

| Network | `MinerConfirmationWindow` | `RuleChangeActivationThreshold` | Threshold % |
|---------|--------------------------|-------------------------------|-------------|
| Mainnet | 2,016 | 1,916 | 95% |
| Testnet | 2,016 | 1,916 | 95% |
| Testnet B5 | 2,016 | 1,512 | 75% |
| Testnet B10 | 2,016 | 1,512 | 75% |
| Devnet | 2,016 | 1,916 | 95% |
| Simnet | 100 | 75 | 75% |

The confirmation window is derived from: `target PoW timespan / target PoW spacing`.

---

## Additional Parameters

| Parameter | Value | Networks |
|-----------|-------|----------|
| `SkipProofOfWork` | *(zero value — PoW enforced)* | All |
| `RelayNonStdTxs` | `false` | All |
| `EnableNonNativeSubnetworks` | `false` | All |
| `AcceptUnroutable` | `false` | All |
| `MaxBlockLevel` | `225` | Mainnet, Testnet, Testnet B5, Devnet |
| `MaxBlockLevel` | `250` | Testnet B10, Simnet |
| `DisallowDirectBlocksOnTopOfGenesis` | `true` | Mainnet, Testnet, Devnet |

---

## Usage Examples

### Verifying Which Network a Node Is Running On

Examine the RPC port the node listens on:

| RPC Port | Network |
|----------|---------|
| `42420` | Mainnet |
| `42422` | Testnet / Testnet B5 / Testnet B10 |
| `42424` | Simnet |
| `42426` | Devnet |

Check the Bech32 address prefix of any generated address:

```
Mainnet:  eiyaro:...
Testnet:  eiyarotest:...
Simnet:   eiyarosim:...
Devnet:   eiyaro:...
```

**Note:** Devnet shares the `eiyaro` Bech32 prefix with Mainnet, so address prefix alone is insufficient to distinguish them. Use the RPC/P2P port instead.

### Selecting the Correct Network for Development

| Use Case | Recommended Network |
|----------|-------------------|
| Production deployment | `MainnetParams` |
| Public testing / faucets | `TestnetParams` |
| Stress-testing high-throughput block production | `TestnetParamsB5` or `TestnetParamsB10` |
| Local development / CI | `DevnetParams` |
| Private multi-node simulation | `SimnetParams` |

### Code Example: Switching Networks

From [doc.go](file:///d:/Eiyaro/Eiyaro/domain/dagconfig/doc.go#L23-L56):

```go
package main

import (
    "flag"
    "fmt"
    "log"

    "github.com/Eiyaro/Eiyaro/util"
    "github.com/Eiyaro/Eiyaro/domain/dagconfig"
)

var testnet = flag.Bool("testnet", false, "operate on the testnet eiyaro network")

var dagParams = &dagconfig.MainnetParams

func main() {
    flag.Parse()

    if *testnet {
        dagParams = &dagconfig.TestnetParams
    }

    pubKey := make([]byte, 32)
    addr, err := util.NewAddressPubKey(pubKey, dagParams)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(addr)
}
```

---

## References

- **PHANTOM Paper**: [https://eprint.iacr.org/2018/104.pdf](https://eprint.iacr.org/2018/104.pdf) — GHOSTDAG K parameter derivation
- **DAA Design**: Timestamp manipulation handling in GHOSTDAG
- **Finality & Pruning**: Design rationale for invalidation rules in pruning
- **Pruning Proof**: [https://github.com/eiyaronet/research/issues/3](https://github.com/eiyaronet/research/issues/3)
- **Eiyaro Launch Plan**: [https://hashdag.medium.com/eiyaro-launch-plan-9a63f4d754a6](https://hashdag.medium.com/eiyaro-launch-plan-9a63f4d754a6)

---

## FAQ

### How do I verify which network my node is on?

Check the RPC port: `42420` = mainnet, `42422` = testnet, `42424` = simnet, `42426` = devnet. You can also inspect a generated address prefix (`eiyaro:` for mainnet/devnet, `eiyarotest:` for testnet, `eiyarosim:` for simnet) or call `GetCurrentNetwork` via the RPC API. Note that devnet shares the `eiyaro` prefix with mainnet, so the port is the most reliable indicator for devnet.

### Can I change the network parameters for development?

Yes, you can switch networks by selecting the appropriate `dagconfig.Params` variant (`MainnetParams`, `TestnetParams`, `DevnetParams`, `SimnetParams`) in your node configuration. For custom parameter overrides, use the `Factory.Config` struct which allows adjusting individual settings like block time, subsidy, and K. Testnet B5 and B10 variants exist specifically for stress-testing with larger merge set size limits.

### What happens to testnet coins after a reset?

Testnet coins have no real-world value and should be considered ephemeral. Since testnet networks share a distinct genesis block from mainnet, mainnet coins are never at risk. Testnet resets may occur during protocol upgrades, at which point all testnet balances are effectively wiped and the chain restarts from the testnet genesis block.

### How do I create a custom network configuration?

Define a new `dagconfig.Params` struct with your desired values and pass it to the consensus factory. You must provide a unique genesis block with its own timestamp, bits, nonce, and hash. Ensure your network has distinct port numbers (P2P and RPC), a unique `Name` field, and appropriate DNS seeds (or none for private networks). All networks are mutually incompatible, so addresses from one network cannot be used on another.

### How long must I wait before newly mined coins become spendable?

The coinbase maturity period is `BlockCoinbaseMaturity = 100` blocks. At 20 seconds per block, this equals approximately 33 minutes and 20 seconds. Any transaction attempting to spend a coinbase output before 100 confirmations will be rejected by the network.