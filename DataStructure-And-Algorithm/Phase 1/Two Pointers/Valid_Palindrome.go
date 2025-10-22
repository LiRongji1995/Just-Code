package Two_Pointers

import "strings"

func isPalindrome(s string) bool {
	s = strings.ToLower(s)
	i, j := 0, len(s)-1
	for i < j {

		if i < j && !isChar(s[i]) {
			i++
			continue
		}

		if i < j && !isChar(s[j]) {
			j--
			continue
		}

		if s[i] != s[j] {
			return false
		}
		i++
		j--
	}
	return true
}

func isChar(c byte) bool {
	if ('a' <= c && c <= 'z') || ('0' <= c && c <= '9') {
		return true
	} else {
		return false
	}
}
