package main

import (
	"fmt"
	"math"
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
		return constOpConst(op, val1, val2)
	} else if val1.HasValue {
		// The left side is a constant. Do the inverse operation
		return tosOpConst(s, Inverse(op), val2, val1)
	} else if val2.HasValue {
		// The right side is a constant. Do the operation on top of stack
		return tosOpConst(s, op, val1, val2)
	} else {
		return tosOpNos(s, op, val1, val2)
	}
}

func Inverse(op Token) Token {
	switch op {
	case TOK_LT:
		return TOK_GT
	case TOK_LE:
		return TOK_GE
	case TOK_GT:
		return TOK_LT
	case TOK_GE:
		return TOK_LE
	case TOK_MINUS:
		return TOK_INV_MINUS
	case TOK_DIV:
		return TOK_INV_DIV
	default:
		return op
	}
}

// constOpConst will calculate the result of the operation on the two constant values
// and return the constant result.
func constOpConst(op Token, val1 *ValueDef, val2 *ValueDef) (*ValueDef, error) {
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
	return &result, nil
}

// tosOpConst will evaluate Top Of Stack with a constant. The constant is found in val2
func tosOpConst(s *State, op Token, val1 *ValueDef, val2 *ValueDef) (*ValueDef, error) {
	if op.IsCompare() && val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
		err := EmitCompareIntConst(s, op, val2.IntValue)
		return &ValueDef{Typ: &BoolType}, err
	} else if op.IsAritmetic() && val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
		err := EmitOpIntConst(s, op, val2.IntValue, "")
		return &ValueDef{Typ: val1.Typ}, err
	} else if op.IsAritmetic() && val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() && val1.Typ.Name() == val2.Typ.Name() {
		err := EmitOpFloatConst(s, op, val2.FloatValue, "")
		return &ValueDef{Typ: val1.Typ}, err
	} else if op == TOK_EQ && val1.Typ.Pt == TYP_STRING && val2.Typ.Pt == TYP_STRING {
		// The pointer to the first string (val1) is found in AX. Compare it to the known constant in val2
		// First check lengths
		emit(s, "cmp", "word [rax]", strconv.Itoa(len(val2.StringValue)), "Compare string lengths")
		lbl := NewLabel(s)
		emit(s, "mov", "rbx", "0", "Initialize result to false")
		emit(s, "jne", LabelName(lbl), "", "If not equal, jump to unequal end")
		emit(s, "mov", "rsi", "str"+strconv.Itoa(val2.StringLitNo), "")
		emit(s, "mov", "rdi", "rax", "")
		emit(s, "repe", "cmpsb", "", "")
		emit(s, "jne", LabelName(lbl), "", "If not equal, jump to unequal end")
		emit(s, "mov", "rbx", "1", "Strings was equal, set rax=true")
		EmitLabel(s, lbl, "")
		emit(s, "mov", "rax", "rbx", "Result to TOS (rax)")
		return &ValueDef{Typ: &BoolType}, nil
	}
	return &NoValue, fmt.Errorf("could not perform %s on types %s and %s", op.Name(), val1.Typ.Name(), val2.Typ.Name())
}

func tosOpNos(s *State, op Token, val1, val2 *ValueDef) (*ValueDef, error) {
	if op.IsCompare() && val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
		err := EmitCompareIntegers(s, op)
		return &ValueDef{Typ: &BoolType}, err
	} else if op.IsAritmetic() && val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
		EmitIntegerOp(s, op)
		return val1, nil
	} else if op.IsAritmetic() && val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
		EmitFloatOp(s, op)
		return val1, nil
	} else if op.IsCompare() && val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
		EmitCompareFloats(s, op)
		return &ValueDef{Typ: &BoolType}, nil
	} else if op == TOK_PLUS && val1.Typ.Pt == TYP_STRING && val2.Typ.Pt == TYP_STRING {
		EmitConcat(s)
		return val1, nil
	} else if op == TOK_EQ && val1.Typ.Pt == TYP_STRING && val2.Typ.Pt == TYP_STRING {
		lbl := NewLabel(s)
		emit(s, "mov", "rbx", "0", "Initialize result to false")
		emit(s, "mov", "rdi", "rax", "Save tos")
		emit(s, "mov", "rsi", "[rsp]", "Get nos")
		emit(s, "mov", "rcx", "4", "Compare first 4 bytes")
		emit(s, "repe", "cmpsb", "", "")
		emit(s, "jne", LabelName(lbl), "", "If lengths not equal, jump to unequal end")
		emit(s, "mov", "eax", "[rsp]", "Get nos prt")
		emit(s, "mov", "ecx", "[rax]", "Get nos length")
		emit(s, "add", "rsi", "4", "Start of string 1")
		emit(s, "add", "rdi", "4", "Start of string 2")
		emit(s, "repe", "cmpsb", "", "")
		emit(s, "jne", LabelName(lbl), "", "If not equal, jump to unequal end")
		emit(s, "mov", "rbx", "1", "Strings was equal, set rax=true")
		EmitLabel(s, lbl, "unequal")
		emit(s, "pop", "rax", "", "Remove NOS")
		s.localSp--
		emit(s, "mov", "rax", "rbx", "Result to TOS (rax)")
		return &ValueDef{Typ: &BoolType}, nil
	}
	return &NoValue, fmt.Errorf("operation %s not implemented", op.Name())
}

func GenertateAssignment(s *State, op Token, lvalue *VarDef, value *ValueDef) (err error) {
	// Set lvalue type if not already set. Needed for new variables.
	if lvalue.Typ == nil && op == TOK_ASSIGN {
		lvalue.SetType(value.Typ)
		// Local variables are at negative offset. The first on -8.
		EmitAllocLocalVar(s, lvalue.Size(), lvalue.Name)
		// fmt.Printf("%d %d\n", -s.localSp*8, lvalue.Offset)
		VarDefs[lvalue.Name].Offset = -s.localSp * 8
	}
	if lvalue.Typ == nil {
		return fmt.Errorf("new variable not allowed before op-assignment")
	}
	// Check types to see if the value can be assigned to the lvalue
	if !CanAssign(lvalue.Typ.Pt, value.Typ.Pt) {
		return fmt.Errorf("assignment expected type %s but got %s",
			lvalue.Typ.Pt.Name(), value.Typ.Name())
	}
	// If the value is known (a compile time constant)
	if value.HasValue {
		if CanAssignConst(lvalue.Typ.Pt, value) {
			if lvalue.Typ.Pt == TYP_STRING {
				err = EmitOpAssignString(s, lvalue.Offset, value.StringLitNo)
			} else if lvalue.Typ.Pt.IsInteger() {
				err = EmitOpAssign(s, op, lvalue.Offset, lvalue.Typ.Pt.Size(), value.IntValue, "")
			} else {
				panic("Unimplemented assignment")
			}

			if err != nil {
				return err
			}

		} else {
			return fmt.Errorf("cannot assign to variable \"%s\"", lvalue.Name)
		}
	} else {
		// The value is on the top of the stack (rax). Save it to the lvalue.
		EmitStore(s, lvalue.Typ.Pt.Size(), lvalue.Offset, "Assign to "+lvalue.Name)
	}
	return nil
}
