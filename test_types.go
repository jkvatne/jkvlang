package main

import (
	"fmt"
	"testing"
)

func TestCommonType(t *testing.T) {
	if len(TokenNames) != int(TOK_SIZE)+1 {
		panic("Token names length must be equal to TOK_SIZE")
	}
	// Testing the CommonType() function
	for t1 := TYP_U8; t1 <= TYP_F64; t1++ {
		for t2 := TYP_U8; t2 <= TYP_F64; t2++ {
			if t1 != TYP_RUNE && t2 != TYP_RUNE {
				tc, err := CommonType(t1, t2)
				if err != nil || tc == nil {
					t.Fail()
				}
				fmt.Printf("%10s %10s %10s\n", PrimaryTypeNames[t1], PrimaryTypeNames[t2], PrimaryTypeNames[tc.Pt])
				if tc.Name() == "None" {
					fmt.Printf("No common type for %s and %s\n", PrimaryTypeNames[t1], PrimaryTypeNames[t2])
				}
			}
		}
	}
}
