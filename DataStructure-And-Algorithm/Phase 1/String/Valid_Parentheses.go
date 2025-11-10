package String

func isValid(s string) bool {
	//用切片来当栈
	stack := []rune{}
	pairs := map[rune]rune{
		')': '(',
		']': '[',
		'}': '{',
	}
	for _, ch := range s {
		if left, ok := pairs[ch]; ok {
			if len(stack) == 0 || stack[len(stack)-1] != left {
				return false
			}

			stack = stack[:len(stack)-1]
		} else {
			stack = append(stack, ch)
		}
	}
	return len(stack) == 0
}
