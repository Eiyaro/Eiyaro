# Eiyaro Bech32 Addressing

## Overview

Eiyaro uses a custom Bech32 encoding scheme for all blockchain addresses. This encoding provides human-readable address prefixes, built-in error detection via a BCH checksum, and a consistent format across all address types. Every address on the Eiyaro network is represented as a Bech32 string of the form `prefix:base32data`.

The address system supports multiple cryptographic schemes — Schnorr and ECDSA — and multiple output types including pay-to-pubkey (P2PK), pay-to-pubkey-hash (P2PKH), and pay-to-script-hash (P2SH).

---

## Bech32 Encoding

### Character Set

Eiyaro Bech32 uses a 32-character alphabet:

```
qpzry9x8gf2tvdw0s3jn54khce6mua7l
```

Each character represents a 5-bit value (0–31), corresponding to its index in this charset. The charset is ordered so that visually similar characters (e.g., `1`/`l`, `0`/`o`) are avoided, reducing the risk of transcription errors.

| Index | Char | Index | Char | Index | Char | Index | Char |
|-------|------|-------|------|-------|------|-------|------|
| 0     | q    | 8     | g    | 16    | 3    | 24    | a    |
| 1     | p    | 9     | f    | 17    | j    | 25    | 7    |
| 2     | z    | 10    | 2    | 18    | n    | 26    | l    |
| 3     | r    | 11    | t    | 19    | 5    | 27    | (unused) |
| 4     | y    | 12    | v    | 20    | 4    | 28    | (unused) |
| 5     | 9    | 13    | d    | 21    | k    | 29    | (unused) |
| 6     | x    | 14    | w    | 22    | h    | 30    | (unused) |
| 7     | 8    | 15    | 0    | 23    | c    | 31    | (unused) |

Note: The charset has only 32 characters (indices 0–31); values 27–31 are reserved and should never appear in valid Bech32 data.

### Separator

Eiyaro Bech32 uses the colon character (`:`) as the separator between the human-readable prefix and the data part, unlike standard Bech32 (BIP 0173) which uses `1`. The last occurrence of `:` in the string marks the boundary.

**Format:** `prefix:base32data`

### Checksum

Eiyaro uses an **8-character** checksum (standard Bech32 uses 6). The checksum is a 40-bit BCH code defined over GF(2⁵) with the generator polynomial:

```
generator = [0x98f2bc8e61, 0x79b76d99e2, 0xf33e5fb3c4, 0xae2eabe2a8, 0x1e4f43e470]
```

The checksum is computed over the prefix (converted to 5-bit values from the lower 5 bits of each ASCII character), a zero separator, and the payload data. It can detect up to 6 errors anywhere in the address and up to 8 consecutive errors (burst errors).

### Prefix (HRP — Human Readable Part)

The prefix is a human-readable string that identifies the network. It uses only printable ASCII characters (33–126). During checksum computation, each character of the prefix contributes its lower 5 bits (`char & 31`), making the prefix **case-insensitive** — `Eiyaro` and `EIYARO` produce the same checksum.

### Data Part Encoding

The data part consists of the version byte prepended to the payload, then converted from 8-bit groups to 5-bit groups (base32 encoding). The conversion uses zero-padding for the final incomplete group. The 5-bit values are then mapped to the charset to produce the final string.

---

## Address Types

Eiyaro defines five address types, each identified by a unique version byte. All addresses implement the `Address` interface from `util/address.go`.

| Address Type | Description | Version Byte | Payload Size | Public Key Size |
|---|---|---|---|---|
| `AddressPublicKey` (P2PK Schnorr) | Pay-to-pubkey using Schnorr signatures | `0x00` | 32 bytes | 32 bytes |
| `AddressPublicKeyECDSA` (P2PK ECDSA) | Pay-to-pubkey using ECDSA secp256k1 | `0x01` | 33 bytes | 33 bytes |
| `AddressPublicKeyHash` (P2PKH Schnorr) | Pay-to-pubkey-hash (Blake2b of Schnorr pubkey) | `0x02` | 32 bytes | 32 bytes |
| `AddressPublicKeyHashECDSA` (P2PKH ECDSA) | Pay-to-pubkey-hash (Blake2b of ECDSA pubkey) | `0x03` | 32 bytes | 33 bytes |
| `AddressScriptHash` (P2SH) | Pay-to-script-hash (Blake2b of redeem script) | `0x08` | 32 bytes | N/A |

### AddressPublicKey (P2PK — Version `0x00`)

The simplest address type. The payload is the full 32-byte Schnorr public key. In a transaction output script, this corresponds to a simple `OP_CHECKSIG` verification against the embedded public key.

- **Public key size:** 32 bytes
- **Hash function:** None (raw public key)
- **Payload:** `publicKey[0..31]`

### AddressPublicKeyECDSA (P2PK ECDSA — Version `0x01`)

Equivalent to P2PK but for ECDSA secp256k1 keys. The payload is the full 33-byte compressed ECDSA public key.

- **Public key size:** 33 bytes
- **Hash function:** None (raw public key)
- **Payload:** `publicKey[0..32]`

### AddressPublicKeyHash (P2PKH Schnorr — Version `0x02`)

The payload is a 32-byte Blake2b-256 hash of the Schnorr public key. This is the most common address type for user wallets. The corresponding script template is:

```
OP_DUP OP_BLAKE2B <32-byte hash> OP_EQUALVERIFY OP_CHECKSIG
```

- **Public key size:** 32 bytes
- **Hash function:** Blake2b-256
- **Payload:** `Blake2b(publicKey)`

### AddressPublicKeyHashECDSA (P2PKH ECDSA — Version `0x03`)

The payload is a 32-byte Blake2b-256 hash of the 33-byte compressed ECDSA public key. The corresponding script template is:

```
OP_DUP OP_BLAKE2B <32-byte hash> OP_EQUALVERIFY OP_CHECKSIGECDSA
```

- **Public key size:** 33 bytes
- **Hash function:** Blake2b-256
- **Payload:** `Blake2b(publicKey)`

### AddressScriptHash (P2SH — Version `0x08`)

The payload is a 32-byte Blake2b-256 hash of a redeem script. This enables custom spending conditions such as multisig, timelocks, and other advanced scripts.

- **Hash function:** Blake2b-256
- **Payload:** `Blake2b(redeemScript)`

---

## Address Prefix System

Each Eiyaro network is identified by a unique Bech32 prefix. The prefix is stored in the `Bech32Prefix` enum type in `util/address.go`.

| Prefix String | Bech32Prefix Constant | Network |
|---|---|---|
| `eiyaro` | `Bech32PrefixEiyaro` | Mainnet |
| `eiyarotest` | `Bech32PrefixEiyaroTest` | Testnet |
| `eydev` | `Bech32PrefixEiyaroDev` | Devnet |
| `eiyarosim` | `Bech32PrefixEiyaroSim` | Simulation network |

The mapping from string to enum is defined in the `stringsToBech32Prefixes` map, and the `ParsePrefix` function converts a prefix string to its `Bech32Prefix` value.

**Example mainnet address:**
```
eiyaro:qpc0z7q3q4q5q6q7q8q9qaqbqcqdqeqfqgqhqiqjqkqlqmqnqoqpqqqrqsqtquqvq
```

---

## Address Encoding Flow

The encoding process is implemented in the `EncodeAddress` method of each address type, which calls the internal `encodeAddress` function.

```
encodeAddress(prefix, payload, version):
    1. Prepend version byte to payload
    2. Convert combined bytes from 8-bit groups to 5-bit groups (base32)
    3. Compute 8-character BCH checksum over (prefix + 0 + 5-bit data)
    4. Concatenate: prefix + ":" + base32(data + checksum)
```

**Step-by-step example (P2PKH Schnorr, mainnet):**

1. Start with a 32-byte Schnorr public key.
2. Compute `payload = Blake2b(publicKey)` → 32 bytes.
3. Select version byte: `0x02` (P2PKH).
4. Prepend version: `[0x02, payload[0], ..., payload[31]]` → 33 bytes.
5. Convert 8-bit to 5-bit: 33 bytes × 8 bits / 5 bits = 53 5-bit values (with 4 bits of zero-padding).
6. Compute checksum over `(prefix_lower_5_bits, 0, 5bit_data)` → 8 bytes (40 bits).
7. Map each 5-bit value plus checksum through charset.
8. Final address: `eiyaro:...` (prefix + `:` + base32 string).

---

## Address Decoding

The `DecodeAddress` function in `util/address.go` performs the full decode-and-validate pipeline:

```
DecodeAddress(addrString, expectedPrefix):
    1. Split string at the last ':' → prefix, dataPart
    2. Validate character set (ASCII 33–126 only)
    3. Validate case consistency (all lower or all upper)
    4. Decode dataPart from base32 → 5-bit byte array
    5. Verify checksum via polyMod == 0
    6. Extract payload without checksum → 5-bit array
    7. Convert 5-bit to 8-bit → [version, payload...]
    8. Parse prefix via ParsePrefix → Bech32Prefix enum
    9. If expectedPrefix != Unknown, validate prefix match
    10. Switch on version byte to construct typed Address:
        - 0x00 → AddressPublicKey
        - 0x01 → AddressPublicKeyECDSA
        - 0x02 → AddressPublicKeyHash
        - 0x03 → AddressPublicKeyHashECDSA
        - 0x08 → AddressScriptHash
        - other → ErrUnknownAddressType
    11. Validate payload length for the specific address type
    12. Return typed Address interface
```

### Checksum Verification

The checksum is verified using `polyMod`. The polynomial modulo operation is computed over `(prefix_lower_5_bits + 0 + payload_including_checksum)`. If the result is `0`, the checksum is valid.

---

## Address Interface

All address types implement the `Address` interface defined in `util/address.go`:

```go
type Address interface {
    // String returns the Bech32-encoded address string.
    // Equivalent to EncodeAddress for all Eiyaro address types.
    String() string

    // EncodeAddress returns the full Bech32-encoded address string
    // including prefix, separator, data, and checksum.
    EncodeAddress() string

    // ScriptAddress returns the raw bytes that are embedded
    // in a transaction output's locking script.
    ScriptAddress() []byte

    // Prefix returns the Bech32Prefix enum value
    // identifying which network this address belongs to.
    Prefix() Bech32Prefix

    // IsForPrefix returns true if this address
    // belongs to the specified network.
    IsForPrefix(prefix Bech32Prefix) bool
}
```

### Method Behavior by Address Type

| Method | P2PK (0x00) | P2PK ECDSA (0x01) | P2PKH (0x02) | P2PKH ECDSA (0x03) | P2SH (0x08) |
|---|---|---|---|---|---|
| `ScriptAddress()` | Full pubkey (32b) | Full pubkey (33b) | Blake2b hash (32b) | Blake2b hash (32b) | Script Blake2b hash (32b) |
| `EncodeAddress()` | Bech32 with v=0x00 | Bech32 with v=0x01 | Bech32 with v=0x02 | Bech32 with v=0x03 | Bech32 with v=0x08 |
| `String()` | Same as `EncodeAddress()` | Same as `EncodeAddress()` | Same as `EncodeAddress()` | Same as `EncodeAddress()` | Same as `EncodeAddress()` |

---

## Construction Functions

### AddressPublicKey (P2PK Schnorr)

```go
// Public key must be exactly 32 bytes.
addr, err := util.NewAddressPublicKey(publicKey, util.Bech32PrefixEiyaro)
```

### AddressPublicKeyECDSA (P2PK ECDSA)

```go
// Public key must be exactly 33 bytes (compressed).
addr, err := util.NewAddressPublicKeyECDSA(publicKey, util.Bech32PrefixEiyaro)
```

### AddressPublicKeyHash (P2PKH Schnorr)

```go
// From a 32-byte Schnorr public key — hashes it internally.
addr, err := util.NewAddressPublicKeyHash(publicKey, util.Bech32PrefixEiyaro)

// From a pre-computed 32-byte Blake2b hash.
addr, err := util.NewAddressPublicKeyHashFromHash(hash, util.Bech32PrefixEiyaro)
```

### AddressPublicKeyHashECDSA (P2PKH ECDSA)

```go
// From a 33-byte compressed ECDSA public key — hashes it internally.
addr, err := util.NewAddressPublicKeyHashECDSA(publicKey, util.Bech32PrefixEiyaro)

// From a pre-computed 32-byte Blake2b hash.
addr, err := util.NewAddressPublicKeyHashECDSAFromHash(hash, util.Bech32PrefixEiyaro)
```

### AddressScriptHash (P2SH)

```go
// From a redeem script bytecode — hashes it internally.
addr, err := util.NewAddressScriptHash(redeemScript, util.Bech32PrefixEiyaro)

// From a pre-computed 32-byte Blake2b hash.
addr, err := util.NewAddressScriptHashFromHash(scriptHash, util.Bech32PrefixEiyaro)
```

---

## DecodeAddress

The `DecodeAddress` function is the single entry point for parsing any Eiyaro address string:

```go
func DecodeAddress(addr string, expectedPrefix Bech32Prefix) (Address, error)
```

**Parameters:**
- `addr`: The full Bech32 address string (e.g., `"eiyaro:qpc0z7q..."`).
- `expectedPrefix`: The network prefix to validate against. Pass `Bech32PrefixUnknown` to skip network validation (accept any valid prefix).

**Returns:**
- A typed `Address` interface (one of `*AddressPublicKey`, `*AddressPublicKeyECDSA`, `*AddressPublicKeyHash`, `*AddressPublicKeyHashECDSA`, `*AddressScriptHash`).
- An error if the address is malformed, has an invalid checksum, belongs to the wrong network, or has an unknown version byte.

**Internal dispatch by version byte:**

| Version | Constructed Type |
|---|---|
| `0x00` | `*AddressPublicKey` |
| `0x01` | `*AddressPublicKeyECDSA` |
| `0x02` | `*AddressPublicKeyHash` |
| `0x03` | `*AddressPublicKeyHashECDSA` |
| `0x08` | `*AddressScriptHash` |
| other | `ErrUnknownAddressType` |

---

## Usage Examples

### Example 1: Encoding a P2PKH Address (Schnorr)

```go
package main

import (
    "fmt"
    "github.com/Eiyaro/Eiyaro/util"
)

func main() {
    // A 32-byte Schnorr public key (example).
    pubKey := make([]byte, util.PublicKeySize)
    // ... fill pubKey from key derivation ...

    addr, err := util.NewAddressPublicKeyHash(pubKey, util.Bech32PrefixEiyaro)
    if err != nil {
        panic(err)
    }

    fmt.Println(addr.EncodeAddress())
    // Output: eiyaro:qp...
}
```

### Example 2: Encoding a P2SH Address (Multisig)

```go
import "github.com/Eiyaro/Eiyaro/util"

func multisigToAddress(redeemScript []byte) string {
    addr, err := util.NewAddressScriptHash(redeemScript, util.Bech32PrefixEiyaroTest)
    if err != nil {
        return ""
    }
    return addr.EncodeAddress()
}
```

### Example 3: Decoding and Validating an Address

```go
import (
    "fmt"
    "github.com/Eiyaro/Eiyaro/util"
)

func validateAddress(addrStr string) {
    addr, err := util.DecodeAddress(addrStr, util.Bech32PrefixEiyaro)
    if err != nil {
        fmt.Printf("Invalid address: %v\n", err)
        return
    }

    fmt.Printf("Valid mainnet address\n")
    fmt.Printf("  Type: %T\n", addr)
    fmt.Printf("  Prefix: %s\n", addr.Prefix())
    fmt.Printf("  ScriptAddress: %x\n", addr.ScriptAddress())

    // Type-switch for type-specific processing.
    switch a := addr.(type) {
    case *util.AddressPublicKeyHash:
        fmt.Printf("  Blake2b Hash: %x\n", a.HashBlake2b())
    case *util.AddressPublicKeyHashECDSA:
        fmt.Printf("  Blake2b Hash: %x\n", a.HashBlake2b())
    case *util.AddressScriptHash:
        fmt.Printf("  Script Hash: %x\n", a.HashBlake2b())
    }
}
```

### Example 4: Checking Address Network Prefix

```go
func isMainnet(addr util.Address) bool {
    return addr.IsForPrefix(util.Bech32PrefixEiyaro)
}

func isTestnet(addr util.Address) bool {
    return addr.IsForPrefix(util.Bech32PrefixEiyaroTest)
}

func validateNetwork(addr util.Address, expectedPrefix util.Bech32Prefix) bool {
    return addr.Prefix() == expectedPrefix
}
```

### Example 5: Converting Between Address Representations

```go
func pubKeyToAddressHash(pubKey []byte, prefix util.Bech32Prefix) (string, error) {
    // Create P2PK address from raw public key.
    pkAddr, err := util.NewAddressPublicKey(pubKey, prefix)
    if err != nil {
        return "", err
    }

    // Re-decode to get the raw bytes, then construct P2PKH.
    hash := util.HashBlake2b(pubKey)
    pkhAddr, err := util.NewAddressPublicKeyHashFromHash(hash, prefix)
    if err != nil {
        return "", err
    }

    _ = pkAddr
    return pkhAddr.EncodeAddress(), nil
}
```

---

## Bech32 Algorithm Internals

This section documents the low-level Bech32 implementation in `util/bech32/bech32.go` for completeness.

### Constants

```go
const charset        = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
const checksumLength = 8  // 8-character checksum (40 bits)
```

### Generator Polynomial

The BCH checksum uses the following generator polynomial in GF(2⁵):

```go
var generator = []int{
    0x98f2bc8e61,
    0x79b76d99e2,
    0xf33e5fb3c4,
    0xae2eabe2a8,
    0x1e4f43e470,
}
```

### polyMod

The `polyMod` function computes the checksum polynomial modulo. It processes each 5-bit value through a finite field division using the generator. The final result XORed with `1` must be zero for a valid checksum.

```
polyMod(values):
    check = 1
    for each value in values:
        top = check >> 35
        check = ((check & 0x07ffffffff) << 5) ^ value
        for i in 0..4:
            if (top >> i) & 1 == 1:
                check ^= generator[i]
    return check ^ 1
```

### checksum Calculation

```
calculateChecksum(prefix, payload):
    prefix5 = [char & 31 for char in prefix]
    data = prefix5 + [0] + ints(payload) + [0,0,0,0,0,0,0,0]
    result = polyMod(data)
    return [(result >> (5 * (7 - i))) & 31 for i in 0..7]
```

### convertBits

Converts between 8-bit groups and 5-bit groups, iteratively packing/unpacking bits:

```
convertBits(data, fromBits=8, toBits=5, pad=true):
    // Left-justify each input byte.
    // For each input byte, extract 'fromBits' bits,
    // pack them into 'toBits'-wide output bytes.
    // If 'pad' is true, zero-pad the final incomplete group.
```

---

## Notes & Caveats

### Separator Character
Eiyaro uses `:` (colon) as the Bech32 separator, **not** `1` as in BIP 0173 standard Bech32. This means Eiyaro addresses are not compatible with standard Bech32 decoders and vice versa.

### Checksum Length
Eiyaro uses an **8-character checksum** (40 bits), which provides stronger error detection than the standard 6-character (30-bit) Bech32 checksum. This allows detection of up to 8 consecutive errors and up to 6 arbitrary errors in the address string.

### Case Sensitivity
The data portion of an Eiyaro Bech32 address is **case-sensitive** — it must be either all lowercase or all uppercase. Mixing cases within the data part produces an invalid address. The prefix, however, is **case-insensitive** during validation (its lower 5 bits are used for checksum computation).

### Address Length Constraints
- Minimum length: 10 characters (prefix + `:` + at least 8 checksum characters)
- Maximum length: 90 characters
- Only ASCII characters between 33 and 126 are permitted

### Error Detection
The BCH checksum provides strong guarantees:
- Detects up to **6 errors** anywhere in the address.
- Detects up to **8 consecutive errors** (burst errors).
- Combined with the length check, the probability of accepting an invalid address is approximately 1 in 2⁴⁰ (~1 in a trillion).

### Version Byte Dispatch
The `DecodeAddress` function dispatches on the version byte. Unknown version bytes produce `ErrUnknownAddressType`. If new address types are added in the future, the version byte switch in `DecodeAddress` must be updated.

### Hash Function
All hash-based address types (P2PKH, P2PKH ECDSA, P2SH) use **Blake2b-256** (32-byte output). There is no RIPEMD-160 or SHA-256 usage in Eiyaro addressing — this is a deliberate deviation from Bitcoin-style addressing.

---

## Reference

| File | Description |
|---|---|
| `util/bech32/bech32.go` | Low-level Bech32 encoder/decoder, checksum computation, bit conversion |
| `util/address.go` | Address interface, all address types, prefix system, `DecodeAddress` |

**Source:** `github.com/Eiyaro/Eiyaro`

---

## FAQ

### Why does Eiyaro use ':' instead of '1' as the Bech32 separator?

Standard Bech32 (BIP 0173) uses `1` as the separator to avoid visual confusion with base32 characters. Eiyaro uses `:` (colon) instead, which is also absent from the base32 character set and provides the same visual separation guarantee. The colon additionally avoids issues with some URI-scheme parsers that treat `1` ambiguously. This means Eiyaro addresses are not compatible with standard Bech32 decoders.

### How do I convert between address types?

To convert a P2PK address (version `0x00` or `0x01`) to a P2PKH address (version `0x02` or `0x03`), hash the public key with Blake2b-256 and construct a new address using the appropriate constructor: `util.NewAddressPublicKeyHashFromHash(hash, prefix)` for Schnorr or `util.NewAddressPublicKeyHashECDSAFromHash(hash, prefix)` for ECDSA. There is no reverse conversion (P2PKH to P2PK) because the hash is one-way.

### Can I send funds to a testnet address from mainnet?

No. Mainnet and testnet addresses have different Bech32 prefixes (`eiyaro:` vs `eiyarotest:`) and different network magic values. A mainnet node will reject transactions targeting testnet addresses, and the networks are completely isolated. Attempting this will fail at the address validation step.

### Are Eiyaro addresses case-sensitive?

The data part (after the colon) is case-sensitive and must be either all lowercase or all uppercase — mixed case is invalid. The prefix part is case-insensitive during checksum validation because only the lower 5 bits of each character are used. For maximum compatibility, always use lowercase addresses.

### What makes Eiyaro's Bech32 checksum stronger than standard Bech32?

Eiyaro uses an 8-character (40-bit) BCH checksum instead of the standard 6-character (30-bit) checksum. This provides detection of up to 6 arbitrary errors anywhere in the address and up to 8 consecutive burst errors, compared to the standard's 4-arbitrary / 5-burst detection. The probability of accepting an invalid address is approximately 1 in 2⁴⁰ (~1 in a trillion).