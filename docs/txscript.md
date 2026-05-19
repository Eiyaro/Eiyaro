# Eiyaro Transaction Script Engine (txscript)

## Overview

The Eiyaro transaction script engine (`txscript`) is a stack-based virtual machine that executes a Forth-like scripting language to validate transaction outputs. It is derived from Bitcoin's script system and supports both **Schnorr** and **ECDSA** signature verification. The package provides data structures and functions to parse, build, and execute transaction scripts in a production-grade, consensus-compatible manner.

The scripting language consists of opcodes organized into categories: data pushing, stack manipulation, arithmetic, bitwise logic, conditional branching, cryptographic hashing, and signature verification. Scripts are processed left-to-right without loops, providing a deterministic and bounded execution environment.

---

## Architecture

The script engine is composed of several tightly coupled components:

| Component | File | Role |
|-----------|------|------|
| **Engine** | [engine.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/engine.go) | Virtual machine that executes script opcodes step-by-step |
| **Stack** | [stack.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/stack.go) | Immutable data stack and alt stack for opcode operands |
| **ScriptBuilder** | [scriptbuilder.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/scriptbuilder.go) | Fluent API for programmatic script construction with canonical encoding |
| **Opcode Table** | [opcode.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/opcode.go) | Complete 256-entry opcode lookup table with handler functions |
| **Standard Scripts** | [standard.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/standard.go) | Recognized standard script templates (P2PK, P2PKH, P2SH) |
| **Signing** | [sign.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/sign.go) | Signature creation and transaction signing logic |
| **Script Parsing** | [script.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/script.go) | Raw script byte-to-parsed-opcode deserialization |
| **Script Number** | [scriptnum.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/scriptnum.go) | Little-endian variable-length integer encoding/decoding |
| **SigCache** | [sigcache.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/sigcache.go) | LRU-based Schnorr signature verification cache |
| **SigCache ECDSA** | [sigcache_ecdsa.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/sigcache_ecdsa.go) | LRU-based ECDSA signature verification cache |
| **Errors** | [error.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/error.go) | Typed error codes with `ErrorCode` enumeration |

---

## Script Execution Model

### Two-Script Model

Every transaction input is validated by concatenating two scripts:

1. **Unlocking Script** (`scriptSig` / signature script) — provided by the spender in the transaction input. This script is **push-only**: it may only contain data push opcodes (opcodes `<= OP_16`). It provides the data (signatures, public keys, redeem scripts) needed to satisfy the locking script.

2. **Locking Script** (`scriptPubKey` / public key script) — embedded in the previous transaction output. This script defines the conditions that must be satisfied to spend the output.

The two scripts are executed sequentially: first the unlocking script pushes its data onto the stack, then the locking script executes to validate those stack items.

### Stack-Based Execution

The engine maintains two stacks:

- **Data Stack** (`dstack`): The primary stack where all opcode operands reside. Most opcodes pop inputs from and push results to this stack.
- **Alt Stack** (`astack`): An auxiliary stack for temporary storage, manipulated via `OP_TOALTSTACK` and `OP_FROMALTSTACK`. The alt stack is cleared between scripts.

### Execution Result

A script is considered **valid** if, after all scripts have executed:

1. The combined stack and alt stack depth never exceeded `MaxStackSize` (244).
2. No more than `MaxOpsPerScript` (201) non-push opcodes were executed per script.
3. No element pushed to the stack exceeded `MaxScriptElementSize` (520 bytes).
4. The final data stack contains **exactly one item** that evaluates to **true** (any non-zero byte sequence, except `[0x80]` which is negative zero).

### Conditional Execution

The engine supports nested conditional branches via `OP_IF`, `OP_NOTIF`, `OP_ELSE`, and `OP_ENDIF`. A conditional stack (`condStack`) tracks whether the current execution context is active (`OpCondTrue`), inactive (`OpCondFalse`), or skipped (nested inside a non-executing branch: `OpCondSkip`). Conditional opcodes are always processed to maintain proper nesting, even in non-executing branches.

For `OP_IF`/`OP_NOTIF`, the top stack item must be either an empty byte slice (false) or `[0x01]` (true) — this is the "minimal if" rule that prevents malleability. Any other value causes an `ErrMinimalIf` error.

### P2SH Execution

Pay-to-Script-Hash (P2SH) introduces a third script execution phase:

1. The unlocking script and locking script execute as normal.
2. If the locking script is recognized as P2SH (`OP_BLAKE2B <32-byte hash> OP_EQUAL`), the engine saves the first script's stack.
3. After the P2SH script validates, the last item on the saved stack (the redeem script) is parsed and executed as a third script against the remaining stack items.

---

## Engine API

The `Engine` struct ([engine.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/engine.go#L36-L53)) is the central virtual machine.

### Construction

```go
func NewEngine(scriptPubKey *externalapi.ScriptPublicKey, tx *externalapi.DomainTransaction,
    txIdx int, flags ScriptFlags, sigCache *SigCache, sigCacheECDSA *SigCacheECDSA,
    sighashReusedValues *consensushashing.SighashReusedValues) (*Engine, error)
```

The `Init` method performs argument validation:

- `txIdx` must be a valid index into `tx.Inputs`.
- Both scripts must not be simultaneously empty.
- The signature script must be **push-only**.
- Scripts must not exceed `MaxScriptSize` (10,000 bytes).
- If the locking script is P2SH, the unlocking script must be push-only.
- Script version must not exceed `constants.MaxScriptPublicKeyVersion`.

### Execution

```go
func (vm *Engine) Execute() error
```

Executes all scripts to completion. Returns `nil` on success, or a typed `Error` on failure. Internally calls `Step()` in a loop.

```go
func (vm *Engine) Step() (done bool, err error)
```

Executes a single opcode and advances the program counter. Returns `done=true` when all scripts have been exhausted.

```go
func (vm *Engine) CheckErrorCondition(finalScript bool) error
```

Checks whether script execution resulted in a valid state: stack contains exactly one true value (for the final script), or the top stack item evaluates to true (for intermediate scripts like P2SH).

### Introspection

```go
func (vm *Engine) DisasmPC() (string, error)
func (vm *Engine) DisasmScript(idx int) (string, error)
```

Return human-readable disassembly of the current program counter or a specific script.

```go
func (vm *Engine) GetStack() [][]byte
func (vm *Engine) SetStack(data [][]byte)
func (vm *Engine) GetAltStack() [][]byte
func (vm *Engine) SetAltStack(data [][]byte)
```

Get/set the contents of the data stack and alt stack.

### Reset

```go
func (vm *Engine) Reset()
```

Clears all engine state for reuse while preserving slice capacities, enabling efficient validation of multiple transactions.

---

## Supported Opcodes

All opcodes are defined as typed constants in [opcode.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/opcode.go#L34-L293). The opcode table has 256 entries, indexed by byte value.

### Constants

Numeric constants that push their value directly onto the stack.

| Name | Hex | Decimal | Description |
|------|-----|---------|-------------|
| `OP_0` / `OP_FALSE` | `0x00` | 0 | Push an empty byte slice (represents 0 / false) |
| `OP_DATA_1` through `OP_DATA_75` | `0x01`–`0x4b` | 1–75 | Push the next `N` bytes |
| `OP_PUSHDATA1` | `0x4c` | 76 | Push: next 1 byte is length, then data |
| `OP_PUSHDATA2` | `0x4d` | 77 | Push: next 2 bytes (LE) are length, then data |
| `OP_PUSHDATA4` | `0x4e` | 78 | Push: next 4 bytes (LE) are length, then data |
| `OP_1NEGATE` | `0x4f` | 79 | Push the value -1 |
| `OP_RESERVED` | `0x50` | 80 | Reserved (fails if executed) |
| `OP_1` / `OP_TRUE` | `0x51` | 81 | Push the value 1 |
| `OP_2` | `0x52` | 82 | Push the value 2 |
| `OP_3` | `0x53` | 83 | Push the value 3 |
| `OP_4` | `0x54` | 84 | Push the value 4 |
| `OP_5` | `0x55` | 85 | Push the value 5 |
| `OP_6` | `0x56` | 86 | Push the value 6 |
| `OP_7` | `0x57` | 87 | Push the value 7 |
| `OP_8` | `0x58` | 88 | Push the value 8 |
| `OP_9` | `0x59` | 89 | Push the value 9 |
| `OP_10` | `0x5a` | 90 | Push the value 10 |
| `OP_11` | `0x5b` | 91 | Push the value 11 |
| `OP_12` | `0x5c` | 92 | Push the value 12 |
| `OP_13` | `0x5d` | 93 | Push the value 13 |
| `OP_14` | `0x5e` | 94 | Push the value 14 |
| `OP_15` | `0x5f` | 95 | Push the value 15 |
| `OP_16` | `0x60` | 96 | Push the value 16 |

### Flow Control

| Name | Hex | Decimal | Description |
|------|-----|---------|-------------|
| `OP_NOP` | `0x61` | 97 | No operation |
| `OP_VER` | `0x62` | 98 | Reserved (fails if executed) |
| `OP_IF` | `0x63` | 99 | Execute following statements if top stack item is true |
| `OP_NOTIF` | `0x64` | 100 | Execute following statements if top stack item is false |
| `OP_VERIF` | `0x65` | 101 | **Always illegal** (fails on program counter) |
| `OP_VERNOTIF` | `0x66` | 102 | **Always illegal** (fails on program counter) |
| `OP_ELSE` | `0x67` | 103 | Invert conditional execution |
| `OP_ENDIF` | `0x68` | 104 | Terminate conditional block |
| `OP_VERIFY` | `0x69` | 105 | Assert top stack item is true, remove it |
| `OP_RETURN` | `0x6a` | 106 | Mark transaction output as unspendable |
| `OP_CHECKLOCKTIMEVERIFY` | `0xb0` | 176 | Assert transaction locktime constraint |
| `OP_CHECKSEQUENCEVERIFY` | `0xb1` | 177 | Assert transaction sequence constraint |

#### IF/ELSE/ENDIF Details

- `OP_IF` / `OP_NOTIF`: Pop the top stack item. In an executing branch, the item must be `[]` or `[0x01]` (minimal if rule). Push `OpCondTrue`, `OpCondFalse`, or `OpCondSkip` onto the condStack.
- `OP_ELSE`: Invert the top condStack entry (`OpCondTrue` ↔ `OpCondFalse`). Fails with `ErrUnbalancedConditional` if condStack is empty.
- `OP_ENDIF`: Pop the condStack. Fails with `ErrUnbalancedConditional` if condStack is empty.

Conditional blocks may be nested arbitrarily.

#### CLTV / CSV Details

- `OP_CHECKLOCKTIMEVERIFY`: Pops an 8-byte little-endian locktime from the stack. The locktime and transaction locktime must be of the same type (both below or both at/above `constants.LockTimeThreshold`). The stack locktime must be `<=` the transaction locktime. The input sequence must not be `constants.MaxTxInSequenceNum` (unfinalized).
- `OP_CHECKSEQUENCEVERIFY`: Pops an 8-byte sequence from the stack. If the disabled bit (`constants.SequenceLockTimeDisabled`) is set in the operand, acts as a NOP. The transaction input sequence must not have the disabled bit set. The masked sequences are compared: masked stack sequence `<=` masked tx sequence.

### Stack Operations

| Name | Hex | Decimal | Description |
|------|-----|---------|-------------|
| `OP_TOALTSTACK` | `0x6b` | 107 | Move top item from data stack to alt stack |
| `OP_FROMALTSTACK` | `0x6c` | 108 | Move top item from alt stack to data stack |
| `OP_2DROP` | `0x6d` | 109 | Remove top 2 items |
| `OP_2DUP` | `0x6e` | 110 | Duplicate top 2 items |
| `OP_3DUP` | `0x6f` | 111 | Duplicate top 3 items |
| `OP_2OVER` | `0x70` | 112 | Copy 2 items before top 2 to top |
| `OP_2ROT` | `0x71` | 113 | Rotate top 6 items left by 2 |
| `OP_2SWAP` | `0x72` | 114 | Swap top 2 items with next 2 |
| `OP_IFDUP` | `0x73` | 115 | Duplicate top item if non-zero |
| `OP_DEPTH` | `0x74` | 116 | Push current stack depth |
| `OP_DROP` | `0x75` | 117 | Remove top item |
| `OP_DUP` | `0x76` | 118 | Duplicate top item |
| `OP_NIP` | `0x77` | 119 | Remove second-to-top item |
| `OP_OVER` | `0x78` | 120 | Copy second-to-top item to top |
| `OP_PICK` | `0x79` | 121 | Copy Nth item back to top (`n` popped from stack) |
| `OP_ROLL` | `0x7a` | 122 | Move Nth item back to top (`n` popped from stack) |
| `OP_ROT` | `0x7b` | 123 | Rotate top 3 items left |
| `OP_SWAP` | `0x7c` | 124 | Swap top 2 items |
| `OP_TUCK` | `0x7d` | 125 | Insert copy of top before second-to-top |

#### Stack Operation Details

All multi-item stack operations use generalized `N`-variant functions on the `stack` type:

- `DupN(n)`: Duplicate top `n` items in order. `OP_DUP` → `DupN(1)`, `OP_2DUP` → `DupN(2)`, `OP_3DUP` → `DupN(3)`.
- `DropN(n)`: Drop top `n` items. `OP_DROP` → `DropN(1)`, `OP_2DROP` → `DropN(2)`.
- `OverN(n)`: Copy the `n`th pair of items. `OP_OVER` → `OverN(1)`, `OP_2OVER` → `OverN(2)`.
- `RotN(n)`: Rotate top `3*n` items left by `n`. `OP_ROT` → `RotN(1)`, `OP_2ROT` → `RotN(2)`.
- `SwapN(n)`: Swap top `n` with next `n`. `OP_SWAP` → `SwapN(1)`, `OP_2SWAP` → `SwapN(2)`.
- `PickN(n)`: Copy stack item at offset `n` to top.
- `RollN(n)`: Move stack item at offset `n` to top.

`OP_IFDUP`: Peeks at the top item. If the bytes are non-zero (via `asBool`), duplicates it. Does not affect the value otherwise.

`OP_DEPTH`: Pushes the current data stack depth as a script number *before* the depth opcode itself is counted.

### Splice Operations

| Name | Hex | Decimal | Status | Description |
|------|-----|---------|--------|-------------|
| `OP_CAT` | `0x7e` | 126 | **Disabled** | Concatenate two strings |
| `OP_SUBSTR` | `0x7f` | 127 | **Disabled** | Extract substring |
| `OP_LEFT` | `0x80` | 128 | **Disabled** | Keep leftmost N bytes |
| `OP_RIGHT` | `0x81` | 129 | **Disabled** | Keep rightmost N bytes |
| `OP_SIZE` | `0x82` | 130 | Active | Push byte length of top item |

`OP_SIZE` peeks at the top item (does not pop it) and pushes its byte length as a script number.

### Bitwise Logic

| Name | Hex | Decimal | Status | Description |
|------|-----|---------|--------|-------------|
| `OP_INVERT` | `0x83` | 131 | **Disabled** | Bitwise NOT |
| `OP_AND` | `0x84` | 132 | **Disabled** | Bitwise AND |
| `OP_OR` | `0x85` | 133 | **Disabled** | Bitwise OR |
| `OP_XOR` | `0x86` | 134 | **Disabled** | Bitwise XOR |
| `OP_EQUAL` | `0x87` | 135 | Active | Byte-for-byte equality comparison |
| `OP_EQUALVERIFY` | `0x88` | 136 | Active | `OP_EQUAL` + `OP_VERIFY` |
| `OP_RESERVED1` | `0x89` | 137 | Reserved | Fails if executed |
| `OP_RESERVED2` | `0x8a` | 138 | Reserved | Fails if executed |

`OP_EQUAL`: Pops two byte arrays from the stack, compares them byte-for-byte, and pushes `1` if equal, `0` otherwise.

`OP_EQUALVERIFY`: Same as `OP_EQUAL` followed by `OP_VERIFY`. Fails with `ErrEqualVerify` if the comparison does not succeed.

### Arithmetic

| Name | Hex | Decimal | Status | Description |
|------|-----|---------|--------|-------------|
| `OP_1ADD` | `0x8b` | 139 | Active | Add 1 |
| `OP_1SUB` | `0x8c` | 140 | Active | Subtract 1 |
| `OP_2MUL` | `0x8d` | 141 | **Disabled** | Multiply by 2 |
| `OP_2DIV` | `0x8e` | 142 | **Disabled** | Divide by 2 |
| `OP_NEGATE` | `0x8f` | 143 | Active | Negate |
| `OP_ABS` | `0x90` | 144 | Active | Absolute value |
| `OP_NOT` | `0x91` | 145 | Active | Logical NOT (0→1, non-zero→0) |
| `OP_0NOTEQUAL` | `0x92` | 146 | Active | Return 1 if non-zero, else 0 |
| `OP_ADD` | `0x93` | 147 | Active | Add two numbers |
| `OP_SUB` | `0x94` | 148 | Active | Subtract (second-to-top − top) |
| `OP_MUL` | `0x95` | 149 | **Disabled** | Multiply |
| `OP_DIV` | `0x96` | 150 | **Disabled** | Divide |
| `OP_MOD` | `0x97` | 151 | **Disabled** | Modulo |
| `OP_LSHIFT` | `0x98` | 152 | **Disabled** | Left shift |
| `OP_RSHIFT` | `0x99` | 153 | **Disabled** | Right shift |
| `OP_BOOLAND` | `0x9a` | 154 | Active | Boolean AND |
| `OP_BOOLOR` | `0x9b` | 155 | Active | Boolean OR |
| `OP_NUMEQUAL` | `0x9c` | 156 | Active | Numeric equality |
| `OP_NUMEQUALVERIFY` | `0x9d` | 157 | Active | `OP_NUMEQUAL` + `OP_VERIFY` |
| `OP_NUMNOTEQUAL` | `0x9e` | 158 | Active | Numeric inequality |
| `OP_LESSTHAN` | `0x9f` | 159 | Active | Less than |
| `OP_GREATERTHAN` | `0xa0` | 160 | Active | Greater than |
| `OP_LESSTHANOREQUAL` | `0xa1` | 161 | Active | Less than or equal |
| `OP_GREATERTHANOREQUAL` | `0xa2` | 162 | Active | Greater than or equal |
| `OP_MIN` | `0xa3` | 163 | Active | Minimum of two values |
| `OP_MAX` | `0xa4` | 164 | Active | Maximum of two values |
| `OP_WITHIN` | `0xa5` | 165 | Active | Check if value is in range `[min, max)` |

#### Arithmetic Details

All arithmetic opcodes interpret stack items as **script numbers** (see [Script Number](#script-number) section). By default, numbers are limited to 4 bytes (range `[-2^31 + 1, 2^31 - 1]`). Results of arithmetic may overflow `int64` and these overflowed results can still be used as booleans (e.g., in `OP_VERIFY`), but will fail if reinterpreted as numbers.

- `OP_NOT`: If argument is 0 → push 1; if non-zero → push 0. Interprets argument as integer, not boolean.
- `OP_0NOTEQUAL`: If argument is 0 → push 0; if non-zero → push 1.
- `OP_BOOLAND`: Both non-zero → 1, else 0.
- `OP_BOOLOR`: Either non-zero → 1, else 0.
- `OP_WITHIN`: Takes `x`, `min`, `max` (top is max). Returns 1 if `min <= x < max`, else 0.

### Cryptographic Operations

| Name | Hex | Decimal | Description |
|------|-----|---------|-------------|
| `OP_SHA256` | `0xa8` | 168 | SHA-256 hash |
| `OP_CHECKMULTISIGECDSA` | `0xa9` | 169 | ECDSA multisig verification |
| `OP_BLAKE2B` | `0xaa` | 170 | BLAKE2b-256 hash |
| `OP_CHECKSIGECDSA` | `0xab` | 171 | ECDSA single-signature verification |
| `OP_CHECKSIG` | `0xac` | 172 | Schnorr single-signature verification |
| `OP_CHECKSIGVERIFY` | `0xad` | 173 | `OP_CHECKSIG` + `OP_VERIFY` |
| `OP_CHECKMULTISIG` | `0xae` | 174 | Schnorr multisig verification |
| `OP_CHECKMULTISIGVERIFY` | `0xaf` | 175 | `OP_CHECKMULTISIG` + `OP_VERIFY` |

#### OP_CHECKSIG (Schnorr)

Pops `signature` and `pubkey` from the stack. The signature is 65 bytes total: 64-byte Schnorr signature + 1-byte hash type. Validation:

1. Extract `hashType` (last byte) and `sigBytes` (first 64 bytes).
2. Verify `hashType` is a standard sighash type.
3. Verify signature length is exactly 64 bytes (`ErrSigLength` otherwise).
4. Verify public key encoding is exactly 32 bytes (`ErrPubKeyFormat` otherwise).
5. Compute the Schnorr signature hash via `CalculateSignatureHashSchnorr`.
6. If `sigCache` is available, check cache before performing `SchnorrVerify`.
7. If verification fails and `sigBytes` is not empty, return `ErrNullFail`.

#### OP_CHECKSIGECDSA

Same flow as `OP_CHECKSIG` but uses ECDSA keys and signatures:
- Public key must be 33 bytes (compressed ECDSA).
- Signature is 64 bytes.
- Uses `CalculateSignatureHashECDSA` and `ECDSAVerify`.

#### OP_CHECKMULTISIG (Schnorr)

Pops `numpubkeys`, then `numpubkeys` public keys, then `numsigs`, then `numsigs` signatures. Validation:

1. `numpubkeys` must be in `[0, MaxPubKeysPerMultiSig]` (20).
2. `numsigs` must be in `[0, numpubkeys]`.
3. Iterates through public keys, for each tries to match the next unverified signature.
4. A signature may be empty (skip to next pubkey).
5. Signatures are parsed lazily and cached to avoid repeated deserialization.
6. If all signatures don't match and any signature is non-empty → `ErrNullFail`.

#### OP_CHECKMULTISIGECDSA

Same as `OP_CHECKMULTISIG` but with ECDSA keys (33-byte) and signatures.

#### Hashing Opcodes

- `OP_SHA256`: Pop top item, push `SHA-256(data)` — 32 bytes.
- `OP_BLAKE2B`: Pop top item, push `BLAKE2b-256(data)` — 32 bytes.

### Disabled and Reserved Opcodes

**Disabled opcodes** cause `ErrDisabledOpcode` whenever the program counter passes over them, even in non-executing branches:

`OP_CAT`, `OP_SUBSTR`, `OP_LEFT`, `OP_RIGHT`, `OP_INVERT`, `OP_AND`, `OP_OR`, `OP_XOR`, `OP_2MUL`, `OP_2DIV`, `OP_MUL`, `OP_DIV`, `OP_MOD`, `OP_LSHIFT`, `OP_RSHIFT`

**Always-illegal opcodes** fail on the program counter (not on execution):

`OP_VERIF`, `OP_VERNOTIF`

**Reserved opcodes** (`OP_RESERVED`, `OP_VER`, `OP_RESERVED1`, `OP_RESERVED2`) fail with `ErrReservedOpcode` only if actually executed (not if skipped by conditionals).

### Undefined / Unknown Opcodes

Opcode values `0xa6`–`0xa7` and `0xb2`–`0xff` (excluding the named ones above) are defined as `OP_UNKNOWN166` through `OP_UNKNOWN252`, `OP_SMALLINTEGER` (`0xfa`), `OP_PUBKEYS` (`0xfb`), `OP_PUBKEYHASH` (`0xfd`), `OP_PUBKEY` (`0xfe`), and `OP_INVALIDOPCODE` (`0xff`). All map to `opcodeInvalid` and fail with `ErrReservedOpcode` if executed.

### Execution Limits

| Constant | Value | Description |
|----------|-------|-------------|
| `MaxScriptSize` | 10,000 | Maximum raw script byte length |
| `MaxStackSize` | 244 | Maximum combined data + alt stack depth |
| `MaxOpsPerScript` | 201 | Maximum non-push opcodes per script |
| `MaxPubKeysPerMultiSig` | 20 | Maximum public keys in multisig |
| `MaxScriptElementSize` | 520 | Maximum bytes in a single stack element |

---

## Standard Scripts

Standard scripts are recognized templates that correspond to specific address types. The `ScriptClass` enumeration defines the categories:

| Class | Name String | Description |
|-------|-------------|-------------|
| `NonStandardTy` | `"nonstandard"` | Not a recognized standard form |
| `PubKeyTy` | `"pubkey"` | Pay-to-Public-Key (Schnorr) |
| `PubKeyECDSATy` | `"pubkeyecdsa"` | Pay-to-Public-Key (ECDSA) |
| `PubKeyHashTy` | `"pubkeyhash"` | Pay-to-Public-Key-Hash (Schnorr) |
| `PubKeyHashECDSATy` | `"pubkeyhashecdsa"` | Pay-to-Public-Key-Hash (ECDSA) |
| `ScriptHashTy` | `"scripthash"` | Pay-to-Script-Hash |

All script public key versions are `0`.

### Pay-to-Public-Key (P2PK) — Schnorr

**Template:**

```
<32-byte public key> OP_CHECKSIG
```

**Spending:** Provide a Schnorr signature.

### Pay-to-Public-Key (P2PK) — ECDSA

**Template:**

```
<33-byte public key> OP_CHECKSIGECDSA
```

**Spending:** Provide an ECDSA signature.

### Pay-to-Public-Key-Hash (P2PKH) — Schnorr

**Template:**

```
OP_DUP OP_BLAKE2B <32-byte pubkey hash> OP_EQUALVERIFY OP_CHECKSIG
```

**Spending:** Provide `<signature> <pubkey>`. The script duplicates the pubkey, hashes it with BLAKE2b, verifies it matches the stored hash, then checks the signature.

### Pay-to-Public-Key-Hash (P2PKH) — ECDSA

**Template:**

```
OP_DUP OP_BLAKE2B <32-byte pubkey hash> OP_EQUALVERIFY OP_CHECKSIGECDSA
```

**Spending:** Provide `<ECDSA signature> <33-byte pubkey>`.

### Pay-to-Script-Hash (P2SH)

**Template:**

```
OP_BLAKE2B <32-byte script hash> OP_EQUAL
```

**Spending:** Provide the redeem script as the last data push in the signature script, preceded by data that satisfies the redeem script. The engine validates the P2SH template, then executes the redeem script against the remaining stack.

### OP_RETURN (Data Carrier)

Scripts beginning with `OP_RETURN` are provably unspendable. Any output with an `OP_RETURN` script is prunable from the UTXO set.

### Script Recognition Functions

```go
func GetScriptClass(script []byte) ScriptClass
func IsPayToScriptHash(script *externalapi.ScriptPublicKey) bool
func ExtractScriptPubKeyAddress(scriptPubKey *externalapi.ScriptPublicKey, dagParams *dagconfig.Params) (ScriptClass, util.Address, error)
func IsUnspendable(scriptPubKey []byte) bool
func PushedData(script []byte) ([][]byte, error)
func CalcScriptInfo(sigScript, scriptPubKey []byte, isP2SH bool) (*ScriptInfo, error)
```

`ScriptInfo` provides metadata about a script pair:

```go
type ScriptInfo struct {
    ScriptPubKeyClass ScriptClass
    NumInputs         int       // number of inputs from sigScript
    ExpectedInputs    int       // expected number (may be -1 for unknown)
    SigOps            int       // signature operation count
}
```

### Pay-to-Address Script Generation

```go
func PayToAddrScript(addr util.Address) (*externalapi.ScriptPublicKey, error)
```

Generates the appropriate `ScriptPublicKey` for a given address type:
- `AddressPublicKey` → P2PK Schnorr script
- `AddressPublicKeyECDSA` → P2PK ECDSA script
- `AddressPublicKeyHash` → P2PKH Schnorr script
- `AddressPublicKeyHashECDSA` → P2PKH ECDSA script
- `AddressScriptHash` → P2SH script

---

## Script Builder

The `ScriptBuilder` ([scriptbuilder.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/scriptbuilder.go#L51-L54)) provides a fluent, chainable API for constructing scripts with automatic canonical encoding.

### API Reference

```go
func NewScriptBuilder() *ScriptBuilder
```

Creates a new builder with a pre-allocated 500-byte backing array.

```go
func (b *ScriptBuilder) AddOp(opcode byte) *ScriptBuilder
```

Appends a single opcode byte. Fails (sets `b.err`) if the resulting script would exceed `MaxScriptSize`.

```go
func (b *ScriptBuilder) AddOps(opcodes []byte) *ScriptBuilder
```

Appends multiple raw opcode bytes.

```go
func (b *ScriptBuilder) AddData(data []byte) *ScriptBuilder
```

Appends data using the **canonically minimal** encoding:
- `nil` or `[]byte{0}` → `OP_0`
- `[0x01]` through `[0x10]` → `OP_1` through `OP_16`
- `[0x81]` → `OP_1NEGATE`
- 1–75 bytes → `OP_DATA_N` + data
- 76–255 bytes → `OP_PUSHDATA1` + length + data
- 256–65535 bytes → `OP_PUSHDATA2` + length (LE) + data
- 65536+ bytes → `OP_PUSHDATA4` + length (LE) + data

Fails if data exceeds `MaxScriptElementSize` or the resulting script exceeds `MaxScriptSize`.

```go
func (b *ScriptBuilder) AddInt64(val int64) *ScriptBuilder
```

Pushes an integer using canonical encoding. Values `-1`, `0`, `1`–`16` use the dedicated opcodes; other values use the minimal `scriptNum` encoding.

```go
func (b *ScriptBuilder) AddLockTimeNumber(lockTime uint64) *ScriptBuilder
```

Pushes a locktime value as an 8-byte little-endian uint64, with trailing zero bytes stripped for minimal encoding.

```go
func (b *ScriptBuilder) AddSequenceNumber(sequence uint64) *ScriptBuilder
```

Same as `AddLockTimeNumber` but for sequence numbers.

```go
func (b *ScriptBuilder) AddFullData(data []byte) *ScriptBuilder
```

**Testing only.** Adds data without enforcing `MaxScriptElementSize` or `MaxScriptSize` limits.

```go
func (b *ScriptBuilder) Reset() *ScriptBuilder
```

Clears the script content and error state.

```go
func (b *ScriptBuilder) Script() ([]byte, error)
```

Returns the built script bytes and any accumulated error.

---

## Signing

The signing subsystem ([sign.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/sign.go)) provides functions to create signatures and assemble signature scripts for spending transaction outputs.

### Sighash Types

Sighash types are defined in the `consensushashing` package. Only standard sighash types pass validation in `OP_CHECKSIG` and `OP_CHECKSIGECDSA`. The hash type is the **last byte** of the full signature byte sequence.

### Raw Signature Generation

```go
func RawTxInSignature(tx *externalapi.DomainTransaction, idx int,
    hashType consensushashing.SigHashType, key *secp256k1.SchnorrKeyPair,
    sighashReusedValues *consensushashing.SighashReusedValues) ([]byte, error)
```

1. Computes the Schnorr signature hash for transaction input `idx` using `CalculateSignatureHashSchnorr`.
2. Signs the hash with the provided Schnorr key pair.
3. Returns `serializedSignature || byte(hashType)` (65 bytes).

```go
func RawTxInSignatureECDSA(tx *externalapi.DomainTransaction, idx int,
    hashType consensushashing.SigHashType, key *secp256k1.ECDSAPrivateKey,
    sighashReusedValues *consensushashing.SighashReusedValues) ([]byte, error)
```

Same flow using ECDSA signing with `CalculateSignatureHashECDSA`.

### Signature Script Generation

```go
func SignatureScript(tx *externalapi.DomainTransaction, idx int,
    hashType consensushashing.SigHashType, privKey *secp256k1.SchnorrKeyPair,
    sighashReusedValues *consensushashing.SighashReusedValues) ([]byte, error)
```

Returns `<signature>` as a script. Used for P2PK Schnorr inputs.

```go
func SignatureScriptPubKeyHash(tx *externalapi.DomainTransaction, idx int,
    hashType consensushashing.SigHashType, privKey *secp256k1.SchnorrKeyPair,
    sighashReusedValues *consensushashing.SighashReusedValues) ([]byte, error)
```

Returns `<signature> <pubkey>` as a script. Used for P2PKH Schnorr inputs.

```go
func SignatureScriptECDSA(tx *externalapi.DomainTransaction, idx int,
    hashType consensushashing.SigHashType, privKey *secp256k1.ECDSAPrivateKey,
    sighashReusedValues *consensushashing.SighashReusedValues) ([]byte, error)
```

ECDSA variant of `SignatureScript`.

### Transaction Output Signing

```go
func SignTxOutput(dagParams *dagconfig.Params, tx *externalapi.DomainTransaction, idx int,
    scriptPublicKey *externalapi.ScriptPublicKey, hashType consensushashing.SigHashType,
    sighashReusedValues *consensushashing.SighashReusedValues,
    kdb KeyDB, sdb ScriptDB, previousScript *externalapi.ScriptPublicKey) ([]byte, error)
```

The high-level signing function:

1. Extracts the script class and address from `scriptPublicKey`.
2. Based on class:
   - `PubKeyTy` / `PubKeyHashTy` → looks up the Schnorr private key via `KeyDB`, generates the appropriate signature script.
   - `ScriptHashTy` → retrieves the redeem script via `ScriptDB`, then recurses to sign the redeem script, finally appending the redeem script as the last push.
3. Merges with `previousScript` if provided.

### Key and Script Database Interfaces

```go
type KeyDB interface {
    GetKey(util.Address) (*secp256k1.SchnorrKeyPair, error)
}

type ScriptDB interface {
    GetScript(util.Address) ([]byte, error)
}
```

Convenience closure adapters are provided: `KeyClosure` and `ScriptClosure`.

---

## Signature Cache

### SigCache (Schnorr)

[`SigCache`](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/sigcache.go#L35-L39) caches successful Schnorr signature verifications using a randomized eviction policy.

```go
func NewSigCache(maxEntries uint) *SigCache
```

**Key:** `secp256k1.Hash` (the signature hash).  
**Value:** `sigCacheEntry` containing `*SchnorrSignature` and `*SchnorrPublicKey`.  
**Thread safety:** `sync.RWMutex` — multiple readers, exclusive writers.

```go
func (s *SigCache) Exists(sigHash secp256k1.Hash, sig *secp256k1.SchnorrSignature,
    pubKey *secp256k1.SchnorrPublicKey) bool
```

Checks both hash match AND signature/public key equality to prevent false positives from hash collisions.

```go
func (s *SigCache) Add(sigHash secp256k1.Hash, sig *secp256k1.SchnorrSignature,
    pubKey *secp256k1.SchnorrPublicKey)
```

Evicts a random entry (via Go map iteration order) when the cache exceeds `maxEntries`.

### SigCacheECDSA

[`SigCacheECDSA`](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/sigcache_ecdsa.go#L35-L39) is the structurally identical ECDSA counterpart using `ECDSASignature` / `ECDSAPublicKey` types.

Both caches serve two purposes:

1. **DoS protection:** Prevents attackers from causing worst-case signature verification behavior with invalid transactions.
2. **Performance optimization:** Avoids re-verifying signatures already seen in the mempool when validating blocks.

---

## Script Number

The [`scriptNum`](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/scriptnum.go#L46) type (`int64`) handles the consensus-critical numeric encoding for script arithmetic.

### Encoding

Numbers are serialized as **little-endian** bytes with a **sign bit** in the most significant byte's MSB:

| Value | Encoding |
|-------|----------|
| `0` | `[]` (empty) |
| `1` | `[0x01]` |
| `-1` | `[0x81]` |
| `127` | `[0x7f]` |
| `-127` | `[0xff]` |
| `128` | `[0x80, 0x00]` |
| `-128` | `[0x80, 0x80]` |
| `256` | `[0x00, 0x01]` |
| `-256` | `[0x00, 0x81]` |

### Minimal Encoding

Numbers must use the **minimally encoded** form. A number fails validation if:
- The most significant byte (excluding sign bit) is zero AND the second-most-significant byte's sign bit is not set (or there is only one byte).
- Negative zero (`[0x80]`) is rejected.

### Constraints

- Default maximum length: 4 bytes (range `[-2^31 + 1, 2^31 - 1]`).
- Absolute maximum: 8 bytes (must fit in `int64`).
- Arithmetic results may overflow `int64` and can still be used as booleans but fail if reinterpreted as integers.
- `Int32()` method clamps out-of-range values to `[minInt32, maxInt32]` instead of truncating.

---

## Error Handling

All errors returned by the package are of type [`Error`](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/txscript/error.go#L284-L287), which contains:

```go
type Error struct {
    ErrorCode   ErrorCode
    Description string
}
```

### Error Codes

#### API Usage Errors

| Code | Description |
|------|-------------|
| `ErrInternal` | Internal consistency failure in the engine |
| `ErrInvalidFlags` | Invalid combination of `ScriptFlags` |
| `ErrInvalidIndex` | Out-of-bounds index (script index, tx input index) |
| `ErrUnsupportedAddress` | Address type not supported for script generation |
| `ErrNotMultisigScript` | Script is not a multisig script |
| `ErrTooManyRequiredSigs` | Required sigs > provided public keys |

#### Execution State Errors

| Code | Description |
|------|-------------|
| `ErrEarlyReturn` | `OP_RETURN` was executed |
| `ErrEmptyStack` | Final stack is empty |
| `ErrEvalFalse` | Final stack top evaluates to false |
| `ErrScriptUnfinished` | `CheckErrorCondition` called before execution complete |
| `ErrInvalidProgramCounter` | Program counter past end of scripts |

#### Limit Exceeded Errors

| Code | Description |
|------|-------------|
| `ErrScriptTooBig` | Script length > `MaxScriptSize` (10,000) |
| `ErrElementTooBig` | Stack element > `MaxScriptElementSize` (520) |
| `ErrTooManyOperations` | Non-push opcodes > `MaxOpsPerScript` (201) |
| `ErrStackOverflow` | Combined stack depth > `MaxStackSize` (244) |
| `ErrInvalidPubKeyCount` | Invalid number of public keys for multisig |
| `ErrInvalidSignatureCount` | Invalid number of signatures for multisig |
| `ErrNumberTooBig` | Numeric value exceeds allowed byte length |

#### Verification Errors

| Code | Description |
|------|-------------|
| `ErrVerify` | `OP_VERIFY` failed |
| `ErrEqualVerify` | `OP_EQUALVERIFY` failed |
| `ErrNumEqualVerify` | `OP_NUMEQUALVERIFY` failed |
| `ErrCheckSigVerify` | `OP_CHECKSIGVERIFY` failed |
| `ErrCheckMultiSigVerify` | `OP_CHECKMULTISIGVERIFY` failed |

#### Opcode Usage Errors

| Code | Description |
|------|-------------|
| `ErrDisabledOpcode` | Disabled opcode encountered |
| `ErrReservedOpcode` | Reserved opcode encountered |
| `ErrMalformedPush` | Data push exceeds remaining script bytes |
| `ErrInvalidStackOperation` | Stack operation with invalid index |
| `ErrUnbalancedConditional` | `OP_ELSE`/`OP_ENDIF` without matching `OP_IF`/`OP_NOTIF` |

#### Malleability Errors

| Code | Description |
|------|-------------|
| `ErrMinimalData` | Non-minimal data push encoding |
| `ErrInvalidSigHashType` | Invalid signature hash type byte |
| `ErrSigLength` | Schnorr/ECDSA signature of incorrect length |
| `ErrSigHighS` | ECDSA signature with high-S value |
| `ErrNotPushOnly` | Signature script contains non-push opcodes |
| `ErrPubKeyFormat` | Invalid public key encoding |
| `ErrCleanStack` | Stack has more than one item after execution |
| `ErrNullFail` | Non-empty signature on failed verification |

#### Soft Fork Errors

| Code | Description |
|------|-------------|
| `ErrNegativeLockTime` | Negative locktime in script |
| `ErrUnsatisfiedLockTime` | Locktime/sequence constraint not satisfied |
| `ErrMinimalIf` | `OP_IF`/`OP_NOTIF` argument not `[]` or `[0x01]` |

### Error Inspection

```go
func IsErrorCode(err error, c ErrorCode) bool
```

Uses `errors.As` to check if an error chain contains a `txscript.Error` with the given code.

---

## Script Parsing

```go
func ParseScript(script []byte) ([]parsedOpcode, error)
```

Parses raw script bytes into a slice of `parsedOpcode` structs. Each `parsedOpcode` contains:
- `opcode *opcode` — pointer into the static `opcodeArray` lookup table.
- `data []byte` — associated data for push opcodes.

The parser handles:
- **Single-byte opcodes** (no data).
- **OP_DATA_N** opcodes (fixed-length data, `N` bytes follow).
- **OP_PUSHDATA1/2/4** opcodes (variable-length data with little-endian length prefix).

`ErrMalformedPush` is returned if a push opcode would read beyond the script boundary.

### Disassembly

```go
func DisasmString(version uint16, buf []byte) (string, error)
```

Returns a one-line disassembly string. For known versions, numeric opcodes are rendered as their values (`0`, `1`, ..., `16`, `-1`) rather than names. Data pushes are rendered as hex. Parse errors append `[error]`.

### Signature Operation Counting

```go
func GetSigOpCount(script []byte) int
func GetPreciseSigOpCount(scriptSig []byte, scriptPubKey *externalapi.ScriptPublicKey) int
```

`GetSigOpCount` uses a pessimistic count: `OP_CHECKSIG`/`OP_CHECKSIGVERIFY`/`OP_CHECKSIGECDSA` = 1, `OP_CHECKMULTISIG`/`OP_CHECKMULTISIGVERIFY`/`OP_CHECKMULTISIGECDSA` = 20 (max).

`GetPreciseSigOpCount` inspects the preceding opcode for multisig to determine the actual number of public keys. For P2SH, it parses the redeem script from the signature script.

---

## Usage Examples

### Building a P2PKH Script (Schnorr)

```go
import "github.com/Eiyaro/Eiyaro/domain/consensus/utils/txscript"

pubKeyHash := util.HashBlake2b(publicKeyBytes) // 32 bytes
script, err := txscript.NewScriptBuilder().
    AddOp(txscript.OpDup).
    AddOp(txscript.OpBlake2b).
    AddData(pubKeyHash).
    AddOp(txscript.OpEqualVerify).
    AddOp(txscript.OpCheckSig).
    Script()
if err != nil {
    // handle error
}
```

### Building a P2PKH Script via PayToAddrScript

```go
addr, err := util.NewAddressPublicKeyHash(pubKeyBytes, dagParams.Prefix)
if err != nil {
    // handle error
}
scriptPubKey, err := txscript.PayToAddrScript(addr)
```

### Building a Multisig Script (2-of-3)

```go
script, err := txscript.NewScriptBuilder().
    AddOp(txscript.Op2).
    AddData(pubKey1).
    AddData(pubKey2).
    AddData(pubKey3).
    AddOp(txscript.Op3).
    AddOp(txscript.OpCheckMultiSig).
    Script()
```

### Building a P2SH Script

```go
redeemScript, err := txscript.NewScriptBuilder().
    AddOp(txscript.Op2).
    AddData(pubKey1).
    AddData(pubKey2).
    AddData(pubKey3).
    AddOp(txscript.Op3).
    AddOp(txscript.OpCheckMultiSig).
    Script()

p2shScript, err := txscript.PayToScriptHashScript(redeemScript)
```

### Validating a Transaction Script

```go
engine, err := txscript.NewEngine(scriptPubKey, tx, inputIndex,
    txscript.ScriptNoFlags, sigCache, sigCacheECDSA, sighashReusedValues)
if err != nil {
    // handle initialization error
}
err = engine.Execute()
if err != nil {
    // handle script validation failure
}
// script is valid
```

### Signing a Transaction Input

```go
sigScript, err := txscript.SignTxOutput(dagParams, tx, inputIndex,
    scriptPubKey, consensushashing.SigHashAll, sighashReusedValues,
    keyDB, scriptDB, previousScript)
if err != nil {
    // handle signing error
}
tx.Inputs[inputIndex].SignatureScript = sigScript
```

### Disassembling a Script

```go
disasm, err := txscript.DisasmString(scriptPubKey.Version, scriptPubKey.Script)
if err != nil {
    // handle error
}
fmt.Println(disasm)
// Output example: "OP_DUP OP_BLAKE2B 0x<hexhash> OP_EQUALVERIFY OP_CHECKSIG"
```

---

## FAQ

### What are the most commonly used opcodes?

The most frequently used opcodes are `OP_CHECKSIG` (Schnorr signature verification), `OP_CHECKSIGECDSA` (ECDSA signature verification), `OP_DUP`, `OP_BLAKE2B` (hashing), `OP_EQUALVERIFY`, `OP_CHECKLOCKTIMEVERIFY`, and `OP_RETURN`. For multisig scenarios, `OP_CHECKMULTISIG` and `OP_CHECKMULTISIGVERIFY` are used. Most standard transactions use P2PKH templates: `OP_DUP OP_BLAKE2B <hash> OP_EQUALVERIFY OP_CHECKSIG`.

### How do I create a custom script?

Use the `ScriptBuilder` fluent API. Start with `txscript.NewScriptBuilder()`, chain `AddOp()` for opcodes and `AddData()` for data pushes, then call `Script()` to get the final byte sequence. The builder automatically uses canonical minimal encoding. For example, a 2-of-3 multisig: `NewScriptBuilder().AddOp(Op2).AddData(pk1).AddData(pk2).AddData(pk3).AddOp(Op3).AddOp(OpCheckMultiSig).Script()`. Custom scripts can be wrapped in P2SH for address-based spending.

### What are the transaction fee limits?

Transaction fees are market-driven and not hard-coded into the script engine. The fee is implicitly the difference between total input value and total output value. Use the `GetFeeEstimate` RPC to query current feerate recommendations in sompi per gram. The block mass limit (500,000 grams) and per-sigop mass (1,000 grams per sigop) constrain how many operations a transaction can include, which indirectly bounds fees.

### Can I use Bitcoin scripts with Eiyaro?

Eiyaro's script engine is derived from Bitcoin's but is not directly compatible. Key differences include: BLAKE2b replaces SHA-256/RIPEMD-160 for hashing, Schnorr signatures use 32-byte public keys (vs. 33-byte ECDSA), several opcodes (`OP_CAT`, `OP_MUL`, `OP_DIV`, bitwise ops) are disabled, and Eiyaro supports both Schnorr and ECDSA signature verification natively. You can port Bitcoin script logic but must adapt the hashing and signature operations.

### Why does my script fail with ErrCleanStack?

After script execution, the data stack must contain exactly one item that evaluates to true. An `ErrCleanStack` error means the stack has more than one item remaining — typically because your unlocking script pushed extra data that was not consumed by the locking script. Ensure your signature script only pushes the exact data needed to satisfy the locking script. For P2PKH, that means exactly `<signature> <pubkey>` — no extra pushes.

### What's the difference between OP_CHECKSIG and OP_CHECKSIGECDSA?

`OP_CHECKSIG` uses Schnorr signatures with 32-byte public keys and 64-byte signatures (plus 1 hash type byte = 65 bytes total). `OP_CHECKSIGECDSA` uses ECDSA signatures with 33-byte compressed public keys and 64-byte signatures. Both are supported in parallel, and the address type determines which is used: `AddressPublicKey` → Schnorr, `AddressPublicKeyECDSA` → ECDSA. Schnorr is generally preferred for its smaller size and native multisig efficiency.