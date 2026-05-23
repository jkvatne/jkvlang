package main

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"

	"github.com/jkvatne/jkv/code"
)

// GenerateOp will handle the infix operations +,-,*,/,%,|,&,^,<,>,<=,>=,==,!=
// Integer operands are promoted to the smallest size that can accomondate both.
// F.ex. I16 op U16 results in an I32
// There are 4 different cases: const op const, tos op const, const op tos, tos op nos
func GenerateOp(op Token, val1 *ValueDef, val2 *ValueDef) (*ValueDef, error) {
	// Convert int values to float in case of mixed types.
	if val1.Typ.Pt != TYP_F64 && val1.Typ.Pt != TYP_F32 {
		val1.FloatValue = float64(val1.IntValue)
	}
	if val2.Typ.Pt != TYP_F64 && val2.Typ.Pt != TYP_F32 {
		val2.FloatValue = float64(val2.IntValue)
	}
	// For user defined types, both must be identical, or one operand must be a basic type.
	if !val1.Typ.Basic && !val2.Typ.Basic && val1.Typ != val2.Typ {
		return &NoValue, fmt.Errorf("Operation on incompatible types %s and %s", val1.Typ.Pt.Name(), val2.Typ.Pt.Name())
	}
	if val1.IsConst && val2.IsConst {
		// If both operands are constant. Evaluate at compile time.
		return generateConstOpConst(op, val1, val2)
	} else if val1.IsConst || val2.IsConst {
		EmitAssertTosInRax("Get TOS")
		return generateTosOpConst(op, val1, val2)
	} else {
		EmitAssertTosInRax("Get TOS")
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
	case TOK_OR:
		result.IntValue = val1.IntValue | val2.IntValue
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

func emitTosOpNos2(op Token, val1, val2 *ValueDef) (*ValueDef, error) {
	EmitPopBx("Pop arg 2 into RBX")
	if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
	} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
	}
	return &NoValue, nil
}

// generateTosOpConst2 uses inverted op if first argument is a const
func generateTosOpConst2(op Token, val1 *ValueDef, val2 *ValueDef) (*ValueDef, error) {
	return &NoValue, nil
}

// emitTosOpNos will generate code for the operation op on the two top entries on the stack.
func emitTosOpNos(op Token, val1, val2 *ValueDef) (*ValueDef, error) {
	EmitAssertTosInRax("Get TOS")
	if op.IsCompare() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			err := emitCompareIntegers(op, false)
			return &ValueDef{Typ: &BoolType}, err
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
			err := emitCompareFloats(op)
			return &ValueDef{Typ: &BoolType}, err
		} else if val1.Typ.Pt == TYP_STRING && val2.Typ.Pt == TYP_STRING {
			if op == TOK_EQ {
				EmitCompareStringsEq(val1.IsTempObj, val2.IsTempObj)
				return &ValueDef{Typ: &BoolType}, nil
			} else if op == TOK_NE {
				EmitCompareStringsNe(val1.IsTempObj, val2.IsTempObj)
				return &ValueDef{Typ: &BoolType}, nil
			}
		}
	} else if op.IsAritmetic() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			emitIntegerOp(op)
			return val1, nil
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
			emitFloatOp(op)
			return val1, nil
		} else if val1.Typ.Pt == TYP_STRING && val2.Typ.Pt == TYP_STRING {
			if op == TOK_PLUS {
				EmitConcat(val1.IsTempObj, val2.IsTempObj)
				return val1, nil
			}
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
			err = emitCompareIntConst(op, val2.IntValue, false)
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
			err = emitCompareFloatConst(op, val2.FloatLitNo)
		} else if val1.Typ.Pt == TYP_STRING && val2.Typ.Pt == TYP_STRING {
			err = EmitCompareStrToLit(op, val2.StringValue, val2.StringLitNo, val1.IsTempObj)
		} else {
			err = fmt.Errorf("Unknown type combination for compare")
		}
		return &ValueDef{Typ: &BoolType}, err
	} else if op.IsAritmetic() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			err = emitOpIntConst(op, val2.IntValue+val1.IntValue, "")
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() && val1.Typ.Pt.Name() == val2.Typ.Pt.Name() {
			// FloatLitNo is in either val1 or val2. The other is allways zero
			emitOpFloatConst(op, val2.FloatLitNo+val1.FloatLitNo)
			return &ValueDef{Typ: val1.Typ}, nil
		}
		return &ValueDef{Typ: val1.Typ}, err
	}
	return &NoValue, fmt.Errorf("could not perform %s on types %s and %s", op.Name(), val1.Typ.Name(), val2.Typ.Name())
}

// emitCompareFloatConst compares float in rax with float constant
func emitCompareFloatConst(op Token, litNo int) (err error) {
	EmitAssertTosInRax("Get TOS")
	emit("movq", xmm(1), "rax", "")
	emit("mov", "rax", "[flt"+strconv.Itoa(litNo)+"]", "Load float value from literal")
	emit("movq", xmm(2), "rax", "")
	emit("ucomisd", xmm(1), xmm(2), "Compare two floats "+op.Name())
	err = EmitJumpCond(op, true)
	return err
}

// emitCompareFloats compares two floats.
func emitCompareFloats(op Token) (err error) {
	EmitAssertTosInRax("Get TOS")
	emit("movq", xmm(2), "rax", "")
	EmitPopAx("")
	emit("movq", xmm(1), "rax", "")
	emit("ucomisd", xmm(1), xmm(2), "Compare two floats "+op.Name())
	err = EmitJumpCond(op, true)
	return err
}

// emitCompareIntegers will compare the top two stack entries
func emitCompareIntegers(op Token, unsigned bool) (err error) {
	EmitPopBx("Pop next on stack into RBX")
	emit("cmp", "rax", "rbx", "Compare and set flags")
	return EmitJumpCond(op, unsigned)
}

// emitCompareIntConst will compare top of stack with a constant
func emitCompareIntConst(op Token, value int64, unsigned bool) error {
	sval := strconv.FormatInt(value, 10)
	emit("cmp", "rax", sval, "Compare and set flags")
	return EmitJumpCond(op, unsigned)
}

// emitIntegerOp will generate a stack operation on the top two stack entries, like add or sub
// The stack pointer will be incremented (pop), and the result will now be on top of the stack (AX)
func emitIntegerOp(op Token) {
	EmitPopBx("")
	if op == TOK_DIV {
		emit("xchg", "rax", "rbx", "")
		emit("cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit("idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit("xchg", "rax", "rbx", "")
		emit("cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit("idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit("mov", "rax", "rdx", "Move reminder to AX (top of stack)")
	} else {
		instruction := TokenOp[op]
		if instruction == "" {
			slog.Error("EmitIntegerOp called with invalid token", "op", op.Name())
		}
		if op == TOK_MULT {
			emit("mul", "rbx", "", "Integer op mul")
		} else {
			emit(instruction, "rax", "rbx", "Integer op")
		}
	}
}

// emitOpIntConst will evaluate tos=tos op <constant>
// It uses 64bit integer values on the 64 bit rax register
func emitOpIntConst(op Token, value int64, comment string) error {
	sval := strconv.FormatInt(value, 10)
	if op == TOK_DIV {
		emit("cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit("mov", "rbx", sval, "Get divisor from stack into RBX")
		emit("idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_INV_DIV {
		emit("mov", "rbx", sval, "Get divisor from stack into RBX")
		emit("xchg", "rax", "rbx", "")
		emit("cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit("idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit("mov", "rbx", sval, "RBX=constant divisor")
		emit("cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit("idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit("mov", "rax", "rdx", "Move reminder to AX (top of stack)")
	} else if op == TOK_INV_MOD {
		emit("mov", "rbx", sval, "RBX=constant divisor")
		emit("xchg", "rax", "rbx", "")
		emit("cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit("idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit("mov", "rax", "rdx", "Move reminder to AX (top of stack)")
	} else if op == TOK_ASSIGN {
		emit("mov", "rax", sval, "Assign OpIntConst")
	} else {
		instr := TokenOp[op]
		if instr == "" {
			return fmt.Errorf("invalid operation %s", op.Name())
		}
		emit(instr, "rax", strconv.FormatInt(value, 10), comment)
	}
	return nil
}

func emitOpFloatConst(op Token, litNo int) {
	EmitAssertTosInRax("Get TOS")
	emit("movq", xmm(1), "rax", "EmitOpFloatConst move tos in rax to xmm1")
	emit("mov", "rax", "[flt"+strconv.Itoa(litNo)+"]", "emitOpFloatConst")
	emit("movq", xmm(2), "rax", "EmitOpFloatConst mov nos to xmm2")
	doFloatOp(op)
}

// emitFloatOp will generate a stack operation on the top two stack entries
func emitFloatOp(op Token) {
	EmitAssertTosInRax("Get TOS")
	emit("movq", xmm(2), "rax", "EmitFloatOp move tos in rax to xmm2")
	emit("pop", "rax", "", "EmitFloatOp pop nos")
	code.LocalSp--
	emit("movq", xmm(1), "rax", "EmitFloatOp mov nos to xmm1")
	doFloatOp(op)
}

func doFloatOp(op Token) {
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
	emit("movq", "rax", xmm(1), "Move float result into rax")
	code.RaxIsTOS = true
}
