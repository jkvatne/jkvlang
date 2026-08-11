package main

import (
	"fmt"
	"math"

	"github.com/jkvatne/jkv/code"
)

// GenerateOp will handle the infix operations +,-,*,/,%,|,&,^,<,>,<=,>=,==,!=
// Integer operands are promoted to the smallest size that can accomondate both.
// F.ex. I16 op U16 results in an I32
// There are 4 different cases: const op const, tos op const, const op tos, tos op nos
func GenerateOp(op Token, val1 *ValueDef, val2 *ValueDef) (*ValueDef, error) {
	// Convert int values to float in case of mixed types.
	if val1.Typ.Pt != code.TYP_F64 && val1.Typ.Pt != code.TYP_F32 {
		val1.FloatValue = float64(val1.IntValue)
	}
	if val2.Typ.Pt != code.TYP_F64 && val2.Typ.Pt != code.TYP_F32 {
		val2.FloatValue = float64(val2.IntValue)
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
	} else {
		EmitAssertTosInRax("Get TOS before TosOpNos")
		return emitTosOpNos(op, val1, val2)
	}
}

// generateConstOpConst will calculate the result of the operation on the two constant values
// and return the constant result.
// The operations are : + - * / & |  %% == != < <= > >=
func generateConstOpConst(op Token, val1 *ValueDef, val2 *ValueDef) (result *ValueDef, err error) {
	result = new(ValueDef)
	result.Typ = widest(val1, val2).Typ
	result.IsConst = true
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
		if val2.Typ.Pt.IsInteger() {
			if val2.IntValue == 0 {
				return &NoValue, fmt.Errorf("can not divide by zero")
			}
			result.IntValue = val1.IntValue / val2.IntValue
		} else if val2.Typ.Pt.IsFloat() {
			result.FloatValue = val1.FloatValue / val2.FloatValue
		}
	case TOK_MOD:
		if val2.Typ.Pt.IsInteger() {
			if val2.IntValue == 0 {
				return &NoValue, fmt.Errorf("can not divide by zero")
			}
			result.IntValue = val1.IntValue % val2.IntValue
		} else {
			return &NoValue, fmt.Errorf("mod needs integer arguments")
		}
	case TOK_AND:
		result.IntValue = val1.IntValue & val2.IntValue
	case TOK_AND_NOT:
		result.IntValue = val1.IntValue &^ val2.IntValue
	case TOK_OR:
		result.IntValue = val1.IntValue | val2.IntValue
	case TOK_XOR:
		result.IntValue = val1.IntValue ^ val2.IntValue
	case TOK_SHL:
		if val2.IntValue > 64 {
			return nil, fmt.Errorf("can not shift by more than 64 bits")
		}
		result.IntValue = val1.IntValue << val2.IntValue
	case TOK_SHR:
		if val2.IntValue > 64 {
			return nil, fmt.Errorf("can not shift by more than 64 bits")
		}
		result.IntValue = val1.IntValue >> val2.IntValue
	case TOK_LOG_OR:
		result.Typ = &BoolType
		result.BoolValue = val1.BoolValue || val2.BoolValue
	case TOK_LOG_AND:
		result.Typ = &BoolType
		result.BoolValue = val1.BoolValue && val2.BoolValue
	case TOK_EQ:
		result.Typ = &BoolType
		result.BoolValue = math.Abs(val1.FloatValue-val2.FloatValue)/max(val1.FloatValue, val2.FloatValue, 1e-30) < 1e-7
	case TOK_NE:
		result.Typ = &BoolType
		result.BoolValue = math.Abs(val1.FloatValue-val2.FloatValue)/max(val1.FloatValue, val2.FloatValue, 1e-30) >= 1e-7
	case TOK_LT:
		result.Typ = &BoolType
		result.BoolValue = val1.FloatValue < val2.FloatValue
	case TOK_LE:
		result.Typ = &BoolType
		result.BoolValue = val1.FloatValue <= val2.FloatValue
	case TOK_GT:
		result.Typ = &BoolType
		result.BoolValue = val1.FloatValue > val2.FloatValue
	case TOK_GE:
		result.Typ = &BoolType
		result.BoolValue = val1.FloatValue >= val2.FloatValue
	default:
		// Invalid operand
		return &NoValue, fmt.Errorf("invalid operation: %s", TokenNames[op])
	}
	return result, nil
}

// emitTosOpNos will generate code for the operation op on the two top entries on the stack.
func emitTosOpNos(op Token, val1, val2 *ValueDef) (*ValueDef, error) {
	EmitAssertTosInRax("Get TOS befor TosOpNos")
	if op.IsCompare() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			err := EmitCompareIntegers(op, false)
			return &ValueDef{Typ: &BoolType}, err
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
			err := EmitCompareFloats(op)
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
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			EmitIntegerOp(op)
			return val1, nil
		} else if val1.Typ.Pt == code.TYP_F64 && val2.Typ.Pt == code.TYP_F64 {
			emitF64Op(op, true, true)
			return val1, nil
		} else if val1.Typ.Pt == code.TYP_F32 && val2.Typ.Pt == code.TYP_F32 {
			emitF32Op(op, true, true)
			return val1, nil
		} else if val1.Typ.Pt == code.TYP_STRING && val2.Typ.Pt == code.TYP_STRING && op == TOK_PLUS {
			EmitConcat(val1.IsTempObj, val2.IsTempObj)
			return val1, nil
		} else if val1.Typ.Pt.IsInteger() && val2.Typ.Pt == code.TYP_F64 {
			emitF64Op(op, false, true)
			return val2, nil
		} else if val1.Typ.Pt.IsInteger() && val2.Typ.Pt == code.TYP_F32 {
			emitF32Op(op, false, true)
			return val2, nil
		} else if val1.Typ.Pt == code.TYP_F64 && val2.Typ.Pt.IsInteger() {
			emitF64Op(op, false, true)
			return val1, nil
		} else if val1.Typ.Pt == code.TYP_F32 && val2.Typ.Pt.IsInteger() {
			emitF32Op(op, false, true)
			return val1, nil
		} else {
			return nil, fmt.Errorf("invalid combination of operands to '%s'", TokenNames[op])
		}
	}
	return &NoValue, fmt.Errorf("operation %s not implemented", op.Name())
}

// generateTosOpConst will evaluate Top Of Stack with a constant. The constant is found in val2
func generateTosOpConst(op Token, val1 *ValueDef, val2 *ValueDef) (*ValueDef, error) {
	var err error
	if val1.IsConst {
		op = Inverse(op)
	}
	if op.IsCompare() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			err = EmitCompareIntConst(op, val2.IntValue, false)
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
			EmitAssertTosInRax("Get TOS before compare float const")
			err = EmitCompareFloatConst(op, val2.F64LitNo)
		} else if val1.Typ.Pt == code.TYP_STRING && val2.Typ.Pt == code.TYP_STRING {
			err = EmitCompareStrToLit(op, val2.StringValue, val2.StringLitNo, val1.IsTempObj)
		} else if val1.Typ.Pt == code.TYP_BOOL && val2.Typ.Pt == code.TYP_BOOL {
			err = EmitCompareIntConst(op, val2.IntValue, false)
		} else {
			err = fmt.Errorf("unknown type combination for compare")
		}
		return &ValueDef{Typ: &BoolType}, err
	} else if op.IsAritmetic() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			x := val2.IntValue + val1.IntValue
			if x > 0x7FFFFFFF || x < -0x7FFFFFFF {
				// err = fmt.Errorf("invalid integer combination for arithmetic")
				EmitPushConst(x, "")
				_, err2 := emitTosOpNos(op, val2, val2)
				return &ValueDef{Typ: val2.Typ}, err2
			} else {
				err = EmitOpIntConst(op, x, "TosOpConst")
			}
		} else if val1.Typ.Pt.IsNumber() && val2.Typ.Pt.IsNumber() {
			// FloatLitNo is in either val1 or val2. The other is allways zero
			if val1.Typ.Pt == code.TYP_F64 {
				floatLitNo := AddF64Lit(val1.FloatValue + val2.FloatValue)
				EmitOpF64Const(op, floatLitNo)
			} else if val1.Typ.Pt == code.TYP_F32 {
				floatLitNo := AddF32Lit(float32(val1.FloatValue + val2.FloatValue))
				EmitOpF32Const(op, floatLitNo)
			}
			return &ValueDef{Typ: val1.Typ}, nil
		} else {
			err = fmt.Errorf("unknown type combination for '%s'", op.Name())
		}
		return &ValueDef{Typ: val1.Typ}, err
	}
	return &NoValue, fmt.Errorf("could not perform %s on types %s and %s", op.Name(), val1.Typ.Name(), val2.Typ.Name())
}

// emitF64Op will generate a stack operation on the top two stack entries
func emitF64Op(op Token, op1float bool, op2float bool) {
	EmitAssertTosInRax("Get TOS before FloatOp")
	if op2float {
		emit("movq", xmm(2), "rax", "EmitFloatOp move tos in rax to xmm2")
	} else {
		emit("cvtsi2sd", xmm(2), "rax", "convert integer into xmm2")
	}
	emit("pop", "rax", "", "EmitFloatOp pop nos"+Sp(-1))
	if op1float {
		emit("movq", xmm(1), "rax", "EmitFloatOp mov nos to xmm1")
	} else {
		emit("cvtsi2sd", xmm(1), "rax", "convert integer into xmm1")
	}
	doF64Op(op)
}

// emitF32Op will generate a stack operation on the top two stack entries
func emitF32Op(op Token, op1float bool, op2float bool) {
	EmitAssertTosInRax("Get TOS before FloatOp")
	if op2float {
		emit("movd", xmm(2), "eax", "EmitFloatOp move tos in rax to xmm2")
	} else {
		emit("cvtsi2sd", xmm(2), "eax", "convert integer into xmm2")
	}
	emit("pop", "rax", "", "EmitFloatOp pop nos"+Sp(-1))
	if op1float {
		emit("movd", xmm(1), "eax", "EmitFloatOp mov nos to xmm1")
	} else {
		emit("cvtsi2sd", xmm(1), "eax", "convert integer into xmm1")
	}
	doF32Op(op)
}

func doF64Op(op Token) {
	if op == TOK_PLUS {
		emit("addsd", xmm(1), xmm(2), "Add tos to nos")
	} else if op == TOK_MINUS {
		emit("subsd", xmm(1), xmm(2), "Subtract nos from tos")
	} else if op == TOK_MULT {
		emit("mulsd", xmm(1), xmm(2), "Multiply nos by tos")
	} else if op == TOK_DIV {
		emit("divsd", xmm(1), xmm(2), "Divide tos by nos")
	} else if op == TOK_INV_DIV {
		emit("divsd", xmm(2), xmm(1), "Divide nos by tos")
		emit("movq", xmm(1), xmm(2), "")
	} else {
		panic("EmitFloatOp not implemented for " + op.Name())
	}
	code.SetAx()
	emit("movq", "rax", xmm(1), "Move float result into rax")
}

func doF32Op(op Token) {
	if op == TOK_PLUS {
		emit("addss", xmm(1), xmm(2), "Add tos to nos")
	} else if op == TOK_MINUS {
		emit("subss", xmm(1), xmm(2), "Subtract nos from tos")
	} else if op == TOK_MULT {
		emit("mulss", xmm(1), xmm(2), "Multiply nos by tos")
	} else if op == TOK_DIV {
		emit("divss", xmm(1), xmm(2), "Divide tos by nos")
	} else if op == TOK_INV_DIV {
		emit("divss", xmm(2), xmm(1), "Divide nos by tos")
		emit("movq", xmm(1), xmm(2), "")
	} else {
		panic("EmitFloatOp not implemented for " + op.Name())
	}
	code.SetAx()
	emit("movd", "eax", xmm(1), "Move float result into rax")
}
