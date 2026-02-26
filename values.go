package main

import (
	"fmt"
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
	False     = ValueDef{typ: &BoolType, hasValue: true, boolValue: false}
	True      = ValueDef{typ: &BoolType, hasValue: true, boolValue: true}
	NoValue   = ValueDef{typ: &NoneType, hasValue: false, boolValue: false}
	ZeroValue = ValueDef{typ: &PtrType, hasValue: true, intValue: 0, floatValue: 0, boolValue: false}
)

func GenerateOp(s *State, op Token, val1 ValueDef, val2 ValueDef) (ValueDef, error) {
	var result ValueDef
	if val1.typ.pt == TYP_F64 || val1.typ.pt == TYP_F32 {
		val2.floatValue = float64(val2.intValue)
	}
	if val2.typ.pt == TYP_F64 || val2.typ.pt == TYP_F32 {
		val1.floatValue = float64(val1.intValue)
	}
	if IsCompare(op) {
		EmitOp(s, op)
		result.typ = &BoolType
		return result, nil
	}
	if val1.hasValue && val2.hasValue {
		// Both operands are constant. Evaluate at compile time.
		result.typ = widest(val1, val2).typ
		result.hasValue = true
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
		case TOK_AND:
			result.intValue = val1.intValue & val2.intValue
		case TOK_OR:
			result.intValue = val1.intValue | val2.intValue
		case TOK_LOG_OR:
			result.boolValue = val1.boolValue || val2.boolValue
		case TOK_LOG_AND:
			result.boolValue = val1.boolValue && val2.boolValue
		default:
			// Invalid operand
			return NoValue, fmt.Errorf("invalid operation: %s", TokenNames[op])
		}
	} else if val1.hasValue {
		EmitOpConst(s, op, val1)
		result.typ = val2.typ
	} else if val2.hasValue {
		EmitOpConst(s, op, val2)
		result.typ = val2.typ
	} else {
		EmitOp(s, op)
		result.typ = val1.typ
	}
	return result, nil
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
			value.hasValue = true
			return value, nil
		}
	}
	return NoValue, fmt.Errorf("not a value: %s", s)
}

func widest(v1 ValueDef, v2 ValueDef) ValueDef {
	if v1.typ.pt > v2.typ.pt {
		return v1
	}
	return v2
}

func ValueAsString(v ValueDef) string {
	if v.typ.pt == TYP_U8 || v.typ.pt == TYP_U16 || v.typ.pt == TYP_U32 || v.typ.pt == TYP_I16 || v.typ.pt == TYP_I32 || v.typ.pt == TYP_I64 {
		return strconv.FormatInt(v.intValue, 10)
	} else if v.typ.pt == TYP_BOOL {
		if v.boolValue {
			return "true"
		}
		return "false"
	} else if v.typ.pt == TYP_F64 {
		return strconv.FormatFloat(v.floatValue, 'g', -1, 64)
	} else if v.typ.pt == TYP_F32 {
		return strconv.FormatFloat(v.floatValue, 'g', -1, 32)
	}
	return v.stringValue
}
