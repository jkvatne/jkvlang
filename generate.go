package main

import (
	"fmt"
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
func EmitTosOpNos(s *State, op Token, val1, val2 *ValueDef) (*ValueDef, error) {
	if op.IsCompare() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			err := EmitCompareIntegers(s, op, false)
			return &ValueDef{Typ: &BoolType}, err
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
			EmitCompareFloats(s, op)
			return &ValueDef{Typ: &BoolType}, nil
		} else if val1.Typ.Pt == TYP_STRING && val2.Typ.Pt == TYP_STRING {
			if op == TOK_EQ {
				EmitCompareStringsEq(s)
				return &ValueDef{Typ: &BoolType}, nil
			} else if op == TOK_NE {
				EmitCompareStringsNe(s)
				return &ValueDef{Typ: &BoolType}, nil
			}
		}
	} else if op.IsAritmetic() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			EmitIntegerOp(s, op)
			return val1, nil
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
			EmitF64Op(s, op)
			return val1, nil
		} else if val1.Typ.Pt == TYP_STRING && val2.Typ.Pt == TYP_STRING {
			if op == TOK_PLUS {
				EmitConcat(s)
				return val1, nil
			}
		}
	}
	return &NoValue, fmt.Errorf("operation %s not implemented", op.Name())
}

// GenerateTosOpConst will evaluate Top Of Stack with a constant. The constant is found in val2
func GenerateTosOpConst(s *State, op Token, val1 *ValueDef, val2 *ValueDef) (*ValueDef, error) {
	var err error
	if op.IsCompare() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			err = EmitCompareIntConst(s, op, val2.IntValue, false)
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() {
			// First push constant into xmm<sp+1>
			emit(s, "movq", xmm(s.XmmSp), "[flt"+strconv.Itoa(val2.FloatLitNo)+"]", "Load NOS into xmm<sp>")
			s.XmmSp++
			err = EmitCompareFloats(s, op)
		} else if val1.Typ.Pt == TYP_STRING && val2.Typ.Pt == TYP_STRING {
			err = EmitCompareStrings(s, op, val2.StringValue, val2.StringLitNo)
		} else {
			err = fmt.Errorf("Unknown type combination for compare")
		}
		return &ValueDef{Typ: &BoolType}, err
	} else if op.IsAritmetic() {
		if val1.Typ.Pt.IsInteger() && val2.Typ.Pt.IsInteger() {
			err = EmitOpIntConst(s, op, val2.IntValue, "")
		} else if val1.Typ.Pt.IsFloat() && val2.Typ.Pt.IsFloat() && val1.Typ.Name() == val2.Typ.Name() {
			// Move constant value to sp+1
			if s.XmmSp > 6 {
				panic("Floating point stack overflow")
			}
			EmitPushFloat(s, val2.FloatLitNo)
			EmitF64Op(s, op)
			return &ValueDef{Typ: val1.Typ}, nil
		}
		return &ValueDef{Typ: val1.Typ}, err
	}
	return &NoValue, fmt.Errorf("could not perform %s on types %s and %s", op.Name(), val1.Typ.Name(), val2.Typ.Name())
}

func GenertateAssignment(s *State, op Token, lvalue *VarDef, value *ValueDef) (err error) {
	// Set lvalue type if not already set. Needed for new variables.
	if lvalue.Typ == nil && op == TOK_ASSIGN {
		lvalue.SetType(value.Typ)
		// old := VarDefs[lvalue.Name].Offset
		// fmt.Printf("Assign value sp=%d; offset=%d; old offset=%d\n", s.localSp, lvalue.Offset, old)
		VarDefs[lvalue.Name].Value.Offset = EmitAllocLocalVar(s, "Allocate local variable "+lvalue.Name)
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
				err = EmitOpAssignString(s, lvalue.Offset(), value.StringLitNo)
			} else if lvalue.Typ.Pt.IsInteger() {
				if lvalue.Name == "err" {
					emit(s, "mov", "r15", strconv.Itoa(int(value.IntValue)), "Set tos to r15 = error value")
				} else {
					err = EmitOpAssign(s, op, lvalue.Offset(), lvalue.Typ.Pt.Size(), value.IntValue, "")
				}
			} else if lvalue.Typ.Pt == TYP_F64 {
				err = EmitOpAssignFloat(s, op, lvalue.Offset(), value.FloatLitNo, "")
			} else {
				panic("Unimplemented assignment")
			}

			if err != nil {
				return err
			}

		} else {
			return fmt.Errorf("cannot assign to variable \"%s\"", lvalue.Name)
		}
	} else if value.Typ.Pt.IsInteger() {
		// The value is on the top of the stack (rax). Save it to the lvalue.
		instr := TokenOp[op]
		EmitStore(s, instr, lvalue.Typ.Pt.Size(), lvalue.Offset(), "Assign to "+lvalue.Name)
	} else if value.Typ.Pt == TYP_F64 {
		EmitStoreF64(s, lvalue.Offset(), "Assign F64 to "+lvalue.Name)
	} else if value.Typ.Pt == TYP_STRING {
		instr := TokenOp[op]
		EmitStore(s, instr, lvalue.Typ.Pt.Size(), lvalue.Offset(), "Assign to "+lvalue.Name)
	} else {
		return fmt.Errorf("cannot assign to variable \"%s\"", lvalue.Name)
	}
	return nil
}
