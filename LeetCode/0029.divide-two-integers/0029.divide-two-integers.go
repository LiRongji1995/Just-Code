package _029_divide_two_integers

import "math"

// 解法一 递归版的二分搜索
func divide(dividend int, divisor int) int {
	sign, res := -1, 0
	//low, high := 0, abs(dividend)
	if dividend == 0 {
		return 0
	}
	if divisor == 1 {
		return dividend
	}
	if dividend == math.MinInt32 && divisor == -1 {
		res = math.MaxInt32
	}
	if dividend > 0 && divisor > 0 || dividend < 0 && divisor < 0 {
		sign = 1
	}

	if dividend > math.MaxInt32 {
		dividend = math.MinInt32
	}
	res = binarySearchQuotient(0, abs(dividend), abs(divisor), abs(dividend))
	if res > math.MaxInt32 {
		return sign * math.MaxInt32
	}
	if res < math.MinInt32 {
		return sign * math.MinInt32
	}
	return sign * res
}

// 解法二 非递归的二分搜索
func divide1(divided int, divisor int) int {
	if divided == math.MinInt32 && divisor == -1 {
		return math.MaxInt32
	}
	result := 0
	sign := -1
	if divided > 0 && divisor > 0 || divided < 0 && divisor < 0 {
		sign = 1
	}
	dvd, dvs := abs(divided), abs(divisor)
	for dvd >= dvs {
		temp := dvs
		m := 1
		for temp<<1 <= dvd {
			temp <<= 1
			m <<= 1
		}
		dvd -= temp
		result += m
	}
	return sign * result
}

func binarySearchQuotient(low, high, val, dividend int) int {
	quotient := low + (high-low)>>1
	if ((quotient+1)*val > dividend && quotient*val <= dividend) ||
		((quotient+1)*val >= dividend && quotient*val < dividend) {
		if (quotient+1)*val == dividend {
			return quotient + 1
		}
		return quotient
	}
	if (quotient+1)*val > dividend && quotient*val > dividend {
		return binarySearchQuotient(low, quotient-1, val, dividend)
	}
	if (quotient+1)*val < dividend && quotient*val < dividend {
		return binarySearchQuotient(quotient+1, high, val, dividend)
	}
	return 0
}

func abs(a int) int {
	if a > 0 {
		return a
	}
	return -a
}
