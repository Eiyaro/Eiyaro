package pow

import (
	"encoding/binary"
	"math/big"
	"sync"

	"github.com/Eiyaro/Eiyaro/domain/consensus/model/externalapi"
	"github.com/Eiyaro/Eiyaro/domain/consensus/utils/hashes"
	"lukechampine.com/blake3"
)

const (
	nogopowMatrixSize      = 256
	nogopowMatrixCount     = 128
	nogopowFixedPointShift = 30
	nogopowFixedPointMask  = (1 << nogopowFixedPointShift) - 1
	nogopowMatrixModulo    = 1000
)

type nogopowMatrix [nogopowMatrixSize][nogopowMatrixSize]int64

var nogopowMatrixPool = sync.Pool{
	New: func() interface{} {
		return &nogopowMatrix{}
	},
}

func getNogopowMatrix() *nogopowMatrix {
	m := nogopowMatrixPool.Get().(*nogopowMatrix)
	for i := range m {
		for j := range m[i] {
			m[i][j] = 0
		}
	}
	return m
}

func putNogopowMatrix(m *nogopowMatrix) {
	nogopowMatrixPool.Put(m)
}

func generateNogopowMatrix(hash *externalapi.DomainHash) *nogopowMatrix {
	gen := newxoShiRo256PlusPlus(hash)
	mat := getNogopowMatrix()

	for i := 0; i < nogopowMatrixSize; i++ {
		for j := 0; j < nogopowMatrixSize; j += 16 {
			val := gen.Uint64()
			mat[i][j] = int16ToFixed(int16(val & 0x0F))
			mat[i][j+1] = int16ToFixed(int16((val >> 4) & 0x0F))
			mat[i][j+2] = int16ToFixed(int16((val >> 8) & 0x0F))
			mat[i][j+3] = int16ToFixed(int16((val >> 12) & 0x0F))
			mat[i][j+4] = int16ToFixed(int16((val >> 16) & 0x0F))
			mat[i][j+5] = int16ToFixed(int16((val >> 20) & 0x0F))
			mat[i][j+6] = int16ToFixed(int16((val >> 24) & 0x0F))
			mat[i][j+7] = int16ToFixed(int16((val >> 28) & 0x0F))
			mat[i][j+8] = int16ToFixed(int16((val >> 32) & 0x0F))
			mat[i][j+9] = int16ToFixed(int16((val >> 36) & 0x0F))
			mat[i][j+10] = int16ToFixed(int16((val >> 40) & 0x0F))
			mat[i][j+11] = int16ToFixed(int16((val >> 44) & 0x0F))
			mat[i][j+12] = int16ToFixed(int16((val >> 48) & 0x0F))
			mat[i][j+13] = int16ToFixed(int16((val >> 52) & 0x0F))
			mat[i][j+14] = int16ToFixed(int16((val >> 56) & 0x0F))
			mat[i][j+15] = int16ToFixed(int16((val >> 60) & 0x0F))
		}
	}
	return mat
}

func int16ToFixed(val int16) int64 {
	return int64(val) << nogopowFixedPointShift
}

func fixedToInt16(val int64) int16 {
	return int16((val >> nogopowFixedPointShift) & 0xF)
}

func nogopowMatMulVector(mat *nogopowMatrix, vector *[nogopowMatrixSize]int64, result *[nogopowMatrixSize]int64) {
	for i := 0; i < nogopowMatrixSize; i++ {
		var sum int64
		for j := 0; j < nogopowMatrixSize; j++ {
			prod := mat[i][j] * vector[j]
			prod >>= nogopowFixedPointShift
			sum += prod
		}
		result[i] = sum
	}
}

func nogopowMatrixMultiply(matA, matB *nogopowMatrix) *nogopowMatrix {
	result := getNogopowMatrix()
	var wg sync.WaitGroup
	rowsPerWorker := (nogopowMatrixSize + 3) / 4

	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			startRow := workerID * rowsPerWorker
			endRow := startRow + rowsPerWorker
			if endRow > nogopowMatrixSize {
				endRow = nogopowMatrixSize
			}
			for i := startRow; i < endRow; i++ {
				for j := 0; j < nogopowMatrixSize; j++ {
					var sum int64
					for k := 0; k < nogopowMatrixSize; k++ {
						prod := matA[i][k] * matB[k][j]
						prod >>= nogopowFixedPointShift
						sum += prod
					}
					result[i][j] = sum
				}
			}
		}(w)
	}
	wg.Wait()
	return result
}

func nogopowHashMatrix(mat *nogopowMatrix, seed []byte) *externalapi.DomainHash {
	h := blake3.New(32, nil)

	for _, v := range seed {
		h.Write([]byte{v})
	}

	buf := make([]byte, 8)
	for i := 0; i < nogopowMatrixSize; i++ {
		for j := 0; j < nogopowMatrixSize; j++ {
			binary.LittleEndian.PutUint64(buf, uint64(mat[i][j]))
			h.Write(buf)
		}
	}

	var res [externalapi.DomainHashSize]byte
	sum := h.Sum(nil)
	copy(res[:], sum[:externalapi.DomainHashSize])
	return externalapi.NewDomainHashFromByteArray(&res)
}

func (state *State) CalculateProofOfWorkValueNogopow() (*big.Int, *externalapi.DomainHash) {
	writer := hashes.Blake3HashWriter()

	writer.InfallibleWrite(state.PrevHeader.ByteSlice())

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(state.Timestamp))
	writer.InfallibleWrite(buf)

	zeroes := [32]byte{}
	writer.InfallibleWrite(zeroes[:])

	binary.LittleEndian.PutUint64(buf, state.Nonce)
	writer.InfallibleWrite(buf)

	powHash := writer.Finalize()

	matA := state.nogopowMatA
	matB := state.nogopowMatB

	matResult := nogopowMatrixMultiply(matA, matB)
	defer putNogopowMatrix(matResult)

	resultHash := nogopowHashMatrix(matResult, powHash.ByteSlice())
	return toBig(resultHash), resultHash
}

func (state *State) nogopowHashPow(hash *externalapi.DomainHash) *externalapi.DomainHash {
	matA := state.nogopowMatA
	matB := state.nogopowMatB

	matResult := nogopowMatrixMultiply(matA, matB)
	defer putNogopowMatrix(matResult)

	return nogopowHashMatrix(matResult, hash.ByteSlice())
}

func newBlockNumberMatrix(blockNumber uint64) *nogopowMatrix {
	mat := getNogopowMatrix()
	for i := 0; i < nogopowMatrixSize; i++ {
		for j := 0; j < nogopowMatrixSize; j++ {
			val := int64(blockNumber) * int64(i+j+1)
			val %= nogopowMatrixModulo
			mat[i][j] = int64ToFixed(val)
		}
	}
	return mat
}

func int64ToFixed(val int64) int64 {
	return val << nogopowFixedPointShift
}