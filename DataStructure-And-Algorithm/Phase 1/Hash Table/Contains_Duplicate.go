package Hash_Table

func containsDuplicate(nums []int) bool {
	duplicate := make(map[int]int)
	for _, v := range nums {
		duplicate[v]++
		if duplicate[v] > 1 {
			return true
		}
	}
	return false
}
