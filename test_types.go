package main

import (
	"fmt"
	"testing"

	"github.com/jkvatne/jkv/code"
)

func TestCommonType(t *testing.T) {
	if len(TokenNames) != int(TOK_SIZE)+1 {
		panic("Token names length must be equal to TOK_SIZE")
	}
	// Testing the CommonType() function
	for t1 := code.TYP_U8; t1 <= code.TYP_F64; t1++ {
		for t2 := code.TYP_U8; t2 <= code.TYP_F64; t2++ {
			if t1 != code.TYP_RUNE && t2 != code.TYP_RUNE {
				tc, err := CommonType(t1, t2)
				if err != nil || tc == nil {
					t.Fail()
				}
				fmt.Printf("%10s %10s %10s\n", code.PrimaryTypeNames[t1], code.PrimaryTypeNames[t2], code.PrimaryTypeNames[tc.Pt])
				if tc.Name() == "None" {
					fmt.Printf("No common type for %s and %s\n", code.PrimaryTypeNames[t1], code.PrimaryTypeNames[t2])
				}
			}
		}
	}
}
