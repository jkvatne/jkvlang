package main

import (
	"fmt"
	"strconv"
	"strings"
)

type ValueDef struct {
	Typ         *TypeDef
	HasValue    bool
	IntValue    int64
	FloatValue  float64
	BoolValue   bool
	StringValue string
}

var (
	False     = ValueDef{Typ: &BoolType, HasValue: true, BoolValue: false}
	True      = ValueDef{Typ: &BoolType, HasValue: true, BoolValue: true}
	NoValue   = ValueDef{Typ: &NoneType, HasValue: false, BoolValue: false}
	ZeroValue = ValueDef{Typ: &PtrType, HasValue: true, IntValue: 0, FloatValue: 0, BoolValue: false}
)

// EvalConstOp will calculate the result of the operation on the two constant values
// and return the constant result.
func EvalConstOp(s *State, op Token, val1 ValueDef, val2 ValueDef) (ValueDef, error) {
	var result ValueDef
	result.Typ = widest(val1, val2).Typ
	result.HasValue = true
	switch op {
	case TOK_PLUS:
		result.IntValue = val1.IntValue + val2.IntValue
		result.FloatValue = val1.FloatValue + val2.FloatValue
	case TOK_MINUS:
		result.IntValue = val1.IntValue - val2.IntValue
		result.FloatValue = val1.FloatValue - val2.FloatValue
	case TOK_MULT:
		result.IntValue = val1.IntValue * val2.IntValue
		result.FloatValue = val1.FloatValue * val2.FloatValue
	case TOK_DIV:
		result.IntValue = val1.IntValue / val2.IntValue
		result.FloatValue = val1.FloatValue / val2.FloatValue
	case TOK_AND:
		result.IntValue = val1.IntValue & val2.IntValue
	case TOK_OR:
		result.IntValue = val1.IntValue | val2.IntValue
	case TOK_LOG_OR:
		result.BoolValue = val1.BoolValue || val2.BoolValue
	case TOK_LOG_AND:
		result.BoolValue = val1.BoolValue && val2.BoolValue
	case TOK_GT:
		result.BoolValue = val1.IntValue > val2.IntValue
	case TOK_GE:
		result.BoolValue = val1.IntValue >= val2.IntValue
	case TOK_LT:
		result.BoolValue = val1.IntValue < val2.IntValue
	case TOK_LE:
		result.BoolValue = val1.IntValue <= val2.IntValue
	case TOK_EQ:
		result.BoolValue = val1.IntValue == val2.IntValue
	case TOK_NE:
		result.BoolValue = val1.IntValue != val2.IntValue
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
	if val1.Typ.Pt != TYP_F64 && val1.Typ.Pt != TYP_F32 {
		val1.FloatValue = float64(val1.IntValue)
	}
	if val2.Typ.Pt != TYP_F64 && val2.Typ.Pt != TYP_F32 {
		val2.FloatValue = float64(val2.IntValue)
	}
	if !val1.Typ.Basic && !val2.Typ.Basic && val1.Typ != val2.Typ {
		return NoValue, fmt.Errorf("Operation on incompatible types %s and %s", val1.Typ.Pt.Name(), val2.Typ.Pt.Name())
	}
	// If both operands are constant. Evaluate at compile time.
	if val1.HasValue && val2.HasValue {
		return EvalConstOp(s, op, val1, val2)
	} else if val1.HasValue {
		return GenerateConstOp(s, op, val1, val2, true)
	} else if val2.HasValue {
		return GenerateConstOp(s, op, val1, val2, false)
	} else if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
		// both operands are integers, do operation on the two top stack elements.
		EmitIntegerOp(s, op)
		ct, err := CommonType(val1.Typ.Pt, val2.Typ.Pt)
		if err != nil {
			return NoValue, err
		}
		result.Typ = ct
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
		value.Typ.Pt = TYP_F64
		value.FloatValue = num
	} else {
		var num int64
		num, err = strconv.ParseInt(s, 10, 64)
		if err == nil {
			if num >= 0 && num <= 255 {
				value.Typ = TypeDefs["U8"]
			} else if num >= -32768 && num <= 32767 {
				value.Typ = TypeDefs["I16"]
			} else if num >= 32768 && num <= 65536 {
				value.Typ = TypeDefs["U16"]
			} else if num >= -2147483648 && num <= 2147483647 {
				value.Typ = TypeDefs["I32"]
			} else if num >= 2147483648 && num <= 4294967296 {
				value.Typ = TypeDefs["U32"]
			} else {
				value.Typ = TypeDefs["I64"]
			}
			value.IntValue = num
			value.HasValue = true
			return value, nil
		}
	}
	return NoValue, fmt.Errorf("not a value: %s", s)
}

func widest(v1 ValueDef, v2 ValueDef) ValueDef {
	if v1.Typ.Pt > v2.Typ.Pt {
		return v1
	}
	return v2
}

func ValueAsString(v ValueDef) string {
	if v.Typ.Pt == TYP_U8 || v.Typ.Pt == TYP_U16 || v.Typ.Pt == TYP_U32 || v.Typ.Pt == TYP_I16 || v.Typ.Pt == TYP_I32 || v.Typ.Pt == TYP_I64 {
		return strconv.FormatInt(v.IntValue, 10)
	} else if v.Typ.Pt == TYP_BOOL {
		if v.BoolValue {
			return "true"
		}
		return "false"
	} else if v.Typ.Pt == TYP_F64 {
		return strconv.FormatFloat(v.FloatValue, 'g', -1, 64)
	} else if v.Typ.Pt == TYP_F32 {
		return strconv.FormatFloat(v.FloatValue, 'g', -1, 32)
	}
	return v.StringValue
}
