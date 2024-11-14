package _007_reverse_integer

import "math"

func reverse(x int) int {
	tmp := 0
	for x != 0 {
		tmp = tmp*10 + x%10
		x = x / 10
	}
	if tmp > math.MaxInt32 || tmp < math.MinInt32 {
		return 0
	}
	return tmp
}
