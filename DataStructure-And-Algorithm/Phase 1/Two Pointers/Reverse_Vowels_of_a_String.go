package Two_Pointers

func reverseVowels(s string) string {
	cleanString := []byte(s)
	i, j := 0, len(cleanString)-1

	for i < j {
		for i < j && !isVowel(cleanString[i]) {
			i++
		}
		for i < j && !isVowel(cleanString[j]) {
			j--
		}
		if i < j {
			cleanString[i], cleanString[j] = cleanString[j], cleanString[i]
			i++
			j--
		}
	}

	return string(cleanString)
}

func isVowel(b byte) bool {
	return b == 'a' || b == 'e' || b == 'i' || b == 'o' || b == 'u' ||
		b == 'A' || b == 'E' || b == 'I' || b == 'O' || b == 'U'
}
