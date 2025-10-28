package Two_Pointers

func nextNum(n int) int {
	res := 0
	for n > 0 {
		num := n % 10
		res = res + num*num
		n /= 10
	}
	return res
}

func isHappy(n int) bool {

	slow := n
	fast := n

	for {
		slow = nextNum(slow)
		fast = nextNum(nextNum(fast))

		if slow == 1 || fast == 1 {
			return true
		}

		if slow == fast {
			return false
		}

	}
}
