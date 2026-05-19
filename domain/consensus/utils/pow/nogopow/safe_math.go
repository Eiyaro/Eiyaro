﻿package nogopow

import (
	"fmt"
	"math"
)

// SafeMulInt64 performs safe int64 multiplication with overflow checking.
// Returns error if overflow would occur.
func SafeMulInt64(a, b int64) (int64, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}

	if a > 0 {
		if b > 0 {
			if a > math.MaxInt64/b {
				return 0, fmt.Errorf("multiplication overflow: %d * %d", a, b)
			}
		} else {
			if b < math.MinInt64/a {
				return 0, fmt.Errorf("multiplication overflow: %d * %d", a, b)
			}
		}
	} else {
		if b > 0 {
			if a < math.MinInt64/b {
				return 0, fmt.Errorf("multiplication overflow: %d * %d", a, b)
			}
		} else {
			if a != 0 && b < math.MaxInt64/a {
				return 0, fmt.Errorf("multiplication overflow: %d * %d", a, b)
			}
		}
	}

	return a * b, nil
}

// SafeAddInt64 performs safe int64 addition with overflow checking.
// Returns error if overflow or underflow would occur.
func SafeAddInt64(a, b int64) (int64, error) {
	if b > 0 && a > math.MaxInt64-b {
		return 0, fmt.Errorf("addition overflow: %d + %d", a, b)
	}
	if b < 0 && a < math.MinInt64-b {
		return 0, fmt.Errorf("addition underflow: %d + %d", a, b)
	}
	return a + b, nil
}

// SafeSubInt64 performs safe int64 subtraction with overflow checking.
// Returns error if overflow or underflow would occur.
func SafeSubInt64(a, b int64) (int64, error) {
	if b < 0 && a > math.MaxInt64+b {
		return 0, fmt.Errorf("subtraction overflow: %d - %d", a, b)
	}
	if b > 0 && a < math.MinInt64+b {
		return 0, fmt.Errorf("subtraction underflow: %d - %d", a, b)
	}
	return a - b, nil
}

// SafeDivInt64 performs safe int64 division with zero check.
// Returns error if division by zero would occur.
func SafeDivInt64(a, b int64) (int64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero: %d / 0", a)
	}
	if a == math.MinInt64 && b == -1 {
		return 0, fmt.Errorf("division overflow: %d / %d", a, b)
	}
	return a / b, nil
}

// SafeLeftShift performs safe left shift with overflow checking.
// Returns error if shift would cause overflow.
func SafeLeftShift(val int64, shift uint) (int64, error) {
	if shift >= 64 {
		return 0, fmt.Errorf("shift amount too large: %d", shift)
	}

	if val > 0 && val > (math.MaxInt64>>shift) {
		return 0, fmt.Errorf("left shift overflow: %d << %d", val, shift)
	}
	if val < 0 && val < (math.MinInt64>>shift) {
		return 0, fmt.Errorf("left shift underflow: %d << %d", val, shift)
	}

	return val << shift, nil
}

// SafeRightShift performs safe right shift.
// Note: Right shift never overflows, but we validate shift amount.
func SafeRightShift(val int64, shift uint) (int64, error) {
	if shift >= 64 {
		return 0, fmt.Errorf("shift amount too large: %d", shift)
	}
	return val >> shift, nil
}

// ClampInt64 clamps a value between min and max.
func ClampInt64(val, min, max int64) int64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// AbsInt64 returns absolute value with overflow check.
// MinInt64 cannot be represented as positive int64.
func AbsInt64(val int64) (int64, error) {
	if val == math.MinInt64 {
		return 0, fmt.Errorf("absolute value overflow: %d", val)
	}
	if val < 0 {
		return -val, nil
	}
	return val, nil
}