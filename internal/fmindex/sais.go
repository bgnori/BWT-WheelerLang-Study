package fmindex

// buildSuffixArraySAIS builds the suffix array of text using the SA-IS
// (Suffix Array – Induced Sorting) algorithm by Nong, Zhang & Chan (2009)
// in O(n) time and O(n) space.
//
// text must end with byte 0 (sentinel) which is the unique minimum character.
func buildSuffixArraySAIS(text []byte) []int {
	n := len(text)
	if n == 0 {
		return nil
	}
	if n == 1 {
		return []int{0}
	}
	// Convert to int slice so that the recursive helper can work with
	// arbitrary-alphabet integer strings (the reduced string after naming).
	s := make([]int, n)
	for i, b := range text {
		s[i] = int(b)
	}
	sa := make([]int, n)
	saisInt(s, sa, 256)
	return sa
}

// saisInt runs SA-IS on integer string s (alphabet [0, alphabetSize)) and
// writes the suffix array into sa (caller must allocate len(s) elements).
//
// Invariant: s[len(s)-1] == 0 and is the unique minimum element (sentinel).
func saisInt(s, sa []int, alphabetSize int) {
	n := len(s)

	// ---- 1. Classify each suffix as S-type (true) or L-type (false). -------
	// Sentinel is always S-type.
	// s[i] is S-type if s[i] < s[i+1], or s[i]==s[i+1] and s[i+1] is S-type.
	t := make([]bool, n)
	t[n-1] = true
	for i := n - 2; i >= 0; i-- {
		t[i] = s[i] < s[i+1] || (s[i] == s[i+1] && t[i+1])
	}

	// isLMS: Left-Most S-suffix – an S-type position immediately following
	// an L-type position.  Position 0 is never LMS by definition.
	isLMS := func(i int) bool {
		return i > 0 && t[i] && !t[i-1]
	}

	// ---- 2. Helpers for bucket boundaries. ---------------------------------
	// computeBuckets returns the last index (end=true) or first index
	// (end=false) of each character's bucket in the suffix array.
	computeBuckets := func(end bool) []int {
		freq := make([]int, alphabetSize)
		for _, c := range s {
			freq[c]++
		}
		bkt := make([]int, alphabetSize)
		sum := 0
		for i, f := range freq {
			if end {
				sum += f
				bkt[i] = sum - 1
			} else {
				bkt[i] = sum
				sum += f
			}
		}
		return bkt
	}

	// ---- 3. Induced sort given an ordered list of LMS positions. -----------
	// Places each LMS suffix at the end of its bucket (in reverse order),
	// then sweeps left→right to induce L-type suffixes, and right→left to
	// induce S-type suffixes.
	inducedSort := func(sortedLMS []int) {
		for i := range sa {
			sa[i] = -1
		}
		// Place LMS at bucket ends.
		bkt := computeBuckets(true)
		for i := len(sortedLMS) - 1; i >= 0; i-- {
			p := sortedLMS[i]
			sa[bkt[s[p]]] = p
			bkt[s[p]]--
		}
		// Induce L-type (left → right).
		bkt = computeBuckets(false)
		for i := 0; i < n; i++ {
			if sa[i] <= 0 { // unset (-1) or position 0 (no predecessor)
				continue
			}
			j := sa[i] - 1
			if !t[j] { // j is L-type
				sa[bkt[s[j]]] = j
				bkt[s[j]]++
			}
		}
		// Induce S-type (right → left).
		bkt = computeBuckets(true)
		for i := n - 1; i >= 0; i-- {
			if sa[i] <= 0 {
				continue
			}
			j := sa[i] - 1
			if t[j] { // j is S-type
				sa[bkt[s[j]]] = j
				bkt[s[j]]--
			}
		}
	}

	// ---- 4. Collect LMS positions (left to right). -------------------------
	var lmsList []int
	for i := 1; i < n; i++ {
		if isLMS(i) {
			lmsList = append(lmsList, i)
		}
	}

	// First induced sort uses LMS positions in document order (approximate).
	inducedSort(lmsList)

	// ---- 5. Name LMS substrings. -------------------------------------------
	// Two LMS substrings are equal iff they agree on every character and
	// type until both reach their respective next LMS position simultaneously.
	nameArr := make([]int, n)
	for i := range nameArr {
		nameArr[i] = -1
	}
	curName := 0
	prev := -1
	for _, pos := range sa {
		if !isLMS(pos) {
			continue
		}
		newName := prev < 0
		if !newName {
			for d := 0; ; d++ {
				lms1 := d > 0 && isLMS(pos+d)
				lms2 := d > 0 && isLMS(prev+d)
				if s[pos+d] != s[prev+d] || t[pos+d] != t[prev+d] || lms1 != lms2 {
					newName = true
					break
				}
				if lms1 { // both ended at the same offset → equal substrings
					break
				}
			}
		}
		if newName {
			curName++
			prev = pos
		}
		nameArr[pos] = curName - 1
	}

	// ---- 6. Build reduced string. ------------------------------------------
	// Compact the assigned names left-to-right; remember original positions.
	reducedS := make([]int, 0, len(lmsList))
	reducedPos := make([]int, 0, len(lmsList))
	for i := 0; i < n; i++ {
		if nameArr[i] >= 0 {
			reducedS = append(reducedS, nameArr[i])
			reducedPos = append(reducedPos, i)
		}
	}
	m := len(reducedS)

	// ---- 7. Sort reduced suffix array. -------------------------------------
	reducedSA := make([]int, m)
	if curName < m {
		// Not all names unique: recurse on the reduced string.
		saisInt(reducedS, reducedSA, curName)
	} else {
		// All names unique: direct SA (names are a permutation of [0,m)).
		for i, v := range reducedS {
			reducedSA[v] = i
		}
	}

	// ---- 8. Map reduced SA back to original LMS positions. -----------------
	sortedLMS := make([]int, m)
	for i, v := range reducedSA {
		sortedLMS[i] = reducedPos[v]
	}

	// ---- 9. Final induced sort with correctly ordered LMS suffixes. --------
	inducedSort(sortedLMS)
}
