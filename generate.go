package main

import (
	"fmt"
	"strconv"
)

func GenerateAssignment(s *State, op Token, lvalue *VarDef, value *ValueDef) (err error) {
	// Set lvalue type if not already set. Needed for new variables.
	if lvalue.Typ == nil && op == TOK_ASSIGN {
		lvalue.SetType(value.Typ)
		// VarDefs[lvalue.Name].Value.Offset = EmitAllocLocalVar(s, "Allocate local variable "+lvalue.Name)
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
				if value.FloatLitNo == 0 {
					value.FloatLitNo = AddFloatLiteral(value.FloatValue)
					err = EmitOpAssignFloat(s, op, lvalue.Offset(), value.FloatLitNo, "")
				} else {
					err = EmitOpAssignFloat(s, op, lvalue.Offset(), value.FloatLitNo, "")
				}
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
		if !s.RaxIsTOS {
			EmitPopAx(s, "Pop TOS into rax before assignment")
		}
		EmitStore(s, instr, lvalue.Typ.Pt.Size(), lvalue.Offset(), "Assign to "+lvalue.Name)
		lvalue.MustFree = true
	} else {
		return fmt.Errorf("cannot assign to variable \"%s\"", lvalue.Name)
	}
	return nil
}
