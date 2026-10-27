package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/jkvatne/jkv/code"
)

// GenerateOp will handle the infix operations +,-,*,/,%,|,&,^,<,>,<=,>=,==,!=
// Integer operands are promoted to the smallest size that can accomondate both.
// F.ex. I16 op U16 results in an I32
// There are 4 different cases: const op const, tos op const, const op tos, tos op nos
func GenerateOp(op Token, val1 *ValueDef, val2 *ValueDef) (*ValueDef, error) {
	if val1.Typ == nil {
		var err error
		val1.Typ, err = TypeFromConstValue(val1)
		if err != nil {
			return &NoValue, fmt.Errorf("first operand is nil")
		}
	}
	if val2.Typ == nil {
		var err error
		val2.Typ, err = TypeFromConstValue(val2)
		if val1.Typ.Pt == code.TYP_F32 && val2.Typ.Pt == code.TYP_F64 {
			val2.Typ = &F32Type
		}
		if err != nil {
			return &NoValue, fmt.Errorf("second operand is nil")
		}
	}
	// For user defined types, both must be identical, or one operand must be a basic type.
	if !val1.Typ.Basic && !val2.Typ.Basic && val1.Typ != val2.Typ {
		return &NoValue, fmt.Errorf("operation on incompatible types %s and %s", val1.Typ.Pt.Name(), val2.Typ.Pt.Name())
	}
	if val1.IsConst && val2.IsConst {
		// If both operands are constant. Evaluate at compile time.
		return generateConstOpConst(op, val1, val2)
	} else if val2.IsConst {
		EmitAssertTosInRax("Get TOS before TosOpConst2")
		return generateTosOpConst(op, val1, val2)
	} else if val1.IsConst {
		EmitAssertTosInRax("Get TOS before TosOpConst1")
		return generateTosOpConst(Inverse(op), val2, val1)
	}
	EmitAssertTosInRax("Get TOS before TosOpNos")
	return generateTosOpNos(op, val1, val2)
}

// generateConstOpConst will calculate the result of the operation on the two constant values
// and return the constant result.
// The operations are : + - * / & |  %% == != < <= > >=
func generateConstOpConst(op Token, val1 *ValueDef, val2 *ValueDef) (result *ValueDef, err error) {
	result = new(ValueDef)
	result.IsConst = true
	f1, err := StrToConst(val1.StringValue)
	if err != nil {
		return nil, err
	}
	f2, err := StrToConst(val2.StringValue)

	val1.Typ, err = TypeFromConstValue(val1)
	val2.Typ, err = TypeFromConstValue(val2)

	result.Typ = widest(val1, val2).Typ

	if err != nil {
		return nil, err
	}
	switch op {
	case TOK_PLUS:
		result.StringValue = strconv.FormatFloat(f1+f2, 'f', -1, 64)
	case TOK_MINUS:
		result.StringValue = strconv.FormatFloat(f1-f2, 'f', -1, 64)
	case TOK_MULT:
		result.StringValue = strconv.FormatFloat(f1*f2, 'f', -1, 64)
	case TOK_DIV:
		// TODO Check for f2=0
		if result.Typ.Pt.IsInteger() {
			result.StringValue = strconv.FormatInt(int64(f1)/int64(f2), 10)
		} else {
			result.StringValue = strconv.FormatFloat(f1/f2, 'f', -1, 64)
		}
	case TOK_MOD:
		result.StringValue = strconv.FormatFloat(float64(int64(f1)%int64(f2)), 'f', -1, 64)
	case TOK_AND:
		result.StringValue = strconv.FormatFloat(float64(int64(f1)&int64(f2)), 'f', -1, 64)
	case TOK_AND_NOT:
		result.StringValue = strconv.FormatFloat(float64(int64(f1)&^int64(f2)), 'f', -1, 64)
	case TOK_OR:
		result.StringValue = strconv.FormatFloat(float64(int64(f1)|int64(f2)), 'f', -1, 64)
	case TOK_XOR:
		result.StringValue = strconv.FormatFloat(float64(int64(f1)^int64(f2)), 'f', -1, 64)
	case TOK_SHL:
		result.StringValue = strconv.FormatFloat(float64(uint64(f1)<<int(f2)), 'f', -1, 64)
	case TOK_SHR:
		result.StringValue = strconv.FormatFloat(float64(int64(f1)>>int64(f2)), 'f', -1, 64)
	case TOK_LOG_OR:
		result.Typ = &BoolType
		result.SetBoolValue(val1.BoolValue() || val2.BoolValue())
	case TOK_LOG_AND:
		result.Typ = &BoolType
		result.SetBoolValue(val1.BoolValue() && val2.BoolValue())
	case TOK_EQ:
		result.Typ = &BoolType
		result.SetBoolValue(math.Abs(f1-f2)/max(f1, f2, 1e-30) < 1e-7)
	case TOK_NE:
		result.Typ = &BoolType
		result.SetBoolValue(math.Abs(f1-f2)/max(f1, f2, 1e-30) > 1e-7)
	case TOK_LT:
		result.Typ = &BoolType
		result.SetBoolValue(f1 < f2)
	case TOK_LE:
		result.Typ = &BoolType
		result.SetBoolValue(f1 <= f2)
	case TOK_GT:
		result.Typ = &BoolType
		result.SetBoolValue(f1 > f2)
	case TOK_GE:
		result.Typ = &BoolType
		result.SetBoolValue(f1 >= f2)
	default:
		// Invalid operand
		return &NoValue, fmt.Errorf("const op const, invalid operation: %s", TokenNames[op])
	}
	return result, nil
}

// generateTosOpNos will generate code for the operation op on the two top entries on the stack.
func generateTosOpNos(op Token, val1, val2 *ValueDef) (*ValueDef, error) {
	EmitAssertTosInRax("Get TOS befor TosOpNos")
	if val1.Typ == nil {
		var err error
		val1.Typ, err = TypeFromConstValue(val1)
		if err != nil {
			return &NoValue, fmt.Errorf("first operand is nil")
		}
	}
	if val2.Typ == nil {
		var err error
		val2.Typ, err = TypeFromConstValue(val2)
		if err != nil {
			return &NoValue, fmt.Errorf("first operand is nil")
		}
	}
	if op.IsCompare() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			err := EmitCompareIntegers(op, false)
			return &ValueDef{Typ: &BoolType}, err
		} else if val1.Typ.Pt == code.TYP_F64 && val2.Typ.Pt == code.TYP_F64 {
			err := EmitCompareF64(op)
			return &ValueDef{Typ: &BoolType}, err
		} else if val1.Typ.Pt == code.TYP_F32 && val2.Typ.Pt == code.TYP_F32 {
			err := EmitCompareF32(op)
			return &ValueDef{Typ: &BoolType}, err
		} else if val1.Typ.Pt == code.TYP_STRING && val2.Typ.Pt == code.TYP_STRING {
			if op == TOK_EQ {
				EmitCompareStringsEq(val1.IsTempObj, val2.IsTempObj)
				return &ValueDef{Typ: &BoolType}, nil
			} else if op == TOK_NE {
				EmitCompareStringsNe(val1.IsTempObj, val2.IsTempObj)
				return &ValueDef{Typ: &BoolType}, nil
			}
		} else {
			return nil, fmt.Errorf("compare with invalid operands: %s", TokenNames[op])
		}
	} else if op.IsAritmetic() {
		if val1.Typ.Pt.IsInteger() && (val2.Typ.Pt.IsInteger() || val2.Typ.Pt == code.TYP_BOOL) {
			EmitIntegerOp(op)
			return val1, nil
		} else if val1.Typ.Pt == code.TYP_STRING && val2.Typ.Pt == code.TYP_STRING && op == TOK_PLUS {
			EmitConcat(val1.IsTempObj, val2.IsTempObj)
			return val1, nil
		}
		var err error
		if val1.Typ.Pt == code.TYP_F64 || val2.Typ.Pt == code.TYP_F64 {
			err = EmitF64Op(op, val1.Typ.Pt, val2.Typ.Pt)
			if err != nil {
				return nil, err
			}
		} else if val1.Typ.Pt == code.TYP_F32 || val2.Typ.Pt == code.TYP_F32 {
			err = EmitF32Op(op, val1.Typ.Pt, val2.Typ.Pt)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid operation: %s", TokenNames[op])
		}
		if val1.Typ.Pt == code.TYP_F64 {
			return val1, nil
		} else if val2.Typ.Pt == code.TYP_F64 {
			return val2, nil
		} else if val1.Typ.Pt == code.TYP_F32 {
			return val1, nil
		} else if val2.Typ.Pt == code.TYP_F32 {
			return val2, nil
		}
		return val1, err
	}
	return nil, fmt.Errorf("tosnos operation %s not implemented", TokenNames[op])

}

// generateTosOpConst will evaluate Top Of Stack with a constant. The constant is found in val2
func generateTosOpConst(op Token, val1 *ValueDef, val2 *ValueDef) (*ValueDef, error) {
	var err error
	if val1.IsConst {
		op = Inverse(op)
	}
	var f1, f2 float64
	f1, _ = StrToConst(val1.StringValue)
	f2, _ = StrToConst(val2.StringValue)
	if val1.Typ == nil {
		var err error
		val1.Typ, err = TypeFromConstValue(val1)
		if err != nil {
			return &NoValue, fmt.Errorf("first operand is nil")
		}
	}
	if val2.Typ == nil {
		var err error
		val2.Typ, err = TypeFromConstValue(val2)
		if err != nil {
			return &NoValue, fmt.Errorf("first operand is nil")
		}
	}
	if op.IsCompare() {
		if val1.Typ.Pt == code.TYP_U64 && val2.Typ.Pt == code.TYP_U64 {
			err = EmitCompareIntConst64(op, val2.StringValue, true)
		} else if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			err = EmitCompareIntConst(op, int64(f2), false)
		} else if val1.Typ.Pt == code.TYP_F64 && val2.Typ.Pt == code.TYP_F64 {
			EmitAssertTosInRax("Get TOS before compare float const")
			err = EmitCompareF64Const(op, f2)
		} else if val1.Typ.Pt == code.TYP_F32 && val2.Typ.Pt == code.TYP_F32 {
			EmitAssertTosInRax("Get TOS before compare float const")
			err = EmitCompareF32Const(op, float32(f2))
		} else if val1.Typ.Pt == code.TYP_F32 && val2.Typ.Pt == code.TYP_F64 {
			EmitAssertTosInRax("Get TOS before compare float const")
			err = EmitCompareF32Const(op, float32(f2))
		} else if val1.Typ.Pt == code.TYP_STRING && val2.Typ.Pt == code.TYP_STRING {
			err = EmitCompareStrToLit(op, val2.StringValue, val2.StringLitNo, val1.IsTempObj)
		} else if val1.Typ.Pt == code.TYP_BOOL && val2.Typ.Pt == code.TYP_BOOL {
			err = EmitCompareIntConst(op, int64(f2), false)
		} else {
			err = fmt.Errorf("unknown type combination for compare %s with %s", val1.Typ.Pt.Name(), val2.Typ.Pt.Name())
		}
		return &ValueDef{Typ: &BoolType}, err
	} else if op.IsAritmetic() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			x := int64(f2) + int64(f1)
			if x > 0x7FFFFFFF || x < -0x7FFFFFFF {
				// err = fmt.Errorf("invalid integer combination for arithmetic")
				EmitPushConst(x, "")
				_, err2 := generateTosOpNos(op, val2, val2)
				return &ValueDef{Typ: val2.Typ}, err2
			}
			err = EmitOpIntConst(op, x, "TosOpConst")
		} else if val1.Typ.Pt.IsNumber() && val2.Typ.Pt.IsNumber() {
			// FloatLitNo is in either val1 or val2. The other is allways zero
			if val1.Typ.Pt == code.TYP_F64 {
				EmitOpF64Const(op, f1+f2)
			} else if val1.Typ.Pt == code.TYP_F32 {
				EmitOpF32Const(op, float32(f1+f2))
			}
			return &ValueDef{Typ: val1.Typ}, nil
		} else {
			err = fmt.Errorf("unknown type combination for '%s'", op.Name())
		}
		return &ValueDef{Typ: val1.Typ}, err
	}
	return &NoValue, fmt.Errorf("could not perform %s on types %s and %s", op.Name(), val1.Typ.Name(), val2.Typ.Name())
}
