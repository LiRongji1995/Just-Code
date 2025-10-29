package Hash_Table

/*
Given two strings s and t, determine if they are isomorphic.

Two strings s and t are isomorphic if the characters in s can be replaced to get t.

All occurrences of a character must be replaced with another character while preserving the order
of characters. No two characters may map to the same character, but a character may map to itself.
*/
/*
isomorphic 同构
同构需要检查两件事：

每个 s[i] 始终映射到同一个 t[i]（不能反复变）

不同的 s 字符不能映射到同一个 t 字符（必须一对一）
*/

func isIsomorphic(s string, t string) bool {
	if len(s) != len(t) {
		return false
	}
	m1 := make(map[byte]byte)
	m2 := make(map[byte]byte)

	for i := 0; i < len(s); i++ {
		b1 := s[i]
		b2 := t[i]
		/*
			看看 b1（s 里的这个字符）之前有没有出现过。

			如果没出现过：我还没有为它定义“它应该变成哪个字符”，那我会在 else { m1[b1] = b2 } 里记录下来。

			如果出现过（ok == true）：那它以前就已经说过“我应该映射成 v”。

			现在又看到它跟一个新的字符 b2 配对。

			如果 v != b2，那就是前后不一致，直接违反同构条件，返回 false。
		*/
		if v, ok := m1[b1]; ok {
			if v != b2 {
				return false
			}
		} else {
			m1[b1] = b2
		}
		/*
			我们需要记录双向映射：

			一个 map：sChar -> tChar

			再一个 map：tChar -> sChar

			在遍历过程中：

			如果 s[i] 以前见过，那它应该总是映射到同一个 t[i]，否则返回 false

			如果 t[i] 以前已经被别的 s[j] 占用了，那也返回 false（因为必须一对一）
		*/
		if v, ok := m2[b2]; ok {
			if v != b1 {
				return false
			}
		} else {
			m2[b2] = b1
		}
	}
	return true
}
