package nogopow

import (
	"encoding/binary"
	"math/big"

	"github.com/zeebo/blake3"
	"golang.org/x/crypto/sha3"
)

func SealHash(header *Header) Hash {
	encoded := rlpEncode(header)

	hasher := blake3.New()
	hasher.Write(encoded)

	var hash Hash
	result := hasher.Sum(nil)
	copy(hash[:], result)
	return hash
}

func rlpEncode(header *Header) []byte {
	buf := make([]byte, 0, 256)

	buf = append(buf, header.ParentHash[:]...)
	buf = append(buf, header.Coinbase[:]...)
	buf = append(buf, header.Root[:]...)
	buf = append(buf, header.TxHash[:]...)

	numBytes := header.Number.Bytes()
	buf = appendU256(buf, numBytes)

	buf = binary.LittleEndian.AppendUint64(buf, header.GasLimit)
	buf = binary.LittleEndian.AppendUint64(buf, header.Time)

	buf = binary.LittleEndian.AppendUint16(buf, uint16(len(header.Extra)))
	buf = append(buf, header.Extra...)

	buf = append(buf, header.Nonce[:]...)

	diffBytes := header.Difficulty.Bytes()
	buf = appendU256(buf, diffBytes)

	return buf
}

func appendU256(buf []byte, bytes []byte) []byte {
	if len(bytes) > 32 {
		bytes = bytes[len(bytes)-32:]
	}

	padded := make([]byte, 32)
	copy(padded[32-len(bytes):], bytes)

	return append(buf, padded...)
}

func BytesToHash(b []byte) Hash {
	var h Hash
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	copy(h[32-len(b):], b)
	return h
}

func BigToHash(b *big.Int) Hash {
	if b == nil {
		return Hash{}
	}
	return BytesToHash(b.Bytes())
}

func difficultyToTarget(difficulty *big.Int) *big.Int {
	if difficulty == nil || difficulty.Sign() <= 0 {
		maxTarget := new(big.Int).Lsh(big.NewInt(1), 256)
		maxTarget.Sub(maxTarget, big.NewInt(1))
		return maxTarget
	}

	maxTarget := new(big.Int).Lsh(big.NewInt(1), 256)
	maxTarget.Sub(maxTarget, big.NewInt(1))

	target := new(big.Int).Div(maxTarget, difficulty)

	return target
}

func targetToDifficulty(target *big.Int) *big.Int {
	if target == nil || target.Sign() <= 0 {
		return big.NewInt(1)
	}

	maxTarget := new(big.Int).Lsh(big.NewInt(1), 256)
	maxTarget.Sub(maxTarget, big.NewInt(1))

	difficulty := new(big.Int).Div(maxTarget, target)

	if difficulty.Sign() <= 0 {
		return big.NewInt(1)
	}

	return difficulty
}

func hashToBig(h Hash) *big.Int {
	return new(big.Int).SetBytes(h[:])
}

func bigToHash(b *big.Int) Hash {
	bytes := b.Bytes()
	var h Hash
	if len(bytes) > 32 {
		bytes = bytes[len(bytes)-32:]
	}
	copy(h[32-len(bytes):], bytes)
	return h
}

func computeBlake3Hash(data []byte) Hash {
	hasher := blake3.New()
	hasher.Write(data)

	var hash Hash
	result := hasher.Sum(nil)
	copy(hash[:], result)
	return hash
}

func computeSHA3Hash(data []byte) Hash {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)

	var hash Hash
	hasher.Sum(hash[:0])
	return hash
}

func verifyHashAgainstTarget(hash Hash, target *big.Int) bool {
	hashVal := hashToBig(hash)
	return hashVal.Cmp(target) <= 0
}
