package Array

func removeElement(nums []int, target int) int {
	if len(nums) == 0 {
		return 0
	}
	j := 0
	for i := 0; i < len(nums); i++ {
		if nums[i] != target {
			if i != j {
				nums[j] = nums[i]
			}
			j++
		}
	}
	return j
}
