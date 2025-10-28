package Hash_Table

func intersection(num1 []int, num2 []int) []int {
	seen := make(map[int]bool)
	for _, x := range num1 {
		seen[x] = true
	}

	resSet := make(map[int]bool)
	for _, y := range num2 {
		if seen[y] {
			resSet[y] = true
		}
	}
	res := make([]int, 0, len(resSet))
	for v := range resSet {
		res = append(res, v)
	}
	return res
}
