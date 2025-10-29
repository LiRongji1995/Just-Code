package Hash_Table

import "strings"

func wordPattern(pattern string, s string) bool {
	words := strings.Fields(s)

	if len(words) != len(pattern) {
		return false
	}
	/*
		字母 -> 单词，保证同一个字母不会对应不同单词

		单词 -> 字母，保证不同字母不会抢同一个单词
	*/
	charToWord := make(map[byte]string)
	wordToChar := make(map[string]byte)

	for i := range len(pattern) {
		p := pattern[i]
		w := words[i]

		if mappedWord, ok := charToWord[p]; ok {
			if mappedWord != w {
				return false
			}
		} else {
			charToWord[p] = w
		}

		if mappedChar, ok := wordToChar[w]; ok {
			if mappedChar != p {
				return false
			}
		} else {
			wordToChar[w] = p
		}
	}
	return true
}
