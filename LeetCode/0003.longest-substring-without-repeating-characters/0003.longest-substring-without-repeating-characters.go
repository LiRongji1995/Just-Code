package _003_longest_substring_without_repeating_characters

import "math"

//这⼀题和第 438 题，第 76 题，第 567 题类似，⽤的思想都是"滑动窗⼝"
//滑动窗⼝的右边界不断的右移，只要没有重复的字符，就持续向右扩⼤窗⼝边界。⼀旦出现了重复字符，就需要缩
//⼩左边界，直到重复的字符移出了左边界，然后继续移动滑动窗⼝的右边界。以此类推，每次移动需要计算当前⻓
//度，并判断是否需要更新最⼤⻓度，最终最⼤的值就是题⽬中的所求。
func lengthOfLongestSubstring(s string) int {
	if len(s) == 0 {
		return 0
	}
	var freq [256]int //256 是因为 ASCII 字符集有 256 个可能的字符。
	result, left, right := 0, 0, -1
	//使用 -1 的好处：使用 -1 初始化，代码中的检查条件（如 right + 1 < len(s)）清晰地指示何时可以尝试扩展右边界，同时避免直接访问 s[right] 的问题。
	//避免错误：如果 right 初始化为 0，在窗口一开始是空的时候可能需要特殊处理，增加了代码的复杂度。
	for left < len(s) {
		if right+1 < len(s) && freq[s[right+1]-'a'] == 0 {
			freq[s[right+1]-'a']++ //-'a' 的目的是将字符转换为对应的索引值，以便在数组（这里是 freq 数组）中正确地记录字符的频率。
			right++
		} else {
			freq[s[left]-'a']--
			left++
		}
		result = int(math.Max(float64(result), float64(right-left+1)))
	}
	return result
}
