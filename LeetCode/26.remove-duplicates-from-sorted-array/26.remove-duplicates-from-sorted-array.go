package _26_remove_duplicates_from_sorted_array

func removeDuplicates(nums []int) int {
	if len(nums) == 0 {
		return len(nums)
	}
	last, finder := 0, 0
	for last < len(nums)-1 {
		for nums[finder] == nums[last] {
			finder++
			if finder == len(nums) {
				return last + 1
			}
		}
		nums[last+1] = nums[finder]
		last++
	}
	return last + 1
}
