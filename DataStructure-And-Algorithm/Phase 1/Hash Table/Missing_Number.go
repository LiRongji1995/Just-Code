package Hash_Table

func missingNumber(nums []int) int {
	var sum = 0
	for _, v := range nums {
		sum += v
	}

	n := len(nums)

	maxSum := n * (n + 1) / 2

	return maxSum - sum
}
