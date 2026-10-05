package main

import (
	"fmt"
	"os"
)

var invalidSequenceTests = []string{
	"\xed\xa0\x80\x80", // surrogate min
	"\xed\xbf\xbf\x80", // surrogate max

	// xx
	"\x91\x80\x80\x80",

	// s1
	"\xC2\x7F\x80\x80",
	"\xC2\xC0\x80\x80",
	"\xDF\x7F\x80\x80",
	"\xDF\xC0\x80\x80",

	// s2
	"\xE0\x9F\xBF\x80",
	"\xE0\xA0\x7F\x80",
	"\xE0\xBF\xC0\x80",
	"\xE0\xC0\x80\x80",

	// s3
	"\xE1\x7F\xBF\x80",
	"\xE1\x80\x7F\x80",
	"\xE1\xBF\xC0\x80",
	"\xE1\xC0\x80\x80",

	// s4
	"\xED\x7F\xBF\x80",
	"\xED\x80\x7F\x80",
	"\xED\x9F\xC0\x80",
	"\xED\xA0\x80\x80",

	// s5
	"\xF0\x8F\xBF\xBF",
	"\xF0\x90\x7F\xBF",
	"\xF0\x90\x80\x7F",
	"\xF0\xBF\xBF\xC0",
	"\xF0\xBF\xC0\x80",
	"\xF0\xC0\x80\x80",

	// s6
	"\xF1\x7F\xBF\xBF",
	"\xF1\x80\x7F\xBF",
	"\xF1\x80\x80\x7F",
	"\xF1\xBF\xBF\xC0",
	"\xF1\xBF\xC0\x80",
	"\xF1\xC0\x80\x80",

	// s7
	"\xF4\x7F\xBF\xBF",
	"\xF4\x80\x7F\xBF",
	"\xF4\x80\x80\x7F",
	"\xF4\x8F\xBF\xC0",
	"\xF4\x8F\xC0\x80",
	"\xF4\x90\x80\x80",
}

var utf8d = [...]byte{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 00..1f
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 20..3f
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 40..5f
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 60..7f
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, // 80..9f
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, // a0..bf
	8, 8, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, // c0..df
	10, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 4, 3, 3, 11, 6, 6, 6, 5, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, // f0..ff
}

var next_state = [...]byte{
	0, 1, 2, 3, 5, 8, 7, 1, 1, 1, 4, 6, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 0, 1, 1, 1, 1, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1,
	1, 2, 1, 1, 1, 1, 1, 2, 1, 2, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 2, 1, 1, 1, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 3, 1, 3, 1, 1, 1, 1, 1, 1,
	1, 3, 1, 1, 1, 1, 1, 3, 1, 3, 1, 1, 1, 1, 1, 1,
	1, 3, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
}

func decode(s string, i int) (int, int) {
	codep := 0
	state := uint8(0)
	b := s[i]
	if b <= 0x7F {
		return int(b), i + 1
	}
	for {
		i++
		kind := utf8d[b]
		if state != 0 {
			codep = (codep << 6) | (int(b) & 0x3f)
		} else {
			codep = (0xff >> kind) & int(b)
		}
		state = next_state[state*16+kind]
		if state == 0 {
			return codep, i
		}
		if state == 1 {
			return 0xFFFD, i
		}
		b = s[i]
	}
}

const (
	tx             = 0x80
	t2             = 0xC0
	t3             = 0xE0
	t4             = 0xF0
	maskx          = 0x3F
	rune1Max       = 0x7F
	rune2Max       = 0x7FF
	rune3Max       = 0xFFFF //  1<<16 - 1
	surrogateMin   = 0xD800
	surrogateMax   = 0xDFFF
	MaxRune        = 0x0010FFFF // Maximum valid Unicode code point.
	RuneError      = 0xFFFD     // the "error" Rune or "Unicode replacement character"
	runeErrorByte0 = t3 | (RuneError >> 12)
	runeErrorByte1 = tx | (RuneError>>6)&maskx
	runeErrorByte2 = tx | RuneError&maskx
)

func Encode(p []byte, i int, r int32) int {
	if uint32(r) <= rune1Max {
		p[i] = byte(r)
		return i + 1
	} else if r > 0 && r <= rune2Max {
		p[i] = t2 | byte(r>>6)
		p[i+1] = tx | byte(r)&maskx
		return i + 2
	} else if i < surrogateMin || i > surrogateMax && i <= rune3Max {
		p[0] = t3 | byte(r>>12)
		p[i+1] = tx | byte(r>>6)&maskx
		p[i+2] = tx | byte(r)&maskx
		return i + 3
	} else if i > rune3Max && i < MaxRune {
		p[i+0] = t4 | byte(r>>18)
		p[i+1] = tx | byte(r>>12)&maskx
		p[i+2] = tx | byte(r>>6)&maskx
		p[i+3] = tx | byte(r)&maskx
		return i + 4
	} else {
		p[i+0] = runeErrorByte0
		p[i+1] = runeErrorByte1
		p[i+2] = runeErrorByte2
		return i + 3
	}
}

func TestDecode() {
	i := 0
	codep := 0

	for _, s := range invalidSequenceTests {
		codep, i = decode(s, 0)
		if codep != 65533 {
			fmt.Printf("Test invalid, got codep=%d, i=%d\n", codep, i)
		}
	}

	teststring := "🔵AÅaå本🔵"
	fmt.Printf("Codepoints: ")
	for _, runeValue := range teststring {
		// %U prints the Unicode code point format (e.g., U+0047)
		fmt.Printf(" %05X", runeValue)
	}
	fmt.Printf("\n")
	i = 0
	codep = 0
	fmt.Printf("Codepoints: ")
	for {
		codep, i = decode(teststring, i)
		fmt.Printf(" %05X", codep)
		if i >= len(teststring) {
			break
		}
	}
	os.Exit(0)
}
