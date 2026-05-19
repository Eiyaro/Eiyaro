//go:build linux && arm64
// +build linux,arm64

package pow

import (
	"math/big"

	"github.com/Eiyaro/Eiyaro/domain/consensus/model/externalapi"
	"github.com/Eiyaro/Eiyaro/domain/consensus/utils/consensushashing"
	"github.com/Eiyaro/Eiyaro/util/difficulty"
)

// State is an intermediate data structure with pre-computed values to speed up mining.
type State struct {
	nogopowMatA  *nogopowMatrix
	nogopowMatB  *nogopowMatrix
	Timestamp    int64
	Nonce        uint64
	Target       big.Int
	PrevHeader   externalapi.DomainHash
	BlockVersion uint16
}

// NewState creates a new state with pre-computed values to speed up mining.
// Uses NogoPow: 256x256 fixed-point matrix multiplication with Blake3 hashing.
func NewState(header externalapi.MutableBlockHeader) *State {
	target := difficulty.CompactToBig(header.Bits())
	timestamp, nonce := header.TimeInMilliseconds(), header.Nonce()
	header.SetTimeInMilliseconds(0)
	header.SetNonce(0)
	prevHeader := consensushashing.HeaderHash(header)
	header.SetTimeInMilliseconds(timestamp)
	header.SetNonce(nonce)

	matA := generateNogopowMatrix(prevHeader)
	matB := newBlockNumberMatrix(header.DAAScore())
	return &State{
		Target:       *target,
		PrevHeader:   *prevHeader,
		nogopowMatA:  matA,
		nogopowMatB:  matB,
		Timestamp:    timestamp,
		Nonce:        nonce,
		BlockVersion: header.Version(),
	}
}

func (state *State) CalculateProofOfWorkValue() (*big.Int, *externalapi.DomainHash) {
	return state.CalculateProofOfWorkValueNogopow()
}

func (state *State) IncrementNonce() {
	state.Nonce++
}

func (state *State) CheckProofOfWork(block *externalapi.DomainBlock, powSkip bool) bool {
	powNum, _ := state.CalculateProofOfWorkValue()
	if powSkip {
		return powNum.Cmp(&state.Target) <= 0
	}
	powHash, err := externalapi.NewDomainHashFromString(block.PoWHash)
	if err != nil {
		return false
	}
	if !powHash.Equal(new(externalapi.DomainHash)) {
		submittedPowNum := toBig(powHash)
		if submittedPowNum.Cmp(powNum) == 0 {
			return powNum.Cmp(&state.Target) <= 0
		}
	}
	return false
}

func CheckProofOfWorkByBits(header externalapi.MutableBlockHeader, block *externalapi.DomainBlock, powSkip bool) bool {
	return NewState(header).CheckProofOfWork(block, powSkip)
}

func toBig(hash *externalapi.DomainHash) *big.Int {
	buf := hash.ByteSlice()
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}
	return new(big.Int).SetBytes(buf)
}

func BlockLevel(header externalapi.BlockHeader, maxBlockLevel int) int {
	if len(header.DirectParents()) == 0 {
		return maxBlockLevel
	}
	proofOfWorkValue, _ := NewState(header.ToMutable()).CalculateProofOfWorkValue()
	level := max(maxBlockLevel-proofOfWorkValue.BitLen(), 0)
	return level
}

// SetUseHoohashCLibrary is a no-op kept for backward compatibility.
// NogoPow uses pure Go implementation only.
func SetUseHoohashCLibrary(use bool) {}