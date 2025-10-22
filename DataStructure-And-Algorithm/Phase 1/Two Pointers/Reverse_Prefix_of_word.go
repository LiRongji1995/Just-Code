package Two_Pointers

import "strings"

func reversePrefix(word string, ch byte) string {
	k := strings.IndexByte(word, ch)
	if k == -1 {
		return word
	}
	b := []byte(word)
	for i, j := 0, k; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}
