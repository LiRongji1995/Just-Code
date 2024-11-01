package _001_two_sum

func twoSum(nums []int, target int) []int { //返回值类型为 []int（一个整数切片,动态长度为切片，固定长度为数组）
	m := make(map[int]int)
	for i := 0; i < len(nums); i++ {
		another := target - nums[i]
		if _, ok := m[another]; ok {
			return []int{m[another], i}
		}
		m[nums[i]] = i
	}
	return nil
}
