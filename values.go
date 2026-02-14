package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

type ValueDef struct {
	typ         *TypeDef
	size        int
	hasValue    bool
	intValue    int64
	floatValue  float64
	boolValue   bool
	stringValue string
	regNo       int
}

var (
	False   = ValueDef{typ: &BoolType, hasValue: true, boolValue: false}
	True    = ValueDef{typ: &BoolType, hasValue: true, boolValue: true}
	NoValue = ValueDef{typ: &NoneType, hasValue: true, boolValue: false}
)

// CanAssign is true if we can assign type "src" to "dst"
func CanAssign(dst PrimaryType, src PrimaryType) bool {
	return dst == TYP_I16 && (src == TYP_I16 || src == TYP_U8) ||
		dst == TYP_I32 && (src == TYP_I32 || src == TYP_I16 || src == TYP_U8) ||
		dst == TYP_I64 && (src == TYP_I32 || src == TYP_U32 || src == TYP_U16 || src == TYP_I16 || src == TYP_U8) ||
		dst == TYP_U8 && src == TYP_U8 ||
		dst == TYP_U16 && (src == TYP_U16 || src == TYP_U8) ||
		dst == TYP_U32 && (src == TYP_U32 || src == TYP_U16 || src == TYP_U8)
}

func CommonType(t1 PrimaryType, t2 PrimaryType) PrimaryType {
	if t1 == TYP_I64 || t2 == TYP_I64 {
		return TYP_I64
	}
	if t1 == TYP_I32 && t2 < TYP_I32 || t2 == TYP_I32 && t1 < TYP_I32 {
		return TYP_I32
	}
	if t1 == TYP_U32 && (t2 == TYP_I16 || t2 == TYP_I32) {
		return TYP_I64
	}
	if t2 == TYP_U32 && (t1 == TYP_I16 || t1 == TYP_I32) {
		return TYP_I64
	}
	if (t1 == TYP_I16 || t2 == TYP_I16) && t1 != TYP_U16 && t2 != TYP_U16 {
		return TYP_I16
	}
	if t1 == TYP_F64 || t2 == TYP_F64 {
		return TYP_F64
	}
	return TYP_NONE
}

func GenerateOp(s *state, op Token, val1 ValueDef, val2 ValueDef) {
	var result ValueDef
	if val1.hasValue && val2.hasValue && val1.typ.pt == val2.typ.pt {
		// Both operands are constant. Evaluate at compile time.
		switch op {
		case TOK_PLUS:
			result.intValue = val1.intValue + val2.intValue
			result.floatValue = val1.floatValue + val2.floatValue
		case TOK_MINUS:
			result.intValue = val1.intValue - val2.intValue
			result.floatValue = val1.floatValue - val2.floatValue
		case TOK_MULT:
			result.intValue = val1.intValue * val2.intValue
			result.floatValue = val1.floatValue * val2.floatValue
		case TOK_DIV:
			result.intValue = val1.intValue / val2.intValue
			result.floatValue = val1.floatValue / val2.floatValue
		}
	}
	slog.Info("Generate", "Op", TokenNames[op])
	EmitOp(s, op)
}

func StringToValue(s string) (value ValueDef, err error) {
	if strings.ContainsRune(s, '.') {
		var num float64
		num, err = strconv.ParseFloat(s, 64)
		if err != nil {
			return NoValue, err
		}
		value.typ.pt = TYP_F64
		value.floatValue = num
	} else {
		var num int64
		num, err = strconv.ParseInt(s, 10, 64)
		if err == nil {
			if num >= 0 && num <= 255 {
				value.typ = TypeDefs["U8"]
			} else if num >= -32768 && num <= 32767 {
				value.typ = TypeDefs["I16"]
			} else if num >= 32768 && num <= 65536 {
				value.typ = TypeDefs["U16"]
			} else if num >= -2147483648 && num <= 2147483647 {
				value.typ = TypeDefs["I32"]
			} else if num >= 2147483648 && num <= 4294967296 {
				value.typ = TypeDefs["U32"]
			} else {
				value.typ = TypeDefs["I64"]
			}
			value.intValue = num
			return value, nil
		}
	}
	return NoValue, fmt.Errorf("Not a value: %s", s)
}
