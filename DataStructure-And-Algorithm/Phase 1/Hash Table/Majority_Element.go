package Hash_Table

func majorityElement(nums []int) int {
	//传统哈希
	//major := make(map[int]int)
	//n := len(nums)
	//
	//for _, v := range nums {
	//	major[v]++
	//	if major[v] > n/2 {
	//		return v
	//	}
	//}
	//return -1

	/*
		核心想法（直觉版）：

		因为多数元素出现次数 > n/2，说明它的数量比其他所有元素加起来还多。

		我可以做“投票对冲”：

		我维护一个“候选人 candidate”和它的“票数 count”

		遍历数组：

		如果当前数 == candidate，count++

		否则 count--（互相抵消一票）

		如果 count 变成 0，说明之前的票都被抵消掉了，我可以把 candidate 换成当前这个数，count 重置为 1

		遍历完以后，candidate 一定就是多数元素
		-------------------------------------------------------------------------------------
		为什么这个逻辑是对的？

		因为每一次“不同的两个数互相抵消”之后，多数元素还是剩得最多的，它最终会撑到最后。

		可以把它想象成：所有非多数派的人不停和多数派对拼、互相同归于尽，但因为多数派人本来就 > 其它所有人加起来，
		最后留下的活口必定是多数派。
	*/

	candidate := 0
	count := 0

	for _, x := range nums {
		if count == 0 {
			candidate = x
			count = 1
		} else if x == candidate {
			count++
		} else {
			count--
		}
	}
	return candidate
}
