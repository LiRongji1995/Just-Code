package Array

func maxArea(height []int) int {
	max := 0
	start := 0
	end := len(height) - 1

	for start < end {
		width := end - start
		high := 0
		if height[end] < height[start] {
			high = height[end]
			end--
		} else {
			high = height[start]
			start++
		}
		size := width * high
		if size > max {
			max = size
		}
	}
	return max
}
