package Array

import (
	"math"
	"sort"
)

// 三重循环，十分暴力，简洁明了，时间拉满
//
//	func threeSumClosest(nums []int, target int) int {
//		res, difference := 0, math.MaxInt16
//		for i := 0; i < len(nums); i++ {
//			for j := i + 1; j < len(nums); j++ {
//				for k := j + 1; k < len(nums); k++ {
//					if abs(target-nums[i]-nums[k]-nums[j]) < difference {
//						difference = abs(target - nums[i] - nums[k] - nums[j])
//						res = nums[i] + nums[k] + nums[j]
//					}
//				}
//			}
//		}
//		return res
//	}
func abs(a int) int {
	if a < 0 {
		return -a
	} else {
		return a
	}
}
func threeSumClosest(nums []int, target int) int {
	length, result, difference := len(nums), 0, math.MaxInt32
	if length > 2 {
		sort.Ints(nums)
		for i := 0; i < length-2; i++ {
			for j, k := i+1, length-1; j < k; {
				sum := nums[i] + nums[j] + nums[k]
				if abs(target-sum) < difference {
					difference = abs(target - sum)
					result = sum
				}
				if sum == target {
					return result
				} else if sum < target {
					j++
				} else {
					k--
				}
			}
		}
	}
	return result
}
