package Hash_Table

/*
只要关心：有没有两个下标 i、j，使得

nums[i] == nums[j]

|i - j| <= k

可以一边从左到右扫数组，一边用一个 map 记录“这个数上次出现的下标”。
*/
func containsNearbyDuplicate(nums []int, k int) bool {
	lastIndex := make(map[int]int)

	for i, v := range nums {
		if prev, ok := lastIndex[v]; ok {
			if i-prev <= k {
				return true
			}
		}
		lastIndex[v] = i
	}
	return false
}
