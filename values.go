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

// EvalConstOp will calculate the result of the operation on the two constant values
// and return the constant result.
func EvalConstOp(s *State, op Token, val1 ValueDef, val2 ValueDef) (ValueDef, error) {
	var result ValueDef
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
	case TOK_GT:
		result.boolValue = val1.intValue > val2.intValue
	case TOK_GE:
		result.boolValue = val1.intValue >= val2.intValue
	case TOK_LT:
		result.boolValue = val1.intValue < val2.intValue
	case TOK_LE:
		result.boolValue = val1.intValue <= val2.intValue
	case TOK_EQ:
		result.boolValue = val1.intValue == val2.intValue
	case TOK_NE:
		result.boolValue = val1.intValue != val2.intValue
	default:
		// Invalid operand
		return NoValue, fmt.Errorf("invalid operation: %s", TokenNames[op])
	}
	return result, nil
}

func GenerateConstOp(s *State, op Token, val1, val2 ValueDef, inverse bool) (ValueDef, error) {
	return val1, nil
}

// GenerateOp will handle the infix operations +,-,*,/,%,|,&,^
// Integer operands are promoted to the smallest size that can accomondate both.
// F.ex. I16 op U16 results in an I32
// For user defined types, either both must be identical, or one operand must be a basic integer type.
func GenerateOp(s *State, op Token, val1 ValueDef, val2 ValueDef) (ValueDef, error) {
	var result ValueDef
	// Convert int values to float in case of mixed types.
	if val1.typ.pt != TYP_F64 && val1.typ.pt != TYP_F32 {
		val1.floatValue = float64(val1.intValue)
	}
	if val2.typ.pt != TYP_F64 && val2.typ.pt != TYP_F32 {
		val2.floatValue = float64(val2.intValue)
	}
	if !val1.typ.basic && !val2.typ.basic && val1.typ != val2.typ {
		return NoValue, fmt.Errorf("Operation on incompatible types %s and %s", val1.typ.pt.Name(), val2.typ.pt.Name())
	}
	// If both operands are constant. Evaluate at compile time.
	if val1.hasValue && val2.hasValue {
		return EvalConstOp(s, op, val1, val2)
	} else if val1.hasValue {
		return GenerateConstOp(s, op, val1, val2, true)
	} else if val2.hasValue {
		return GenerateConstOp(s, op, val1, val2, false)
	} else if val1.typ.pt.IsInteger() && val2.typ.pt.IsInteger() {
		// both operands are integers, do operation on the two top stack elements.
		EmitIntegerOp(s, op)
		ct, err := CommonType(val1.typ.pt, val2.typ.pt)
		if err != nil {
			return NoValue, err
		}
		result.typ = ct
		return result, nil
	}
	return NoValue, fmt.Errorf("invalid operation: %s", TokenNames[op])
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
