package nogopow

import (
	"math/big"
	"sync"
)

// Note: Matrix and fixed-point constants are now defined in consts.go
// matSize = MatrixSize (256)
// matNum = MatrixCount (128)
// FixedPointFactor = 1 << FixedPointShift

type Hash [32]byte

type Address [20]byte

type BlockNonce [32]byte

type Header struct {
	ParentHash Hash
	Coinbase   Address
	Root       Hash
	TxHash     Hash
	Number     *big.Int
	GasLimit   uint64
	Time       uint64
	Extra      []byte
	Nonce      BlockNonce
	Difficulty *big.Int
}

type Block struct {
	header       *Header
	transactions []*Transaction
	uncles       []*Header
}

type Transaction struct {
	Hash Hash
	Data []byte
}

type ChainHeaderReader interface {
	HeaderByNumber(number *big.Int) *Header
	CurrentHeader() *Header
}

type ChainReader interface {
	ChainHeaderReader
	GetBlockByNumber(number *big.Int) *Block
}

type StateDB interface {
	GetBalance(addr Address) *big.Int
	GetNonce(addr Address) uint64
}

var (
	headerPool = sync.Pool{
		New: func() interface{} {
			return &Header{
				Number:     new(big.Int),
				Difficulty: new(big.Int),
				Extra:      make([]byte, 0, 32),
			}
		},
	}

	blockPool = sync.Pool{
		New: func() interface{} {
			return &Block{
				transactions: make([]*Transaction, 0, 100),
				uncles:       make([]*Header, 0, 2),
			}
		},
	}
)

func AcquireHeader() *Header {
	h := headerPool.Get().(*Header)
	h.Number.SetUint64(0)
	h.Difficulty.SetUint64(0)
	h.Extra = h.Extra[:0]
	return h
}

func ReleaseHeader(h *Header) {
	headerPool.Put(h)
}

func AcquireBlock() *Block {
	b := blockPool.Get().(*Block)
	b.transactions = b.transactions[:0]
	b.uncles = b.uncles[:0]
	return b
}

func ReleaseBlock(b *Block) {
	blockPool.Put(b)
}
