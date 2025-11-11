package String

func lengthOfLastWord(s string) int {
	i := len(s) - 1

	for i >= 0 && s[i] == ' ' {
		i--
	}
	res := 0

	for i >= 0 && s[i] != ' ' {
		i--
		res++
	}
	return res
}
