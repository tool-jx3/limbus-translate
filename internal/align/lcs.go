package align

// LCS returns the index pairs (i, j) of a longest common subsequence of a and
// b: a[i] == b[j] for every pair, and both i and j increase from one pair to
// the next. When several subsequences are longest, earlier pairs win.
func LCS[T comparable](a, b []T) [][2]int {
	return heaviestMatch(len(a), len(b), func(i, j int) int32 {
		if a[i] == b[j] {
			return 1
		}
		return 0
	})
}

// heaviestMatch pairs positions of two sequences of lengths n and m, with i
// and j both increasing, so that the summed weight of the pairs is as large
// as possible. weight(i, j) <= 0 means i and j may not be paired. Ties go to
// earlier pairs. It takes O(n·m) time and memory.
func heaviestMatch(n, m int, weight func(i, j int) int32) [][2]int {
	// best[i*w+j] is the largest total weight for the suffixes from i and j.
	w := m + 1
	best := make([]int32, (n+1)*w)
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			v := max(best[(i+1)*w+j], best[i*w+j+1])
			if x := weight(i, j); x > 0 {
				v = max(v, x+best[(i+1)*w+j+1])
			}
			best[i*w+j] = v
		}
	}

	var pairs [][2]int
	for i, j := 0, 0; i < n && j < m; {
		switch x := weight(i, j); {
		case x > 0 && best[i*w+j] == x+best[(i+1)*w+j+1]:
			pairs = append(pairs, [2]int{i, j})
			i++
			j++
		case best[i*w+j] == best[(i+1)*w+j]:
			i++
		default:
			j++
		}
	}
	return pairs
}
