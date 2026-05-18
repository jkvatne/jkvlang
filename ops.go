package main

import (
	"fmt"
	"log/slog"
	"strconv"
)

// GenerateOp will handle the infix operations +,-,*,/,%,|,&,^,<,>,<=,>=,==,!=
// Integer operands are promoted to the smallest size that can accomondate both.
// F.ex. I16 op U16 results in an I32
func GenerateOp(s *State, op Token, val1 *ValueDef, val2 *ValueDef) (*ValueDef, error) {
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
	if val1.HasValue && val2.HasValue {
		// If both operands are constant. Evaluate at compile time.
		return EmitConstOpConst(op, val1, val2)
	} else if val1.HasValue {
		// The left side is a constant. Do the inverse operation
		return GenerateTosOpConst(s, Inverse(op), val2, val1)
	} else if val2.HasValue {
		// The right side is a constant. Do the operation on top of stack
		return GenerateTosOpConst(s, op, val1, val2)
	} else {
		return EmitTosOpNos(s, op, val1, val2)
	}
}

// EmitTosOpNos will generate code for the operation op on the two top entries on the stack.
func EmitTosOpNos(s *State, op Token, val1, val2 *ValueDef) (*ValueDef, error) {
	if !s.RaxIsTOS {
		emit(s, "pop", "rax", "", "EmitTosOpNos, get TOS to rax")
		s.localSp--
		s.RaxIsTOS = true
	}
	if op.IsCompare() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			err := EmitCompareIntegers(s, op, false)
			return &ValueDef{Typ: &BoolType}, err
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
			EmitCompareFloats(s, op)
			return &ValueDef{Typ: &BoolType}, nil
		} else if val1.Typ.Pt == TYP_STRING && val2.Typ.Pt == TYP_STRING {
			if op == TOK_EQ {
				EmitCompareStringsEq(s, val1.IsTempObj, val2.IsTempObj)
				return &ValueDef{Typ: &BoolType}, nil
			} else if op == TOK_NE {
				EmitCompareStringsNe(s, val1.IsTempObj, val2.IsTempObj)
				return &ValueDef{Typ: &BoolType}, nil
			}
		}
	} else if op.IsAritmetic() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			EmitIntegerOp(s, op)
			return val1, nil
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
			EmitFloatOp(s, op)
			return val1, nil
		} else if val1.Typ.Pt == TYP_STRING && val2.Typ.Pt == TYP_STRING {
			if op == TOK_PLUS {
				EmitConcat(s, val1.IsTempObj, val2.IsTempObj)
				return val1, nil
			}
		}
	}
	return &NoValue, fmt.Errorf("operation %s not implemented", op.Name())
}

// GenerateTosOpConst will evaluate Top Of Stack with a constant. The constant is found in val2
func GenerateTosOpConst(s *State, op Token, val1 *ValueDef, val2 *ValueDef) (*ValueDef, error) {
	var err error
	if !s.RaxIsTOS {
		emit(s, "pop", "rax", "", "Pop value for TosOpConst")
		s.localSp--
		s.RaxIsTOS = true
	}
	if op.IsCompare() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			err = EmitCompareIntConst(s, op, val2.IntValue, false)
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
			err = EmitCompareFloatConst(s, op, val2.FloatLitNo)
		} else if val1.Typ.Pt == TYP_STRING && val2.Typ.Pt == TYP_STRING {
			err = EmitCompareStrToLit(s, op, val2.StringValue, val2.StringLitNo, val1.IsTempObj)
		} else {
			err = fmt.Errorf("Unknown type combination for compare")
		}
		return &ValueDef{Typ: &BoolType}, err
	} else if op.IsAritmetic() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			err = EmitOpIntConst(s, op, val2.IntValue, "")
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() && val1.Typ.Name() == val2.Typ.Name() {
			EmitOpFloatConst(s, op, val2.FloatLitNo)
			return &ValueDef{Typ: val1.Typ}, nil
		}
		return &ValueDef{Typ: val1.Typ}, err
	}
	return &NoValue, fmt.Errorf("could not perform %s on types %s and %s", op.Name(), val1.Typ.Name(), val2.Typ.Name())
}

// EmitCompareFloatConst compares float in rax with float constant
func EmitCompareFloatConst(s *State, op Token, litNo int) (err error) {
	emit(s, "movq", xmm(1), "rax", "")
	emit(s, "mov", "rax", "[flt"+strconv.Itoa(litNo)+"]", "Load float value from literal")
	emit(s, "movq", xmm(2), "rax", "")
	emit(s, "ucomisd", xmm(1), xmm(2), "Compare two floats "+op.Name())
	err = EmitJumpCond(s, op, true)
	return err
}

// EmitCompareFloats compares two floats.
func EmitCompareFloats(s *State, op Token) (err error) {
	if !s.RaxIsTOS {
		emit(s, "pop", "rax", "", "")
		s.localSp--
	}
	emit(s, "movq", xmm(2), "rax", "")
	emit(s, "pop", "rax", "", "")
	s.localSp--
	emit(s, "movq", xmm(1), "rax", "")
	emit(s, "ucomisd", xmm(1), xmm(2), "Compare two floats "+op.Name())
	err = EmitJumpCond(s, op, true)
	return err
}

// EmitCompareIntegers will compare the top two stack entries
func EmitCompareIntegers(s *State, op Token, unsigned bool) (err error) {
	emit(s, "pop", "rbx", "", "Pop next on stack into RBX")
	s.localSp--
	emit(s, "cmp", "rax", "rbx", "Compare and set flags")
	return EmitJumpCond(s, op, unsigned)
}

// EmitCompareIntConst will compare top of stack with a constant
func EmitCompareIntConst(s *State, op Token, value int64, unsigned bool) error {
	sval := strconv.FormatInt(value, 10)
	emit(s, "cmp", "rax", sval, "Compare and set flags")
	return EmitJumpCond(s, op, unsigned)
}

// EmitIntegerOp will generate a stack operation on the top two stack entries, like add or sub
// The stack pointer will be incremented (pop), and the result will now be on top of the stack (AX)
func EmitIntegerOp(s *State, op Token) {

	if op == TOK_DIV {
		emit(s, "xchg", "rbx", "rax", "Exchange RAX and RBX since we calculate NOS/TOS")
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "pop", "rbx", "", "Get divisor from stack into RBX")
		s.localSp--
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "pop", "rbx", "", "Get divisor from stack into RBX")
		s.localSp--
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit(s, "mov", "rax", "rdx", "Move reminder to AX (top of stack)")
	} else {
		if !s.RaxIsTOS {
			emit(s, "pop", "rax", "", "Get op 1 from stack")
			s.localSp--
		}
		emit(s, "pop", "rbx", "", "Get op 2 from stack")
		s.localSp--
		instruction := TokenOp[op]
		if instruction == "" {
			slog.Error("EmitIntegerOp called with invalid token", "op", op.Name())
		}
		if op == TOK_MULT {
			emit(s, "mul", "rbx", "", "Integer op mul")
		} else {
			emit(s, instruction, "rax", "rbx", "Integer op")
		}
	}
}

// EmitOpConst will evaluate tos=tos op <constant>
// It uses 64bit integer values on the 64 bit rax register
func EmitOpIntConst(s *State, op Token, value int64, comment string) error {
	sval := strconv.FormatInt(value, 10)
	if op == TOK_DIV {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "mov", "rbx", sval, "Get divisor from stack into RBX")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
	} else if op == TOK_MOD {
		emit(s, "cqo", "", "", "Sign-extend dividend in RAX into RDX:RAX")
		emit(s, "mov", "rbx", sval, "RBX=constant divisor")
		emit(s, "idiv", "rbx", "", "RAX = RDX:RAX/RBX; RDX=Reminder")
		emit(s, "mov", "rax", "rdx", "Move reminder to AX (top of stack)")
	} else if op == TOK_ASSIGN {
		emit(s, "mov", "rax", sval, "Assign OpIntConst")
	} else {
		instr := TokenOp[op]
		if instr == "" {
			return fmt.Errorf("invalid operation %s", op.Name())
		}
		emit(s, instr, "rax", strconv.FormatInt(value, 10), comment)
	}
	return nil
}

func EmitOpFloatConst(s *State, op Token, litNo int) {
	if !s.RaxIsTOS {
		emit(s, "pop", "rax", "", "EmitOpFloatConst, tos is not rax")
		s.localSp--
	}
	emit(s, "movq", xmm(1), "rax", "EmitOpFloatConst move tos in rax to xmm1")
	emit(s, "mov", "rax", "[flt"+strconv.Itoa(litNo)+"]", "EmitPushFloatLit()")
	emit(s, "movq", xmm(2), "rax", "EmitOpFloatConst mov nos to xmm2")
	doFloatOp(s, op)
}

// EmitFloatOp will generate a stack operation on the top two stack entries
func EmitFloatOp(s *State, op Token) {
	if !s.RaxIsTOS {
		emit(s, "pop", "rax", "", "EmitFloatOp, tos is not rax")
		s.localSp--
	}
	emit(s, "movq", xmm(2), "rax", "EmitFloatOp move tos in rax to xmm2")
	emit(s, "pop", "rax", "", "EmitFloatOp pop nos")
	s.localSp--
	emit(s, "movq", xmm(1), "rax", "EmitFloatOp mov nos to xmm1")
	doFloatOp(s, op)
}

func doFloatOp(s *State, op Token) {
	if op == TOK_PLUS {
		emit(s, "addsd", xmm(1), xmm(2), "Add tos to nos")
	} else if op == TOK_MINUS {
		emit(s, "subsd", xmm(1), xmm(2), "Subtract nos from tos")
	} else if op == TOK_MULT {
		emit(s, "mulsd", xmm(1), xmm(2), "Multiply nos by tos")
	} else if op == TOK_DIV {
		emit(s, "divsd", xmm(1), xmm(2), "Divide tos by nos")
	} else if op == TOK_INV_DIV {
		emit(s, "divsd", xmm(2), xmm(1), "Divide nos by tos")
		emit(s, "movq", xmm(1), xmm(2), "")
	} else {
		panic("EmitFloatOp not implemented for " + op.Name())
	}
	emit(s, "movq", "rax", xmm(1), "Move float result into rax")
	s.RaxIsTOS = true
}
