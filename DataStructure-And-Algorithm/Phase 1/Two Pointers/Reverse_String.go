package Two_Pointers

func reverseString(s []byte) {
	n := len(s)
	i, j := 0, n-1
	for i < j {
		s[i], s[j] = s[j], s[i]
		i++
		j--
	}
}
