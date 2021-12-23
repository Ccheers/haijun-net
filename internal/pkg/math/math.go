package math

import "errors"

const (
	bitsize       = 32 << (^uint(0) >> 63) // 判断当前系统是 64 还是 32 位
	maxintHeadBit = 1 << (bitsize - 2)     // 最大的有符号整数位
)

var (
	errCantGrow = errors.New("argument is too large")
)

// IsPowerOfTwo reports whether given integer is a power of two.
// 判断是否是 2 的幂
func IsPowerOfTwo(n int) bool {
	return n&(n-1) == 0
}

// CeilToPowerOfTwo returns the least power of two integer value greater than
// or equal to n.
// 得到一个大于 n 的最小的 2 的幂数
func CeilToPowerOfTwo(n int) (int, error) {
	// 如果 n 已经大于 2 的幂数，直接返回不能增长的错误
	if n&maxintHeadBit != 0 && n > maxintHeadBit {
		return 0, errCantGrow
	}
	if n <= 2 {
		return 2, nil
	}
	n--
	n = fillBits(n)
	n++
	return n, nil
}

// FloorToPowerOfTwo returns the greatest power of two integer value less than
// or equal to n.
// 得到一个小于 n 的最大的 2 的幂数
func FloorToPowerOfTwo(n int) int {
	if n <= 2 {
		return 2
	}
	n = fillBits(n)
	n >>= 1
	n++
	return n
}

// fillBits returns the number n with all bits set to 1.
// 将 n 的所有位置为 1
func fillBits(n int) int {
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n
}
