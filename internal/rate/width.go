package rate

import "sort"

// East Asian Wide / Fullwidth code point ranges (the classic wcwidth "wide"
// set). Characters here render two terminal cells, so display-width padding
// must account for them. Implemented inline to keep the binary dependency-free.
var wideRanges = [][2]rune{
	{0x1100, 0x115F},   // Hangul Jamo
	{0x2329, 0x232A},   // angle brackets
	{0x2E80, 0x303E},   // CJK radicals, Kangxi, CJK symbols & punctuation
	{0x3041, 0x33FF},   // Hiragana, Katakana, CJK symbols
	{0x3400, 0x4DBF},   // CJK Ext A
	{0x4E00, 0x9FFF},   // CJK Unified Ideographs
	{0xA000, 0xA4CF},   // Yi
	{0xAC00, 0xD7A3},   // Hangul Syllables
	{0xF900, 0xFAFF},   // CJK Compatibility Ideographs
	{0xFE10, 0xFE19},   // vertical forms
	{0xFE30, 0xFE6F},   // CJK compatibility forms, small form variants
	{0xFF00, 0xFF60},   // Fullwidth forms
	{0xFFE0, 0xFFE6},   // Fullwidth signs
	{0x1F300, 0x1FAFF}, // emoji & symbols (mostly wide)
	{0x20000, 0x3FFFD}, // CJK Ext B+ and supplementary ideographic plane
}

// runeWidth returns the number of terminal cells a rune occupies (1 or 2).
func runeWidth(r rune) int {
	i := sort.Search(len(wideRanges), func(i int) bool {
		return wideRanges[i][1] >= r
	})
	if i < len(wideRanges) && r >= wideRanges[i][0] && r <= wideRanges[i][1] {
		return 2
	}
	return 1
}

// displayWidth returns the total terminal cell width of s.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}
