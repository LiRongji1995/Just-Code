package Two_Pointers

func duplicateZeros(arr []int) {
	n := len(arr)
	res := make([]int, 0, n*2)

	for _, v := range arr {
		if v == 0 {
			res = append(res, 0)
			if len(res) < n {
				res = append(res, 0)
			}
		} else {
			res = append(res, v)
		}
		if len(res) >= n {
			break
		}
	}
	copy(arr, res[:n])
}
