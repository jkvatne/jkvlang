package main

import (
	"fmt"
	"math"
	"os"
)

// pack64 takes m, e and returns f = m * 2**e.
// It assumes the caller has provided a 53-bit mantissa m
// and an exponent that is in range for the mantissa.
func pack64(m uint64, e int) float64 {
	if m&(1<<52) == 0 {
		return math.Float64frombits(m)
	}
	return math.Float64frombits(m&^(1<<52) | uint64(1075+e)<<52)
}

type unrounded uint64

func unround(x float64) unrounded {
	fmt.Printf("x=%f\n", x)
	f := x * 4
	fmt.Printf("f=%f\n", f)
	f = math.Floor(f)
	fmt.Printf("Floor(f)=%f\n", f)
	u := unrounded(f)
	fmt.Printf("u=%d\n---\n", u)

	return unrounded(math.Floor(4*x)) | bool2[unrounded](math.Floor(4*x) != 4*x)
}

func (u unrounded) String() string {
	return fmt.Sprintf("⟨%d.%d%s⟩", u>>2, 5*((u>>1)&1), "+"[1-u&1:])
}

// bool2 converts b to an integer: 1 for true, 0 for false.
func bool2[T ~int | ~uint64](b bool) T {
	if b {
		return 1
	}
	return 0
}

func TestUnrounded() {
	u := unround(6.1)
	fmt.Printf("u=%d\n", u)

	os.Exit(0)

	f := pack64(0x0010_0000_0000_0000, -52) // = 1.0
	fmt.Printf("%g\n", f)
	f = pack64(0x0018000000000000, -52) // = 1.5
	fmt.Printf("%g\n", f)
	f = pack64(0x0018000000000000, -51) // = 3
	fmt.Printf("%g\n", f)

	fmt.Printf("    x    raw  str\n")
	for _, x := range []float64{6, 6.001, 6.499, 6.5, 6.501, 6.999, 7} {
		u := unround(x)
		fmt.Printf("%f  %d, %s\n", x, uint64(u), u.String())
	}

	os.Exit(0)
}

var values = []float64{1.234e1, 1.9999e-44, 2e123, 1e99, 1e-99, 1e100, 1e-100, 1.99e-99}

func TestFormat() {
	for i := 0; i < len(values); i++ {
		fmt.Printf("%0.16e\n", values[i])
	}
}
