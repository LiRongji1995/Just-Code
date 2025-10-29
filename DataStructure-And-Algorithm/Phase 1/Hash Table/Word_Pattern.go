package Hash_Table

import "strings"

func wordPattern(pattern string, s string) bool {
	words := strings.Fields(s)

	if len(words) != len(pattern) {
		return false
	}

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
