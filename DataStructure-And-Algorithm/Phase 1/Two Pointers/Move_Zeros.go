package Two_Pointers

func moveZeroes(nums []int) {
	for i, j := 0, 0; i < len(nums); i++ {
		if nums[i] != 0 {
			nums[j] = nums[i]
			if i != j {
				nums[i] = 0
			}
			j++
		}
	}
}
