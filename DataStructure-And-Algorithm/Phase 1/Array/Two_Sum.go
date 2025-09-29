package Array

func twoSum(nums []int, target int) []int {
	m := make(map[int]int)
	for i := 0; i < len(nums); i++ {
		ans := target - nums[i]
		if _, ok := m[ans]; ok {
			return []int{i, m[ans]}
		}
		m[nums[i]] = i
	}
	return nil
}
