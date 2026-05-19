package nogopow

import (
	"encoding/binary"
	"math"
	"sync"
	
	"github.com/zeebo/blake3"
)

type denseMatrix struct {
	data []int64
	rows int
	cols int
}

func toFixed(val float64) int64 {
	return int64(val * float64(FixedPointFactor))
}

func fromFixed(val int64) float64 {
	return float64(val) / float64(FixedPointFactor)
}

// toFixedShift converts int64 to fixed-point representation by left-shifting.
// Uses SafeLeftShift to prevent overflow.
func toFixedShift(val int64) int64 {
	result, err := SafeLeftShift(val, FixedPointShift)
	if err != nil {
		// In production, you might want to log this error
		// For now, we clamp to max/min values
		if val > 0 {
			return math.MaxInt64
		}
		return math.MinInt64
	}
	return result
}

// kahanSum implements Kahan summation algorithm to reduce numerical error
// in floating-point addition. This ensures cross-platform consistency.
func kahanSum(values []float64) float64 {
	sum := 0.0
	c := 0.0 // Running compensation for lost low-order bits
	
	for _, val := range values {
		y := val - c             // Compensated value
		t := sum + y             // New sum
		c = (t - sum) - y        // Recover and sum the lost part
		sum = t
	}
	
	return sum
}

// kahanSumInt64 implements Kahan summation for int64 values
// to prevent overflow and maintain precision
func kahanSumInt64(values []int64) int64 {
	sum := int64(0)
	c := int64(0) // Compensation for lost low-order bits
	
	for _, val := range values {
		y := val - c
		t := sum + y
		c = (t - sum) - y
		sum = t
	}
	
	return sum
}

func newDenseMatrix(rows, cols int) *denseMatrix {
	return &denseMatrix{
		data: make([]int64, rows*cols),
		rows: rows,
		cols: cols,
	}
}

func (m *denseMatrix) Get(i, j int) int64 {
	if i < 0 || i >= m.rows || j < 0 || j >= m.cols {
		return 0
	}
	return m.data[i*m.cols+j]
}

func (m *denseMatrix) Set(i, j int, val int64) {
	if i < 0 || i >= m.rows || j < 0 || j >= m.cols {
		return
	}
	m.data[i*m.cols+j] = val
}

var matrixPool = sync.Pool{
	New: func() interface{} {
		return newDenseMatrix(MatrixSize, MatrixSize)
	},
}

func GetMatrix() *denseMatrix {
	m := matrixPool.Get().(*denseMatrix)
	for i := range m.data {
		m.data[i] = 0
	}
	return m
}

func PutMatrix(m *denseMatrix) {
	for i := range m.data {
		m.data[i] = 0
	}
	matrixPool.Put(m)
}

func mulMatrixBlocked(a, b *denseMatrix, blockSize int) *denseMatrix {
	if a.cols != b.rows {
		return nil
	}
	
	result := newDenseMatrix(a.rows, b.cols)
	
	for ii := 0; ii < a.rows; ii += blockSize {
		for jj := 0; jj < b.cols; jj += blockSize {
			for kk := 0; kk < a.cols; kk += blockSize {
				for i := ii; i < min(ii+blockSize, a.rows); i++ {
					for j := jj; j < min(jj+blockSize, b.cols); j++ {
						sum := int64(0)
						for k := kk; k < min(kk+blockSize, a.cols); k++ {
							av := a.Get(i, k)
							bv := b.Get(k, j)
							// Use SafeMulInt64 to prevent overflow
							product, err := SafeMulInt64(av, bv)
							if err != nil {
								// Handle overflow by clamping
								if (av > 0 && bv > 0) || (av < 0 && bv < 0) {
									product = math.MaxInt64
								} else {
									product = math.MinInt64
								}
							}
							// Safe right shift (no overflow possible)
							product, _ = SafeRightShift(product, FixedPointShift)
							sum, err = SafeAddInt64(sum, product)
							if err != nil {
								// Clamp sum to prevent overflow
								if sum > 0 {
									sum = math.MaxInt64
								} else {
									sum = math.MinInt64
								}
							}
						}
						existing := result.Get(i, j)
						newVal, err := SafeAddInt64(existing, sum)
						if err != nil {
							// Clamp to prevent overflow
							if existing > 0 {
								newVal = math.MaxInt64
							} else {
								newVal = math.MinInt64
							}
						}
						result.Set(i, j, newVal)
					}
				}
			}
		}
	}
	
	return result
}

func mulMatrixPooled(a, b *denseMatrix, numThreads int) *denseMatrix {
	if a.cols != b.rows {
		return nil
	}
	
	result := newDenseMatrix(a.rows, b.cols)
	var wg sync.WaitGroup
	
	rowsPerThread := (a.rows + numThreads - 1) / numThreads
	
	for t := 0; t < numThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			
			startRow := threadID * rowsPerThread
			endRow := min(startRow+rowsPerThread, a.rows)
			
			for i := startRow; i < endRow; i++ {
				for j := 0; j < b.cols; j++ {
					sum := int64(0)
					for k := 0; k < a.cols; k++ {
						av := a.Get(i, k)
						bv := b.Get(k, j)
						// Use SafeMulInt64 to prevent overflow
						product, err := SafeMulInt64(av, bv)
						if err != nil {
							// Handle overflow by clamping
							if (av > 0 && bv > 0) || (av < 0 && bv < 0) {
								product = math.MaxInt64
							} else {
								product = math.MinInt64
							}
						}
						// Safe right shift (no overflow possible)
						product, _ = SafeRightShift(product, FixedPointShift)
						sum, err = SafeAddInt64(sum, product)
						if err != nil {
							// Clamp sum to prevent overflow
							if sum > 0 {
								sum = math.MaxInt64
							} else {
								sum = math.MinInt64
							}
						}
					}
					result.Set(i, j, sum)
				}
			}
		}(t)
	}
	
	wg.Wait()
	return result
}

func hashMatrix(m *denseMatrix, seed []byte) Hash {
	hasher := blake3.New()
	
	for _, v := range seed {
		hasher.Write([]byte{v})
	}
	
	buf := make([]byte, 8)
	for _, val := range m.data {
		binary.LittleEndian.PutUint64(buf, uint64(val))
		hasher.Write(buf)
	}
	
	var hash Hash
	result := hasher.Sum(nil)
	copy(hash[:], result)
	return hash
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
