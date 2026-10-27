package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jkvatne/jkv/code"
)

// TypeFromConstValue generates a type from a constant value.
// If the value is a integer constant and value is a new variable, we default
// to I32 instead of U8 for small constants.
func TypeFromConstValue(value *ValueDef) (*TypeDef, error) {
	if value.StringValue != "" {
		if strings.Contains(value.StringValue, ".") {
			return &F64Type, nil
		} else if value.StringValue[0] >= '0' && value.StringValue[0] <= '9' {
			x, err := strconv.ParseUint(value.StringValue, 0, 64)
			if err != nil {
				return nil, err
			}
			if x > 0x7FFFFFFFFFFFFFFF {
				return &U64Type, nil
			} else if x > 0xFFFFFFFF {
				return &I64Type, nil
			} else {
				return &I32Type, nil
			}
		} else {
			return &StringType, nil
		}
	} else if value.Typ.Pt == code.TYP_BOOL {
		return &BoolType, nil
	} else {
		return nil, fmt.Errorf("no type for variable")
	}
}

func UpdateType(value *ValueDef) error {
	if value.Typ == nil {
		var err error
		value.Typ, err = TypeFromConstValue(value)
		if err != nil {
			return fmt.Errorf("type conversion with unknown type")
		}
	}
	return nil
}

// GenerateAssignment generates code for generic assignments, including +=, -= etc.
// lvalue is the left side, and value the right side of the equal sign.
// The op is already verified to be one of the assignment operators.
func GenerateAssignment(op Token, lvalue *VarDef, value *ValueDef) (err error) {
	isNew := lvalue.Typ == nil
	// Set lvalue type if not already set. Needed for new variables.
	if lvalue.Typ == nil && value.Typ == nil && op == TOK_ASSIGN {
		value.Typ, err = TypeFromConstValue(value)
	}
	if lvalue.Typ == nil {
		lvalue.Typ = value.Typ
	}

	// Check types to see if the value can be assigned to the lvalue
	if !CanAssign(lvalue.Typ.Pt, value) {
		return fmt.Errorf(" could not assign const %s to type %s", value.StringValue, lvalue.Typ.Pt.Name())
	}

	// Check type of left and right side and handle indirect, local, const, var
	if lvalue.IsIndirect && value.HasValue() {
		// Assign constant to indirect variable
		return AssignIndirectConst(op, lvalue, value)
	} else if lvalue.IsIndirect && !value.HasValue() {
		// Assign evaluated value to indirect variable
		return AssignIndirectExpression(op, lvalue, value)
	} else if value.HasValue() {
		// Assign constant to local variable
		return AssignVariableConst(op, lvalue, value)
	}
	// Assign evaluated value to local variable
	err = AssignVariableExpression(op, lvalue, value, isNew)
	code.SetSp()
	return err
}

func StrToConst(s string) (float64, error) {
	if strings.Contains(s, "0x") {
		v, err := strconv.ParseUint(s, 0, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid integer value: %s", s)
		}
		return float64(v), nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value: %s", s)
	}
	// TODO: Check range
	return f, nil
}

func AssignVariableConst(op Token, lvalue *VarDef, value *ValueDef) error {
	if lvalue.Offset == 0 {
		return fmt.Errorf("local variable adr is zero")
	}
	if lvalue.Typ.Pt == code.TYP_STRING && value.Typ.Pt == code.TYP_STRING {
		if op == TOK_ASSIGN {
			// Handle assigning a const string to a string variable
			return EmitAssignVariableConstStrStr(lvalue.Offset, value.StringLitNo)
		} else if op == TOK_PLUS_ASGN {
			// Handle appending a const string to a string variable
			return EmitAppendVariableConstStrStr(lvalue.Offset, value.StringLitNo)
		}
	} else if lvalue.Typ.Pt == code.TYP_U64 {
		val, err := strconv.ParseUint(value.StringValue, 0, 64)
		if err != nil {
			return fmt.Errorf("Invalid integer: %s", value.StringValue)
		}
		return EmitAssignVariableConstInt(op, lvalue.Offset, lvalue.Typ.Pt.Size(), val, "")
	} else if lvalue.Typ.Pt == code.TYP_STRING && value.Typ.Pt.IsInteger() && op == TOK_PLUS_ASGN {
		// Handle appending a single character to a string
		val, err := strconv.ParseInt(value.StringValue, 0, 64)
		if err != nil {
			return fmt.Errorf("Invalid integer: %s", value.StringValue)
		}
		return EmitAssignVariableConstStrChar(op, lvalue.Offset, int(val))
	} else if lvalue.Typ.Pt.IsInteger() && value.Typ.Pt.IsInteger() {
		val, err := strconv.ParseInt(value.StringValue, 0, 64)
		if err != nil {
			return fmt.Errorf("Invalid integer: %s", value.StringValue)
		}
		return EmitAssignVariableConstInt(op, lvalue.Offset, lvalue.Typ.Pt.Size(), uint64(val), "")
	} else if lvalue.Typ.Pt == code.TYP_F64 {
		f, err := strconv.ParseFloat(value.StringValue, 64)
		if err != nil {
			return fmt.Errorf("Invalid float: %s", value.StringValue)
		}
		return EmitAssignVariableConstF64(op, lvalue.Offset, f, "")
	} else if lvalue.Typ.Pt == code.TYP_F32 {
		f, err := strconv.ParseFloat(value.StringValue, 64)
		if err != nil {
			return fmt.Errorf("Invalid float: %s", value.StringValue)
		}
		return EmitAssignVariableConstF32(op, lvalue.Offset, float32(f), "")
	} else if lvalue.Typ.Pt == code.TYP_BOOL {
		if value.StringValue == "false" || value.StringValue == "0" || value.StringValue == "nil" {
			return EmitAssignVariableConstInt(op, lvalue.Offset, lvalue.Typ.Pt.Size(), 0, "")
		}
		return EmitAssignVariableConstInt(op, lvalue.Offset, lvalue.Typ.Pt.Size(), 1, "")
	}
	return fmt.Errorf("4 %s not implemented for %s", op.Name(), value.Typ.Name())
}

func AssignStruct(lvalue *VarDef, isNew bool) (err error) {
	if !isNew {
		lbl := EmitCheckForOldStruct(lvalue.Offset)
		err = FreeStruct(lvalue.Typ)
		if err != nil {
			return err
		}
		EmitLabel(lbl, "")
	}
	return EmitAssignVariableExpressionStruct(lvalue.Offset, "Assign struct to "+lvalue.Name)
}

func AssignVariableExpression(op Token, lvalue *VarDef, value *ValueDef, isNew bool) error {
	if lvalue.Typ.Pt == code.TYP_STRING && value.Typ.Pt == code.TYP_STRING {
		if op == TOK_ASSIGN {
			// Handle assigning a string expression to a string variable
			return EmitAssignVariableExpressionStrStr(lvalue.Offset)
		} else if op == TOK_PLUS_ASGN {
			// Handle appending a string expression to a string variable
			return EmitAppendVariableExpressionStrStr(lvalue.Offset)
		}
	} else if lvalue.Typ.Pt == code.TYP_STRING && value.Typ.Pt.IsInteger() && op == TOK_PLUS_ASGN {
		return EmitAppendVariableExpressionStrChar(lvalue.Offset)
	} else if lvalue.Typ.Pt.IsInteger() {
		return EmitAssignVariableExpressionInt(op, lvalue.Typ.Pt.Size(), lvalue.Offset, "Assign int to "+lvalue.Name)
	} else if lvalue.Typ.Pt == code.TYP_F64 {
		return EmitAssignVariableExpressionF64(op, lvalue.Offset, "Assign F64 to "+lvalue.Name)
	} else if lvalue.Typ.Pt == code.TYP_F32 {
		return EmitAssignVariableExpressionF32(op, lvalue.Offset, "Assign F32 to "+lvalue.Name)
	} else if lvalue.Typ.Pt == code.TYP_STRUCT && value.Typ.Pt == code.TYP_STRUCT && op == TOK_ASSIGN {
		return AssignStruct(lvalue, isNew)
	} else if lvalue.Typ.Pt == code.TYP_SLICE && value.Typ.Pt == code.TYP_SLICE && op == TOK_ASSIGN {
		return EmitAssignVariableExpressionSlice(lvalue.Offset, "Assign struct to "+lvalue.Name)
	} else if lvalue.Typ.Pt == code.TYP_BOOL {
		return EmitAssignVariableExpressionInt(op, lvalue.Typ.Pt.Size(), lvalue.Offset, "Assign int to "+lvalue.Name)
	}
	return fmt.Errorf("3 %s not implemented for %s", op.Name(), value.Typ.Name())
}

func AssignIndirectConst(op Token, lvalue *VarDef, value *ValueDef) error {
	if lvalue.Typ.Pt == code.TYP_STRING && value.Typ.Pt == code.TYP_STRING {
		if op == TOK_ASSIGN {
			return EmitAssignIndirectConstStrStr(value.StringLitNo)
		} else if op == TOK_PLUS_ASGN {
			return EmitAppendIndirectConstStrStr(value.StringLitNo)
		}
	} else {
		f, err := StrToConst(value.StringValue)
		if err != nil {
			return fmt.Errorf("AssignIndirectConst: invalid constant %s", value.StringValue)
		}
		if lvalue.Typ.Pt == code.TYP_STRING && value.Typ.Pt.IsInteger() && op == TOK_PLUS_ASGN {
			return EmitAppendIndirectConstStrChar(int(f))
		} else if lvalue.Typ.Pt.IsInteger() {
			return EmitAssignIndirectConstInt(op, lvalue.Typ.Pt.Size(), int64(f), "")
		} else if lvalue.Typ.Pt == code.TYP_F64 {
			return EmitOpAssignIndirectConstF64(op, f)
		} else if lvalue.Typ.Pt == code.TYP_F32 {
			return EmitOpAssignIndirectConstF32(op, float32(f))
		}
	}
	return fmt.Errorf("2 %s not implemented for %s", op.Name(), value.Typ.Name())
}

func AssignIndirectExpression(op Token, lvalue *VarDef, value *ValueDef) (err error) {
	EmitAssertTosInRax("")
	emit("pop", "rdi", "", "rdi is now indirect pointer, rax=result of expression")
	if value.Typ.Pt == code.TYP_STRING && op == TOK_ASSIGN {
		return EmitAssignIndirectExpressionStrStr()
	} else if lvalue.Typ.Pt == code.TYP_STRING && value.Typ.Pt == code.TYP_STRING && op == TOK_PLUS_ASGN {
		return EmitAppendIndirectExpressionStrStr()
	} else if lvalue.Typ.Pt == code.TYP_STRING && value.Typ.Pt.IsInteger() && op == TOK_PLUS_ASGN {
		return EmitAppendIndirectExpressionStrChar()
	} else if lvalue.Typ.Pt.IsInteger() && value.Typ.Pt.IsInteger() {
		return EmitAssignIndirectExpressionInt(op, lvalue.Typ.Pt.Size())
	} else if value.Typ.Pt == code.TYP_F64 {
		return EmitAssignIndirectExpressionF64(op)
	} else if value.Typ.Pt == code.TYP_F32 {
		return EmitAssignIndirectExpressionF32(op)
	} else if op == TOK_ASSIGN && lvalue.Typ.Pt == code.TYP_SLICE && value.Typ.Pt == code.TYP_SLICE {
		return EmitAssignIndirectExpressionSlice()
	} else if op == TOK_ASSIGN && lvalue.Typ.Pt == code.TYP_STRUCT && value.Typ.Pt == code.TYP_STRUCT {
		return EmitAssignIndirectExpressionStruct()
	}
	return fmt.Errorf("AssignIndirectExpression: %s not implemented for %s", op.Name(), value.Typ.Name())
}

// ParseFormalArgList parses the function definition and returns a list of formal arguments
func ParseFormalArgList(s *State) ([]*VarDef, error) {
	var parList []*VarDef
	s.ParCount = 0
	for {
		if s.token == TOK_RPAR {
			break
		}
		if s.token != TOK_ID {
			return parList, fmt.Errorf("expected argument name but got %s", s.tokenString)
		}
		id := s.tokenString
		s.next()
		typ, err := ParseType(s)
		if err != nil {
			return parList, err
		}
		if typ == nil {
			return parList, fmt.Errorf("expected argument type but got nil")
		}
		// Add argument with its implied type, storing it as a local variable
		v := AddLocalPar(s, id, typ)
		parList = append(parList, v)
		if s.token == TOK_RPAR {
			break
		}
		if s.token != TOK_COMMA {
			return parList, fmt.Errorf("expected comma or right parenthesis but got %s", s.tokenString)
		}
		s.next()
	}
	if s.token != TOK_RPAR {
		return parList, fmt.Errorf("expected ')' but got %s", s.tokenString)
	}
	s.next()
	return parList, nil
}

// ParseLvalue will scan up to next comma. The lvalue can consist of either a
// simple variable, a struct variable followed by a dot and field, or an indexed variable
// followed by a bracket expression. We can have a row of indexes/fields
func ParseLvalue(s *State, id string) (*VarDef, error) {
	lvalue := VarDefs[id]
	var ok bool
	startsp := code.LocalSp
	// Loop over field access or indexed access.
	for {
		if s.found(TOK_DOT) {
			if lvalue == nil {
				return nil, fmt.Errorf("new identifier '%s' before dot. Struct must exist", id)
			} else if lvalue.Typ.Pt == code.TYP_STRUCT && s.token == TOK_ID {
				if lvalue.IsIndirect {
					EmitLoadTosIndirect(8, lvalue.Name)
				}
				// The id was followed by a dot and a field id, indicated field access.
				fieldName := s.tokenString
				s.next()
				v := &VarDef{}
				v.Typ, ok = lvalue.Typ.Fields[fieldName]
				if !ok {
					return nil, fmt.Errorf("expected field name '%s' of the struct '%s' but it was not found", fieldName, id)
				}
				v.Name = fieldName
				fieldOfs := lvalue.Typ.Offsets[fieldName]
				EmitLoadField(lvalue.Offset, lvalue.IsIndirect, fieldOfs, lvalue.Name, fieldName)
				v.IsIndirect = true
				lvalue = v
			} else {
				return nil, fmt.Errorf("dot found after something that is not a struct")
			}
		} else if s.found(TOK_LBRACK) {
			// Parse the expression inside brackets. It will result in an index in TOS, or a constant valued index
			index, err := ParseIndex(s, lvalue)
			if err != nil {
				return nil, err
			}
			if !s.found(TOK_RBRACK) {
				return nil, fmt.Errorf("expected ']' but got %s", s.tokenString)
			}

			if !lvalue.IsIndirect && lvalue.Offset == 0 {
				return nil, fmt.Errorf("local var address is 0")
			}
			intValue, err := StrToConst(index.StringValue)
			if lvalue.IsIndirect && lvalue.Typ.Pt == code.TYP_STRING && index.IsConst {
				EmitModifyConstIndexedCharIndirect(int(intValue))
			} else if lvalue.Typ.Pt == code.TYP_STRING && index.IsConst {
				if lvalue.Offset == 0 {
					return nil, fmt.Errorf("local var address is 0")
				}
				EmitModifyConstIndexedChar(lvalue.Offset, int(intValue))
			} else if lvalue.IsIndirect && lvalue.Typ.Pt == code.TYP_STRING {
				EmitModifyIndexedCharIndirect()
			} else if lvalue.Typ.Pt == code.TYP_STRING {
				EmitAssertTosInRax("")
				EmitModifyIndexedChar(lvalue.Offset)
			} else if lvalue.Typ.Pt == code.TYP_SLICE && index.IsConst {
				if !lvalue.IsIndirect {
					EmitLea(lvalue.Offset, "Load indirect var address")
				}
				EmitModifyConstIndexedSlice(int(intValue), lvalue.Typ.Element.Size(), s.returnLbl)
			} else if lvalue.Typ.Pt == code.TYP_SLICE {
				if !lvalue.IsIndirect {
					EmitLea(lvalue.Offset, "Load indirect var address")
				}
				EmitModifyIndexedSlice(lvalue.Typ.Element.Size())
			}
			// Multiply by element size
			if lvalue.Typ.Pt == code.TYP_STRING {
				lvalue = &VarDef{Typ: &U8Type, IsIndirect: true}
			} else if lvalue.Typ.Pt == code.TYP_SLICE {
				lvalue = &VarDef{Typ: lvalue.Typ.Element, IsIndirect: true}
			}
		} else if lvalue == nil {
			// New local variable,we don't yet know the type, so just use nil
			lvalue = AddLocalVar(s, id, nil)
			// NB: Actual size is not known. Allocation must be delayed to the time we set the type
			EmitAllocLocalVar("Assign stack space for " + id + " level=" + strconv.Itoa(s.BlockLevel))
		} else {
			break
		}
	}
	if code.LocalSp > startsp+1 {
		fmt.Printf("Start stack=%d, exit stack=%d\n", startsp, code.LocalSp)
		panic("Stack error for lvalue")
	}
	return lvalue, nil
}

func ShiftFromSize(size int) string {
	if size == 1 {
		return "0"
	} else if size == 2 {
		return "1"
	} else if size == 4 {
		return "2"
	} else if size == 8 {
		return "3"
	}
	panic("Size of element must be 1,2,4 or 8 bytes")
}

// ParseLvalueList parses a list of lvalues to the left of = , += etc.
// The first identifier is given in parameter id.
func ParseLvalueList(s *State, id string) (lvalues []*VarDef, err error) {
	// EmitComment(">>>>> Start parsing Lvalue List")

	// For each lvalue, separated by comma. The identifier is already in the id variable
	for {
		lvalue, err2 := ParseLvalue(s, id)
		if err2 != nil {
			return nil, err2
		}
		// lvalue.IsIndirect = code.AxIsTos()
		EmitFlushRax("Assure pointer is on stack")
		lvalues = append(lvalues, lvalue)
		if !s.found(TOK_COMMA) {
			break
		}
		// ALl lvalues must start with an identifier
		if s.token != TOK_ID {
			return nil, fmt.Errorf("expected variable name after comma, but but got %s", s.tokenString)
		}
		id = s.tokenString
		s.next()
	}
	// Create new vardefs for new local variables with unknown type.
	for _, v := range lvalues {
		if v.Typ == nil {
			// VarDefs[v.Name].Offset = EmitAllocLocalVar("Allocate local variable " + v.Name)
		}
	}
	// txt := fmt.Sprintf(">>>> ParseLvalueList done, len(lvalues)=%d, indirect=%v, offset=%d", len(lvalues), lvalues[0].IsIndirect, lvalues[0].Offset)
	// EmitComment(txt)
	return lvalues, err
}

// ParseActualArgList
// For each actual argument in the argument list, generate code in ArgCode and Value in valueList
// Assumes the ( is consumed already
func ParseActualArgList(s *State, funcname string) (valueList []*ValueDef, err error) {
	parNo := 0
	for { // each argument in the actual argument list
		parNo++
		if s.token == TOK_RPAR {
			break
		}
		// Parse the argument and save the type of the result in the value list
		if parNo > 1 {
			code.NewArgCode()
		}
		code.SetUndef()
		EmitComment("Argument " + strconv.Itoa(parNo) + " of argument list for '" + funcname + "'")
		values, err1 := ParseExpression(s)
		if err1 != nil {
			return nil, err1
		}
		valueList = append(valueList, values...)
		code.PushCleanupCode()
		value := values[0]
		if value.HasValue() {
			f := 0.0
			if value.Typ.Pt.IsInteger() || value.Typ.Pt.IsFloat() {
				f, err = StrToConst(value.StringValue)
				if err != nil && value.Typ.Pt != code.TYP_STRING && value.Typ.Pt != code.TYP_BOOL {
					return nil, fmt.Errorf("ParseActualArgList: invalid constant %s", value.StringValue)
				}
			}
			// Constants/literals are passed as pointers on the stack by EmitPushStringLit() or EmitPushConst() or PushFloat()
			if value.Typ.Pt == code.TYP_STRING {
				EmitPushStringLit(value.StringLitNo, "Actual argument nr "+strconv.Itoa(parNo)+" is string literal")
				EmitPushTos(parNo, funcname)
				if funcname == "printf" || funcname == "print" {
					EmitSkipLenCapForPrint()
				}
			} else if value.Typ.Pt.IsInteger() {
				EmitPushConst(int64(f), "")
				EmitPushTos(parNo, funcname)
			} else if value.Typ.Pt == code.TYP_BOOL {
				if value.StringValue == "0" {
					EmitPushConst(0, "")
				} else {
					EmitPushConst(1, "")
				}
				EmitPushTos(parNo, funcname)
			} else if value.Typ.Pt == code.TYP_F64 {
				EmitPushF64Lit(f)
			} else if value.Typ.Pt == code.TYP_F32 {
				if funcname == "printf" || funcname == "print" {
					// Must convert to F64 for print/fprint
					EmitPushF64Lit(f)
				} else {
					EmitPushF32Lit(float32(f))
				}
			} else {
				// TODO: Handle other types
				return nil, fmt.Errorf("constant arguments of type %s is not yet handled", value.Typ.Pt.Name())
			}
		} else {
			if parNo == 1 && funcname == "print" {
				// Check that the first parameter is a constant string literal
				return nil, fmt.Errorf("print's first parameter must be a constant string")
			}
			EmitPushTos(parNo, funcname)
			if funcname == "printf" || funcname == "print" || funcname == "assert" && parNo > 1 {
				// We have a value on the stack (TOS). printf needs special handling.
				if value.Typ.Pt == code.TYP_STRING {
					EmitSkipLenCapForPrint()
					// If it was a local variable or a constant, we should not free it.
					// (The constant case has already been handled)
					// But if it was a function result, it can be a pointer to a literal.
					if value.IsTempObj {
						v := "   mov rax, rsp  ;  printf() cleanup arg " + strconv.Itoa(parNo) + "\n"
						v += "   add rax, rbx\n"
						v += "   sub rax, " + strconv.Itoa(parNo*8-8) + "\n"
						// v += "   mov rcx, [rax]\n"  // Check if cap is zero
						// v += "   and rcx, 0x[rax]\n"
						v += "   mov rax, [rax]\n"
						v += "   sub rax, 8\n" // The stack contains a C-string pointer, so adjust it back
						v += "   call _free_str\n\n"
						code.SetCleanupCode(v)
					}
				} else if value.Typ.Pt == code.TYP_F64 {
					EmitFlushRax("Float arg to printf")
				} else if value.Typ.Pt == code.TYP_F32 {
					EmitFlushRax("Float arg to printf")
					// Special case: Convert F32 to F64 for printf
					EmitConvertF32toF64()
				} else if value.Typ.Pt.IsInteger() {
					EmitFlushRax("Integer arg to printf")
				} else if value.Typ.Pt == code.TYP_STRUCT {
					EmitFlushRax("Struct field arg to printf")
				} else if value.Typ.Pt == code.TYP_PTR {
					EmitFlushRax("Ptr field arg to printf")
				} else if value.Typ.Pt == code.TYP_BOOL {
					EmitFlushRax("Bool arg to printf")
				} else {
					return nil, fmt.Errorf("printf arguments of type %s is not yet handled", value.Typ.Pt.Name())
				}
			} else {
				// We have a simple value on the stack. Just continue.
			}
		}
		if s.token != TOK_COMMA {
			break
		}
		s.next()
	}
	if s.token != TOK_RPAR {
		return nil, fmt.Errorf("expected right parenthesis but got %s", s.tokenString)
	}
	// Skip the final ")"
	s.next()
	return valueList, nil
}

// ParseFuncCall parses a function call and its arguments
// This is the only location where arguments are evaluated
// Assumes id and ( is already consumed
func ParseFuncCall(s *State, id string, returnSomething bool) ([]*ValueDef, error) {
	HadAx := code.AxIsTos()
	s.currentFuncCall = id
	if FuncCount(id) == 0 {
		s.currentFuncCall = ""
		return nil, fmt.Errorf("expected a function name, got: %s", id)
	}

	// Parse the argument list and push each arg
	// -------------------------------------------------------
	values, err := ParseActualArgList(s, id)
	if err != nil {
		s.currentFuncCall = ""
		return nil, err
	}
	f := FindFuncDefFromParList(id, TypeListVal(values))
	if f == nil {
		return nil, fmt.Errorf("function %s with wrong parameters", id)
	}

	for parNo, value := range values {
		// We have a heap object pointer on top of the stack. If the formal parameter is not "var",
		// and it is the result of a function call, then we have to free it after the call.
		// if !f.parameters[min(i, len(f.parameters))-1].IsInputType {
		// If it was a local variable or a constant, we should not free it.
		// TODO: FInd a better way than checking for the names cptr and lptr
		if value.Typ.Pt == code.TYP_STRING && value.IsTempObj && id != "cptr" && id != "lptr" {
			lbl := code.NewLabel()
			str := "   mov rax, rsp   ; Cleanup args\n"
			str += fmt.Sprintf("   add rax,%d\n", parNo*8)
			str += "   mov rax, [rax]  ; Get pointer to content (string cap/len) on stack\n"
			str += "   mov rbx, [rax]  ; Get len/cap\n"
			str += "   shr rbx, 32\n"
			str += "   or rbx, rbx\n"
			str += "   jz .L" + strconv.Itoa(lbl) + "\n"
			str += fmt.Sprintf("   call _free_str   ; Call free arg %d of %s\n", parNo+1, id)
			str += ".L" + strconv.Itoa(lbl) + ":\n"
			code.SetCleanupCode(str)
		} else if value.Typ.Pt == code.TYP_STRUCT && value.IsTempObj {
			str := "   mov rax, rsp   ; Cleanup struct args\n"
			str += fmt.Sprintf("   add rax,%d\n", parNo*8)
			str += "   mov rax, [rax]  ; Get pointer to struct content\n"
			str += "   mov rcx, " + strconv.Itoa(value.Typ.StructSize) + "\n"
			str += fmt.Sprintf("   call _free_struct   ; Call free arg %d of %s\n", parNo+1, id)
			code.SetCleanupCode(str)
		}
	}

	// Check for correct argument types
	/* TODO
	for i, v := range values {
		par := f.parameters[min(i, len(f.parameters)-1)]
		if !CanAssign(par.Typ.Pt, v.Typ.Pt) && !strings.HasPrefix(id, "print") && par.Typ.Pt != code.TYP_NONE {
			return nil, fmt.Errorf("wrong type for parameter %d in %s", i+1, id)
		}
	}*/

	s.currentFuncCall = id
	nac := len(code.ArgCode)
	if len(values) == 0 && nac >= 1 && code.ArgCode[nac-1] == "" {
		code.ArgCode = code.ArgCode[0 : nac-1]
	}
	// Make space for return values. This code is added to the ArgCode stack.
	if HadAx {
		code.SetAx()
	}
	code.NewArgCode()
	EmitFlushRax("Flush rax before function call")
	EmitAddToSp(len(f.returnTypes), "Make space for return values from "+f.name)

	if len(values) == 0 {
		code.ConsArgCode(2, true)
	} else {
		code.ConsArgCode(len(values)+1, true)
	}

	// Do actual call
	// ----------------------------------
	EmitCall(f.label, len(values), f.builtin)

	code.OutputCleanupCode(len(values))
	EmitAddToSp(-len(values), "Drop "+strconv.Itoa(len(values))+" arguments after call. ")

	if !returnSomething && len(f.returnTypes) > 0 {
		return nil, fmt.Errorf("Funcion returns values that are not handled")
	}
	if !returnSomething && len(f.returnTypes) == 0 {
		// The function call should be alone, so just continue
		s.currentFuncCall = ""
		return nil, nil
	}
	var results []*ValueDef
	for _, t := range f.returnTypes {
		results = append(results, &ValueDef{Typ: t, IsReturned: true, IsTempObj: t.Pt.IsObject()})
	}
	// Function results are on stack and not in RAX.
	code.SetSp()

	s.currentFuncCall = ""
	return results, nil
}

// ParseAssign - this might be the start of a lvalue list or a function call
// The first variable name is now in id. The current token may be '.' or '[' or an assignment token.
func ParseAssign(s *State, id string) error {
	// Expect a list of lvalues
	lvalues, err := ParseLvalueList(s, id)
	if err != nil {
		return err
	}
	op := s.token
	if s.found(TOK_ASSIGN, TOK_PLUS_ASGN, TOK_MINUS_ASGN, TOK_MULT_ASGN, TOK_DIV_ASGN) {
		if len(lvalues) > 1 && op != TOK_ASSIGN {
			return fmt.Errorf("Can not have many lvalues for " + op.Name())
		}

		// Now parse the expression(s) to find the value(s)
		values, err := ParseExpressions(s)
		if err != nil {
			return err
		}
		if len(values) != len(lvalues) {
			return fmt.Errorf("expected %d values but got %d", len(lvalues), len(values))
		}
		if op != TOK_ASSIGN && len(lvalues) > 1 {
			return fmt.Errorf("can not use %s on more than one target", op.Name())
		}
		// Check that all values have a type.
		for _, value := range values {
			if value.Typ == nil {
				// return fmt.Errorf("no type for \"%s\"", id)
			}
		}
		// Assign values to lvalues
		for i, value := range values {
			err = GenerateAssignment(op, lvalues[i], value)
			if err != nil {
				return err
			}
		}
		// Destroy local variables that are pointers (destructive read).
		for _, value := range values {
			if value.localVar != nil && op == TOK_ASSIGN {
				value.localVar.Destroyed = true
			}
		}
		code.ConsArgCode(len(code.ArgCode), false)
		code.OutputArgCode()
	} else {
		return fmt.Errorf("expected assignment, got \"%s\"", s.tokenString)
	}
	return nil
}

func ParseTypeConversion(s *State, typ1 *TypeDef) (values []*ValueDef, err error) {
	// This is a type conversion. First parse value to convert
	if code.AxIsTos() {
		EmitPushAx("ParseTypeConversion")
		code.SetSp()
	}
	values, err = ParseExpression(s)
	if err != nil {
		return nil, err
	}
	value := values[0]
	if value.Typ == nil {
		var err error
		value.Typ, err = TypeFromConstValue(value)
		if err != nil {
			return nil, fmt.Errorf("type conversion with unknown type")
		}
	}

	if CanAssign(typ1.Pt, value) {
		value.Typ = typ1
	} else {
		err = fmt.Errorf("can not convert from %s to %s", value.Typ.Pt.Name(), typ1.Pt.Name())
	}
	if !s.found(TOK_RPAR) {
		return nil, fmt.Errorf("expected right parentheses")
	}
	return []*ValueDef{value}, err
}

// ParsePointer handles conversion of variable to a pointer to that variable
func ParsePointer(s *State, id string) (values []*ValueDef, err error) {
	if !s.found(TOK_LPAR) {
		return nil, fmt.Errorf("expected left parentheses")
	}
	if s.token != TOK_ID {
		return nil, fmt.Errorf("expected id after (")
	}
	id = s.tokenString
	s.next()
	if !s.found(TOK_RPAR) {
		return nil, fmt.Errorf("expected left parenthesis after 'ptr'")
	}
	// Lookup variable
	v, ok := VarDefs[id]
	if !ok {
		return nil, fmt.Errorf("expected local variable, got %s", id)
	}
	ofs := v.Offset
	EmitGetAddrOfLocal(ofs)
	return []*ValueDef{&PtrValue}, nil
}

// ParseIndex will parse the expression inside [] after a local variable
// and leave the resulting address in TOS or as a constant in the returned value.
func ParseIndex(s *State, v *VarDef) (value *ValueDef, err error) {
	if v == nil {
		return nil, fmt.Errorf("did not find variable")
	}
	if v.Typ.Pt != code.TYP_STRING && v.Typ.Pt != code.TYP_SLICE {
		return nil, fmt.Errorf("expected string or array, got %s", v.Typ.Pt.Name())
	}
	values, err := ParseExpression(s)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("expected 1 value but got %d", len(values))
	}
	return values[0], nil
}

// ParseArrayOrStruct handles a string of array or field references.
// F.ex. a[14].f.r[i]. It emits code to load the resulting value.
// Current token is either TOK_DOT or TOK_LBRACK when this function is called
func ParseArrayOrStruct(s *State, id string) ([]*ValueDef, error) {
	var isIndirect bool
	var f float64
	vp, ok := VarDefs[id]
	if vp == nil {
		return nil, fmt.Errorf("variable %s does not exist", id)
	}
	v := *vp
	if !ok {
		return nil, fmt.Errorf("expected local variable, got %s", id)
	}
	for {
		if s.found(TOK_DOT) {
			// The previous item must have been a struct, and the current id a field name in that struct.
			if v.Typ.Pt != code.TYP_STRUCT || s.token != TOK_ID {
				return nil, fmt.Errorf("expected struct field, got %s", v.Typ.Pt.Name())
			}
			fieldName := s.tokenString
			s.next()
			fieldType := v.Typ.Fields[fieldName]
			if fieldType == nil {
				return nil, fmt.Errorf("field '%s' not found in struct '%s'", fieldName, v.Name)
			}
			// A struct field
			fieldOffset, isOk := v.Typ.Offsets[fieldName]
			if !isOk {
				return nil, fmt.Errorf("field \"%s\" not found", fieldName)
			}
			EmitLoadField(v.Offset, isIndirect, fieldOffset, id, fieldName)
			isIndirect = true
			EmitLoadTosIndirect(fieldType.Size(), fieldName)
			v.Typ = fieldType
		} else if s.found(TOK_LBRACK) {
			// It should be an array
			if v.Typ.Pt != code.TYP_SLICE && v.Typ.Pt != code.TYP_STRING {
				return nil, fmt.Errorf("expected slice/array, got %s", v.Typ.Pt.Name())
			}
			index, err := ParseIndex(s, &v)
			if err != nil || index == nil {
				return nil, err
			}
			if !s.found(TOK_RBRACK) {
				return nil, fmt.Errorf("missing right bracket")
			}
			var size int
			if v.Typ.Pt == code.TYP_STRING {
				size = 1
			} else if v.Typ.Pt == code.TYP_SLICE {
				size = v.Typ.Element.Size()
			}
			f, err = StrToConst(index.StringValue)
			if index.IsConst && v.IsGlobal {
				EmitGlobalIndexedByConstValue(id, size, int(f))
			} else if vp.IsGlobal {
				EmitGlobalIndexedTosValue(id, size)
			} else {
				LoadIndexedValue(isIndirect, index.IsConst, v.Offset, int64(f), size)
			}
			if size == 1 {
				v.Typ = &U8Type
			} else if size >= 2 {
				v.Typ = v.Typ.Element
			}

		} else {
			break
		}
	}
	value := &ValueDef{Typ: v.Typ, IsIndirect: isIndirect}
	return []*ValueDef{value}, nil
}

// HandleImportedName will check if the id is a package short-name
// If so, it will mangle the name with the package path
// Assumes the id is in s.TokenString
func HandleImportedName(s *State) (string, error) {
	id := s.tokenString
	s.next()
	path, ok := ImportedPackages[id]
	if ok {
		if s.token != TOK_DOT {
			return "", fmt.Errorf("expected dot after package")
		}
		s.next()
		id = MangleFun(path, s.tokenString)
		s.next()
	}
	return id, nil
}

// ParseVarOrFunc is called for a unary function or variable.
// Called when an identifier is encountered in an expression
// We now have s.token == TOK_ID
func ParseVarOrFunc(s *State) ([]*ValueDef, error) {
	id, err := HandleImportedName(s)
	if err != nil {
		return nil, err
	}
	// Handle the special keyword ptr used to convert var into pointer
	if id == "ptr" {
		return ParsePointer(s, id)
	}
	if s.found(TOK_LPAR) {
		// An ID followed by left parantesis can be a type conversion or a function call
		typ, ok := TypeDefs[id]
		if ok {
			return ParseTypeConversion(s, typ)
		}
		return ParseFuncCall(s, id, true)
	} else if s.token == TOK_LBRACK || s.token == TOK_DOT {
		// It is an array or a struct field. Handle them in a loop in the ParseArrayOrStruct function
		return ParseArrayOrStruct(s, id)
	}
	// If none above, it is a simple variable
	localVar, ok := VarDefs[id]
	if localVar == nil {
		return nil, fmt.Errorf("expected local variable, got %s", id)
	}
	value := &ValueDef{Typ: localVar.Typ}
	if !ok {
		return nil, fmt.Errorf("did not find variable \"%s\"", id)
	}
	if localVar.Name == "err" {
		EmitLoadErr()
	} else if localVar.Typ.Pt.IsFloat() {
		EmitLoadFloat(localVar.Typ.Size(), localVar.Offset, localVar.Name)
	} else if localVar.IsGlobal && localVar.constValue != "" {
		EmitLoadGlobalConst(localVar.constValue)
	} else if localVar.IsGlobal && localVar.Typ.Pt.IsInteger() {
		EmitLoadGlobalVar(localVar.Name, localVar.Typ.Pt)
	} else if localVar.Typ.Pt.IsInteger() {
		if localVar.Offset == 0 {
			return nil, fmt.Errorf("variable \"%s\" has zero offset", localVar.Name)
		}
		EmitLoad(localVar.Typ.Pt.Size(), localVar.Offset, "Load variable "+localVar.Name)
	} else {
		localVar.Destroyed = s.ParsingReturnValue
		value.localVar = localVar
		EmitLoad(localVar.Typ.Pt.Size(), localVar.Offset, "Load struct/string variable "+localVar.Name)
	}
	s.ParsingReturnValue = false
	return []*ValueDef{value}, nil
}

// ParseUnary will parse a parenthesis term, a number, a string, a function call
func ParseUnary(s *State, hasUnaryMinus bool) ([]*ValueDef, error) {
	var err error
	value := &ValueDef{}
	if s.token == TOK_ID {
		// An id can be either a variable or a function call. A func call must return one value
		var values []*ValueDef
		values, err = ParseVarOrFunc(s)
		if err != nil {
			return nil, err
		}
		return values, nil
	} else if s.token == TOK_LPAR {
		// Start of parenthesis term
		s.next()
		// EmitFlushRax("Begin parenthesis term")
		values, err2 := ParseExpression(s)
		if err2 != nil {
			return nil, err2
		}
		// EmitFlushRax("End parenthesis term")
		return values, Expect(s, TOK_RPAR)
	} else if s.token == TOK_NUM {
		value = &ValueDef{IsConst: true}
		value.StringValue = s.tokenString
		err = UpdateType(value)
		if err != nil {
			return nil, err
		}
		s.next()
		/*
					value.IntValue = int64(s.ConstValue.Bits)
					value.UintValue = s.ConstValue.Bits
					value.FloatValue = float64(s.ConstValue.Bits)
					value.Typ = TypeDefs[s.ConstValue.Pt.Name()]
					if hasUnaryMinus {
						s.ConstValue.Bits = uint64(-int64(s.ConstValue.Bits))
						value.IntValue = int64(s.ConstValue.Bits)
						value.FloatValue = float64(s.ConstValue.Bits)
						value.UintValue = s.ConstValue.Bits
						if value.Typ.Pt == code.TYP_U8 {
							value.Typ = TypeDefs["I16"]
						} else if value.Typ.Pt == code.TYP_U16 {
							value.Typ = TypeDefs["I32"]
						} else if value.Typ.Pt == code.TYP_U32 {
							value.Typ = TypeDefs["I64"]
						}
					}
					if value.Typ == nil {
						return nil, fmt.Errorf("missing integer type")
					}
			} else if s.token == TOK_FLOAT {
				if hasUnaryMinus {
					v := -math.Float64frombits(s.ConstValue.Bits)
					s.ConstValue.Bits = math.Float64bits(v)
				}
				value.FloatValue = math.Float64frombits(s.ConstValue.Bits)
				value.Typ = TypeDefs["F64"]
				value.IsConst = true
				s.next()*/
	} else if s.token == TOK_CHAR {
		value = &ValueDef{IsConst: true}
		var num int64
		num, err = strconv.ParseInt(s.tokenString, 10, 64)
		// TODO Check range
		value.StringValue = strconv.Itoa(int(num))
		value.Typ = TypeDefs["U8"]
		s.next()
	} else if s.token == TOK_STRING {
		litNo := AddLiteral(s.tokenString)
		value.Typ = TypeDefs["String"]
		value.StringValue = s.tokenString
		value.StringLitNo = litNo
		value.IsConst = true
		s.next()
	} else if s.token == TOK_LBRACK {
		for {
			s.next()
			if s.token == TOK_RBRACK {
				s.next()
				break
			}
		}
	} else if s.token == TOK_TRUE {
		value = &True
		s.next()
	} else if s.token == TOK_FALSE {
		value = &False
		s.next()
	} else if s.token == TOK_NOT {
		s.next()
		// EmitFlushRax("Begin parenthesis term")
		values, err2 := ParseExpression(s)
		if err2 != nil {
			return nil, err2
		}
		if len(values) != 1 {
			return nil, fmt.Errorf("after ! we expected 1 value but got %d", len(values))
		}
		if values[0].Typ.Pt != code.TYP_BOOL {
			return nil, fmt.Errorf("after ! we expected boolean value but got %v", values[0].Typ.Pt)
		}
		EmitNot()
		value = values[0]
	} else if s.token == TOK_NEW {
		s.next()
		if !s.found(TOK_LPAR) {
			return nil, fmt.Errorf("expected left parenthesis")
		}
		id := s.tokenString
		s.next()
		t, ok := TypeDefs[id]
		if !ok {
			return nil, fmt.Errorf("should have a predefined type, found %s", id)
		}
		value.Typ = t
		if t.Pt == code.TYP_STRUCT {
			EmitNewStruct(t)
			if !s.found(TOK_RPAR) {
				return nil, fmt.Errorf("expected right parenthesis")
			}
			return []*ValueDef{value}, err
		}

		if !s.found(TOK_COMMA) {
			return nil, fmt.Errorf("new must have a given capacity")
		}
		v, err1 := ParseExpression(s)
		if err1 != nil {
			return nil, err
		}
		if v[0].IsConst {
			f, err2 := StrToConst(v[0].StringValue)
			if err2 != nil {
				return nil, fmt.Errorf("invalid integer value: %s", value.StringValue)
			}
			EmitPushConst(int64(f), "")
		}
		if t.Pt == code.TYP_STRING {
			EmitNewString()
		} else if t.Pt == code.TYP_SLICE {
			EmitNewSlice(t.Element.Size())
		}
		if !s.found(TOK_RPAR) {
			return nil, fmt.Errorf("expected right parenthesis")
		}

	} else if s.token == TOK_MINUS {
		s.next()
		v, err3 := ParseUnary(s, true)
		if err3 != nil {
			return nil, err3
		}
		if len(v) != 1 {
			return nil, fmt.Errorf("unary minus on invalid value")
		}
		if v[0].IsConst {
			v[0].StringValue = "-" + v[0].StringValue
		} else if v[0].Typ.Pt == code.TYP_F64 {
			EmitNegateF64()
		} else if v[0].Typ.Pt == code.TYP_F32 {
			EmitNegateF32()
		} else {
			EmitNegate()
		}
		value = v[0]
	} else if s.token == TOK_SYSCALL {
		s.next()
		err = ParseSyscall(s)
		// Syscall allways retunes a single value
		value.Typ = &I32Type
	} else {
		return nil, fmt.Errorf("unexpected unary token %s", s.tokenString)
	}
	return []*ValueDef{value}, err
}

func ParseProd(s *State) ([]*ValueDef, error) {
	values1, err := ParseUnary(s, false)
	if err != nil {
		return nil, err
	}
	var values2 []*ValueDef
	for s.token == TOK_MULT || s.token == TOK_DIV || s.token == TOK_MOD || s.token == TOK_SHL || s.token == TOK_SHR || s.token == TOK_AND_NOT || s.token == TOK_AND {
		if len(values1) != 1 {
			return nil, fmt.Errorf("* and / can only operate on 1 value but got %d", len(values1))
		}
		op := s.token
		s.next()
		code.NewArgCode()
		values2, err = ParseUnary(s, false)
		if err != nil {
			return nil, err
		}
		if len(values2) != 1 {
			return nil, fmt.Errorf("* and / can only operate on 1 value but got %d", len(values2))
		}
		code.NewArgCode()
		values1[0], err = GenerateOp(op, values1[0], values2[0])
		code.ConsArgCode(3, false)
		if err != nil {
			return nil, err
		}
		values1[0].IsReturned = false
	}
	return values1, nil
}

func ParseSumTerm(s *State) ([]*ValueDef, error) {
	// ParseProd should push rax and leave new result in rax
	values1, err := ParseProd(s)
	if err != nil {
		return nil, err
	}
	UpdateType(values1[0])
	var values2 []*ValueDef
	if s.token == TOK_PLUS && values1[0].Typ.Pt == code.TYP_STRING {
		if len(values1) != 1 {
			return nil, fmt.Errorf("+ and - can only operate on 1 value but got %d", len(values1))
		}
		// Concatenation of two or more strings
		if values1[0].HasValue() {
			EmitPushConstString(values1[0].StringLitNo)
		}
		// Loop through all strings that are concatenated
		for s.token == TOK_PLUS {
			s.next()
			// ParseProd should push rax and leave new result in rax
			code.NewArgCode()
			values2, err = ParseProd(s)
			if err != nil {
				return nil, err
			}
			if values2[0].HasValue() {
				EmitPushStringLit(values2[0].StringLitNo, "Sum term push value2")
				values2[0].IsTempObj = false
			}
			if values2[0].Typ.Pt != code.TYP_STRING {
				return nil, fmt.Errorf("string can only be concatenated with another string")
			}
			code.NewArgCode()
			EmitConcat(values1[0].IsTempObj, values2[0].IsTempObj)
			code.ConsArgCode(3, false)
			values1[0] = &ValueDef{Typ: &StringType, IsTempObj: true}
		}
	}
	for s.token == TOK_PLUS || s.token == TOK_MINUS || s.token == TOK_OR || s.token == TOK_XOR {
		if len(values1) != 1 {
			return nil, fmt.Errorf("+ and - can only operate on 1 value but got %d", len(values1))
		}
		op := s.token
		s.next()
		code.NewArgCode()
		values2, err = ParseProd(s)
		if err != nil {
			return nil, err
		}
		code.NewArgCode()
		values1[0], err = GenerateOp(op, values1[0], values2[0])
		code.ConsArgCode(3, false)
		if err != nil {
			return nil, err
		}
		values1[0].IsReturned = false
	}
	return values1, nil
}

func ParseCompareTerm(s *State) ([]*ValueDef, error) {
	values1, err := ParseSumTerm(s)
	if err != nil {
		return nil, err
	}
	if s.token != TOK_LT && s.token != TOK_GT && s.token != TOK_EQ && s.token != TOK_GE && s.token != TOK_LE && s.token != TOK_NE {
		// Not a compare operation, return value1 immediately
		return values1, nil
	}
	values1[0].IsReturned = false
	op := s.token
	s.next()
	code.NewArgCode()
	values2, err := ParseSumTerm(s)
	if err != nil {
		return nil, err
	}
	code.NewArgCode()
	values1[0], err = GenerateOp(op, values1[0], values2[0])
	code.ConsArgCode(3, false)
	if err != nil {
		return nil, err
	}
	return values1, err
}

func ParseLogAnd(s *State) ([]*ValueDef, error) {
	results, err := ParseCompareTerm(s)
	if err != nil {
		return nil, err
	}
	endLabel := 0
	for s.token == TOK_LOG_AND {
		if len(results) != 1 {
			return nil, fmt.Errorf("+ and - can only operate on 1 value but got %d", len(results))
		}
		if results[0].IsConst {
			EmitLoadBool(results[0].StringValue)
		}
		results[0].IsReturned = false
		if endLabel == 0 {
			endLabel = code.NewLabel()
		}
		if results[0].Typ.Pt != code.TYP_BOOL {
			return nil, fmt.Errorf("%s requires boolean operands", s.tokenString)
		}
		s.next()
		EmitJumpFalse("al", endLabel, "")
		values2, err2 := ParseCompareTerm(s)
		if err2 != nil {
			return nil, err2
		}
		if values2[0].Typ.Pt != code.TYP_BOOL {
			return nil, fmt.Errorf("%s requires boolean operands", s.tokenString)
		}
	}
	if endLabel != 0 {
		EmitLabel(endLabel, "")
	}
	return results, nil
}

func ParseExpression(s *State) ([]*ValueDef, error) {
	results, err := ParseLogAnd(s)
	if err != nil {
		return nil, err
	}
	endLabel := 0
	for s.token == TOK_LOG_OR {
		if len(results) != 1 {
			return nil, fmt.Errorf("+ and - can only operate on 1 value but got %d", len(results))
		}
		if results[0].IsConst {
			EmitLoadBool(results[0].StringValue)
		}
		results[0].IsReturned = false
		if endLabel == 0 {
			endLabel = code.NewLabel()
		}
		if results[0].Typ.Pt != code.TYP_BOOL {
			return nil, fmt.Errorf("%s requires boolean operands", s.tokenString)
		}
		s.next()
		EmitJumpTrue("al", endLabel, "")
		values2, err2 := ParseLogAnd(s)
		if err2 != nil {
			return nil, err2
		}
		if values2[0].Typ.Pt != code.TYP_BOOL {
			return nil, fmt.Errorf("%s requires boolean operands", s.tokenString)
		}
	}
	if endLabel != 0 {
		EmitLabel(endLabel, "")
	}
	if results[0].Typ == nil {
		// TODO return nil, fmt.Errorf("value.type is nil - internal error 2")
	}
	return results, nil
}

// ParseExpressions will parse either a comma separated list of values,
// or a function call returning potentially many values.
// It is called from the ParseAssign() function
func ParseExpressions(s *State) ([]*ValueDef, error) {
	var values []*ValueDef
	code.NewArgCode()
	n := 0
	for {
		n++
		code.SetUndef()
		v, err := ParseExpression(s)
		if err != nil {
			return nil, err
		}
		values = append(values, v...)
		if !s.found(TOK_COMMA) {
			break
		}
		code.NewArgCode()
	}
	code.ConsArgCode(n, false)
	return values, nil
}

// ParseBlock assumes that { has been consumed (by s.found())
func ParseBlock(s *State, isTrue bool) error {
	if isTrue {
		s.noCode++
	}
	if len(code.ArgCode) > 0 {
		panic("ParseBlock: ArgCode was not empty")
	}
	s.BlockLevel++
	err := ParseStatements(s)
	if err != nil {
		return err
	}
	err = FreeBlockVars(s.BlockLevel)
	if err != nil {
		return err
	}
	s.BlockLevel--
	s.HasReturned = false
	if isTrue {
		s.noCode--
		if s.noCode < 0 {
			panic("negative noCode")
		}
	}
	return nil
}

// ParseColonQmark will parse the code after '?' or ':'
func ParseColonQmark(s *State, value *ValueDef) (err error) {
	L1, L2 := 0, 0
	oldHasReturned := s.HasReturned
	if !value.HasValue() {
		L1 = code.NewLabel()
		EmitAssertTosInRax("Pop TOS into rax before assignment")
		EmitJumpFalse("al", L1, "Skip block 1 if false")
	}

	// Parse stm1 in if cond ? stm1 : stm2
	err = ParseStatement(s)
	if err != nil {
		return err
	}

	if s.found(TOK_COLON) {
		if !s.HasReturned && !value.HasValue() {
			L2 = code.NewLabel()
			EmitJump(L2, "")
		}
		EmitLabel(L1, "")
		// Parse stm2 in if cond ? stm1 : stm2
		err = ParseStatement(s)
		if err != nil {
			return err
		}
		if !s.HasReturned && !value.HasValue() {
			EmitLabel(L2, "")
		}
	} else {
		EmitLabel(L1, "")
	}
	s.HasReturned = oldHasReturned
	return nil
}

// ParseIfElse will parse the code after "if cond {"
func ParseIfElse(s *State, value *ValueDef) error {
	L1, L2 := 0, 0
	s.next()
	if !value.HasValue() {
		L1 = code.NewLabel()
		EmitAssertTosInRax("Pop TOS into rax before assignment")
		EmitJumpFalse("al", L1, "Skip block 1 if false")
	}

	// Parse stm1 in "if cond { stm1 } ..."
	err := ParseBlock(s, value.IsTrue())
	if err != nil {
		return err
	}

	if !s.found(TOK_RBRACE) {
		return fmt.Errorf("expected } after if clause, but got %s", s.tokenString)
	}

	for s.found(TOK_ELSE) {
		s.tokenString = "else " + s.tokenString
		s.CollectNextLine()
		code.LineNum = code.NextLineNum
		code.EmitLineNo(s.currentLine)

		if !s.HasReturned && !value.HasValue() {
			L2 = code.NewLabel()
			EmitJump(L2, "Skip else block")
		}
		EmitLabel(L1, "")
		L1 = 0
		if s.token == TOK_IF {
			s.next()
			if len(code.ArgCode) > 0 {
				panic("ParseIfElse has len(ArgCode)>0")
			}
			code.NewArgCode()
			values, err := ParseExpression(s)
			if err != nil {
				return err
			}
			code.OutputArgCode()
			if err != nil {
				return err
			}
			if len(values) != 1 {
				return fmt.Errorf("expected one value")
			}
			if len(code.ArgCode) > 0 {
				panic("ParseIfElse has len(ArgCode)>0")
			}
			if value.Typ.Pt != code.TYP_BOOL {
				return fmt.Errorf("expected boolean but got %s", code.PrimaryTypeNames[value.Typ.Pt])
			}
			// Remove returned value from stack
			EmitAssertTosInRax("")
			L1 = code.NewLabel()
			EmitJumpFalse("al", L1, "jump if condition was false")
			if !s.found(TOK_LBRACE) {
				return fmt.Errorf("expected { after if but got %s", s.tokenString)
			}
			// Parsing 'else if' statements
			err = ParseBlock(s, value.IsFalse())
			if err != nil {
				return err
			}
			if !s.HasReturned {
				EmitJump(L2, "jump to end of else block")
			}
			if s.token != TOK_RBRACE {
				return fmt.Errorf("expected } after if clause, but got %s", s.tokenString)
			}
			if L2 != 0 {
				EmitLabel(L2, "Skipped else block")
				L2 = 0
			}
			s.next()
		} else if s.found(TOK_LBRACE) {
			// Else without if
			err = ParseBlock(s, value.IsFalse())
			if err != nil {
				return err
			}
			if s.token != TOK_RBRACE {
				return fmt.Errorf("expected } after else clause, but got %s", s.tokenString)
			}
			s.next()
		} else {
			// Else without {
			return fmt.Errorf("expected { after else but got %s", s.tokenString)
		}
	}
	if L1 != 0 {
		EmitLabel(L1, "")
	}
	if L2 != 0 {
		EmitLabel(L2, "")
	}
	return nil
}

func ParseIf(s *State) error {
	s.next()
	code.NewArgCode()
	// Parse the if condition
	values, err := ParseExpression(s)
	if err != nil {
		return err
	}
	if len(values) != 1 {
		return fmt.Errorf("expected one value")
	}
	if values[0].Typ.Pt != code.TYP_BOOL {
		return fmt.Errorf("expected boolean but got %s", code.PrimaryTypeNames[values[0].Typ.Pt])
	}
	code.OutputArgCode()
	if s.found(TOK_COLON) || s.found(TOK_QMARK) {
		return ParseColonQmark(s, values[0])
	} else if s.token == TOK_LBRACE {
		return ParseIfElse(s, values[0])
	}
	return fmt.Errorf("expected {, ? or : but got %s", s.token.Name())
}

func FreeStruct(t *TypeDef) error {
	for i, f := range t.Fields {
		if f.Pt != code.TYP_STRUCT && f.Pt != code.TYP_SLICE && f.Pt != code.TYP_STRING {
			continue
		}
		ofs := t.Offsets[i]
		EmitLoadWithOffset(ofs, "Free struct field '"+i+"' of type "+f.Name())
		lbl := code.NewLabel()
		// Check that pointer is not null
		EmitJumpFalse("rax", lbl, "")
		if f.Pt == code.TYP_STRUCT {
			err := FreeStruct(f)
			if err != nil {
				return err
			}
		} else if f.Pt == code.TYP_SLICE {
			if f.Element == nil {
				return fmt.Errorf("Struct with zero element size for '%s'", f.Name())
			}
			EmitFreeSlice(f)
		} else if f.Pt == code.TYP_STRING {
			EmitFreeString(i)
		}
		EmitLabel(lbl, "")
		EmitPopAx("")
	}
	EmitFreeStruct(t.StructSize, "Now free the struct "+t.Name()+" itself")
	return nil
}

func ParseFuncDef(s *State) error {
	s.BlockLevel++
	startLevel := s.BlockLevel
	s.next()
	code.LocalSp = 0
	if s.token != TOK_ID {
		return fmt.Errorf("expected function name but got %s", s.tokenString)
	}
	VarReset(s)
	fun := s.tokenString
	if !s.IsPkgFile {
		n := FuncCount(fun)
		if fun == "main" {
			EmitFunction(fun)
		} else if fun[0] != '-' && s.PackageName != "main" {
			EmitFunction(s.PackageName + "@" + fun + "_" + strconv.Itoa(n+1))
		} else {
			EmitFunction(fun + "_" + strconv.Itoa(n+1))
		}
	}
	s.next()
	if s.token != TOK_LPAR {
		return fmt.Errorf("expected left parenthesis but got %s", s.tokenString)
	}
	s.next()
	parList, err := ParseFormalArgList(s)
	if err != nil {
		return err
	}
	code.SetUndef()
	// Parse the return type list of the function, if any
	var returnList []*TypeDef
	if !s.found(TOK_LBRACE) {
		expectRpar := s.found(TOK_LPAR)
		for {
			ft, err := ParseType(s)
			if err != nil {
				return err
			}
			returnList = append(returnList, ft)
			if !s.found(TOK_COMMA) {
				break
			}
		}
		if expectRpar && !s.found(TOK_RPAR) {
			return fmt.Errorf("expected ) but got %s", s.tokenString)
		}
		if !s.found(TOK_LBRACE) {
			return fmt.Errorf("expected { but got %s", s.tokenString)
		}
	}
	var f *FuncDef
	f, err = AddFunc(fun, TypeListVar(parList), returnList, false, false)
	s.returnLbl = code.NewLabel()
	s.currentFuncDef = f
	if err != nil {
		return err
	}
	// Now parse all the statements in the function
	err = ParseStatements(s)
	if err != nil {
		return err
	}

	// After all the statements in the function, we must have a right-brace "}".
	if s.token != TOK_RBRACE {
		return fmt.Errorf("function definition expected ending '}' but got %s", s.tokenString)
	}
	if s.PackageName == "" && !s.HasReturned && f != nil && len(f.returnTypes) > 0 {
		return fmt.Errorf("function definition does not return a value")
	}
	if s.IsPkgFile {
		if s.token != TOK_RBRACE {
			return fmt.Errorf("expected right brace but got %s", s.tokenString)
		}
		s.next()
		EmitComment("Global function " + f.name)
		EmitExtern(f.label)
		return nil
	}

	if f.name == "main" {
		EmitComment("--------------------------------------------")
	}
	EmitLabel(s.returnLbl, "Return label for "+f.name)
	// Free local variables that have objects on the heap, if any
	mustFree := false
	for _, v := range VarDefs {
		if v.Typ == nil {
			return fmt.Errorf("variable %s must have a type", v.Name)
		}
		if !v.Destroyed && (v.Typ.Pt == code.TYP_STRING || v.Typ.Pt == code.TYP_STRUCT || v.Typ.Pt == code.TYP_SLICE) {
			mustFree = true
		}
	}

	if mustFree {
		// Save ax because it might contain the returned value of the current function definition
		EmitPushAx("Save rax before freeing local variables from " + fun)
		for _, v := range VarDefs {
			if v.Destroyed || v.IsArg {
				continue
			}
			code.SetSp()
			// DeleteLocalVar(s, v.Name)
		}
		EmitPopAx("Restore rax after freeing local variables")
	}
	// Free local variables on the stack
	err = FreeBlockVars(1)
	if err != nil {
		return err
	}
	EmitEpilogue(f.name)
	if code.LocalSp != 0 {
		fmt.Printf("Stack error - localstack=%d\n", code.LocalSp)
		EmitComment("Stack error - localstack=" + strconv.Itoa(code.LocalSp))
		return fmt.Errorf("Stack error at end of function '%s',  localstack=%d", fun, code.LocalSp)
	}
	code.OutputArgCode()
	s.next()

	if s.CommentLevel > 0 {
		return fmt.Errorf("missing end of comment")
	}
	if startLevel != s.BlockLevel {
		return fmt.Errorf("Block level wrong on exit from " + fun)
	}
	s.BlockLevel--
	s.currentFuncDef = nil
	return nil
}

func ParseTypeDef(s *State) error {
	if s.token != TOK_ID {
		return fmt.Errorf("expected id but got %s", s.tokenString)
	}
	id := s.tokenString
	if strings.Contains(id, "@") {
		// Ok
	} else if s.tokenString[0] > 'Z' {
		return fmt.Errorf("all types must start with uppercase, got %s", s.tokenString)
	}
	s.next()
	if !s.found(TOK_ASSIGN) {
		return fmt.Errorf("expected '=' but got %s", s.tokenString)
	}
	typ, err := ParseType(s)
	if err != nil {
		return err
	}
	t := TypeDef{}
	if s.PackageName != "" && s.PackageName != "main" {
		t.TypeName = Mangle(s.PackageName) + "@" + id
	} else {
		t.TypeName = id
	}
	t.Element = typ.Element
	t.Fields = make(map[string]*TypeDef)
	for k, f := range typ.Fields {
		t.Fields[k] = f
	}
	t.Offsets = make(map[string]int)
	for k, o := range typ.Offsets {
		t.Offsets[k] = o
	}
	t.Pt = typ.Pt
	t.Basic = false
	t.StructSize = typ.StructSize
	AddType(id, &t)
	return nil
}

func ParseTypeDefs(s *State) error {
	var err error
	if s.found(TOK_LPAR) {
		for !s.found(TOK_RPAR) {
			err = ParseTypeDef(s)
			if err != nil {
				break
			}
		}
	} else {
		err = ParseTypeDef(s)
	}
	return err
}

// ParseVars parses a parenthesis var declaration
func ParseVars(s *State) error {
	var err error
	if s.token == TOK_LPAR {
		s.next()
		for s.token != TOK_RPAR {
			err = ParseVar(s)
			if err != nil {
				return err
			}
		}
		s.next()
	} else {
		err = ParseVar(s)
	}
	return err
}

func ParseConsts(s *State) error {
	var err error
	if s.token == TOK_LPAR {
		s.next()
		for s.token != TOK_RPAR {
			err = ParseConst(s)
			if err != nil {
				break
			}
		}
		s.next()
	} else {
		err = ParseConst(s)
	}
	return err
}

func ParseSliceLitteral(s *State, id string, typ *TypeDef) error {
	s.next()
	lit := new(SliceLit)
	lit.name = id
	for {
		if s.token == TOK_RBRACE {
			break
		}
		hasUnaryMinus := false
		if s.token == TOK_MINUS {
			hasUnaryMinus = true
			s.next()
		}
		if typ.Element == nil {
			return fmt.Errorf("element type of slice litteral is nil")
		}
		if typ.Element.Pt.IsInteger() {
			if typ.Element.Pt == code.TYP_U64 {
				if hasUnaryMinus {
					return fmt.Errorf("unsigned integer can not have minus sign")
				}
				num, err := strconv.ParseUint(s.tokenString, 0, 64)
				if err != nil {
					x, err2 := strconv.ParseFloat(s.tokenString, 64)
					if err2 != nil || strings.Contains(s.tokenString, ".") {
						return fmt.Errorf("failed to parse integer value: %s", s.tokenString)
					}
					num = uint64(x)
				}
				lit.UintValues = append(lit.UintValues, num)
			} else {
				num, err := strconv.ParseInt(s.tokenString, 0, 64)
				if err != nil {
					return fmt.Errorf("Invalid integer: %s", s.tokenString)
				}
				if hasUnaryMinus {
					num = -num
				}
				lit.IntValues = append(lit.IntValues, num)
			}
		} else if typ.Element.Pt.IsFloat() {
			num, err := strconv.ParseFloat(s.tokenString, 64)
			if err != nil {
				return fmt.Errorf("failed to parse float value: %s", s.tokenString)
			}
			if hasUnaryMinus {
				lit.FloatValues = append(lit.FloatValues, -num)
			} else {
				lit.FloatValues = append(lit.FloatValues, num)
			}
		} else {
			return fmt.Errorf("slice lteral with wrong element type")
		}
		s.next()
		if !s.found(TOK_COMMA) {
			break
		}
	}
	lit.count = len(lit.IntValues) + len(lit.FloatValues)
	lit.size = typ.Element.Pt.Size()
	SliceLiteralDefs = append(SliceLiteralDefs, lit)
	return nil
}

// ParseVar will parse a variable or constant declaration
func ParseVar(s *State) error {
	var err error
	if s.token != TOK_ID {
		return fmt.Errorf("expected id but got %s", s.tokenString)
	}
	id := s.tokenString
	s.next()
	if s.token == TOK_LBRACK {
		s.next()
		// TODO: Parse array size
		s.next()
		if !s.found(TOK_RBRACK) {
			return fmt.Errorf("expected ], got %s", s.tokenString)
		}
	}
	typ, err := ParseType(s)
	if err != nil {
		return err
	}
	var v *VarDef
	v = AddLocalVar(s, id, typ)
	EmitAllocLocalVar("Allocate local variable " + v.Name)
	if s.token == TOK_ASSIGN {
		s.next()
		val := ""
		if s.token == TOK_MINUS {
			s.next()
			if s.token != TOK_NUM {
				return fmt.Errorf("expected int or float, got %s", s.tokenString)
			}
			val = "-" + s.tokenString
		} else if s.token == TOK_LBRACE {
			err = ParseSliceLitteral(s, id, v.Typ)
		} else {
			val = s.tokenString
		}
		if v == nil {
			return fmt.Errorf("internal error in ParseVar")
		}
		v.constValue = val
		s.next()
	}
	return nil
}

func ParseConst(s *State) error {
	if s.token != TOK_ID {
		return fmt.Errorf("expected id but got %s", s.tokenString)
	}
	id := s.tokenString
	s.next()
	// Now we should either have a type name or the next constant name
	if s.token != TOK_ID && s.token != TOK_RPAR {
		return fmt.Errorf("expected type name or constant name, but got %s", s.tokenString)
	}
	_, ok := TypeDefs[s.tokenString]
	if ok {
		typ, err1 := ParseType(s)
		LastGlobalConstType = typ
		if err1 != nil {
			return err1
		}
		v, err2 := AddGlobalConst(id, typ)
		if err2 != nil {
			return err2
		}
		if s.token == TOK_ASSIGN {
			s.next()
			val := ""
			if s.token == TOK_MINUS {
				s.next()
				if s.token != TOK_NUM {
					return fmt.Errorf("expected int or float, got %s", s.tokenString)
				}
				val = "-" + s.tokenString
			} else if s.token == TOK_LBRACE {
				err := ParseSliceLitteral(s, id, typ)
				if err != nil {
					return err
				}
			} else {
				val = s.tokenString
			}
			if v == nil {
				return fmt.Errorf("internal error in ParseVar")
			}
			v.constValue = val
			LastGlobalConstValue = val
			s.next()
		}
	} else if LastGlobalConstType != nil && LastGlobalConstType.Pt.IsInteger() {
		// It was a const name without type. Treat it as a iota const if it was an integer
		v, err2 := AddGlobalConst(id, LastGlobalConstType)
		if err2 != nil {
			return err2
		}
		x, err3 := strconv.Atoi(LastGlobalConstValue)
		if err3 != nil {
			return err3
		}
		x = x + 1
		LastGlobalConstValue = strconv.Itoa(x)
		v.constValue = LastGlobalConstValue
	}
	return nil
}
func ParseAppend(s *State) error {
	if !s.found(TOK_LPAR) {
		return fmt.Errorf("expected left parenthesis")
	}
	id := s.tokenString
	s.next()
	v, err := ParseLvalue(s, id)
	if err != nil {
		return err
	}
	if v.Typ.Pt != code.TYP_SLICE {
		return fmt.Errorf("first argument to append must be a slice")
	}
	if !v.IsIndirect {
		emit("lea", "rax", BpRel(v.Offset), "")
	}
	if !s.found(TOK_COMMA) {
		return fmt.Errorf("expected comma, got %s", s.tokenString)
	}
	length := v.Typ.Element.Size()
	// rax is now a pointer to the slice.
	EmitStartAppend(length)
	n := 0
	for {
		n++
		value, err := ParseExpression(s)
		if err != nil {
			return err
		}
		if value[0].IsConst {
			intValue, err2 := StrToConst(value[0].StringValue)
			if err2 != nil {
				return fmt.Errorf("invalid integer value: %s", value[0].StringValue)
			}
			EmitPushConst(int64(intValue), "")
		} else {
			EmitAssertTosInRax("")
		}
		EmitDoAppend(length)
		if s.token == TOK_RPAR {
			s.next()
			break
		}
		if !s.found(TOK_COMMA) {
			return fmt.Errorf("expected comma or right parantesis, got %s", s.tokenString)
		}
	}
	// Add n to length
	EmitUpdateAppendLength(n)
	return nil
}

func ParseSyscall(s *State) error {
	if !s.found(TOK_LPAR) {
		return fmt.Errorf("expected left parenthesis after 'call'")
	}
	id := s.tokenString
	AddExternal(id)
	s.next()
	if !s.found(TOK_COMMA) {
		return fmt.Errorf("expected comma, got %s", s.tokenString)
	}
	code.NewArgCode()
	values, err := ParseActualArgList(s, "syscall")
	if err != nil {
		return err
	}
	argNo := len(values)

	// Make space for return values. This code is added to the ArgCode stack.
	code.NewArgCode()
	EmitAddToSp(1, "Make space for return value from syscall")

	code.ConsArgCode(argNo+1, true)
	emit("mov", "rdi", id, "Syscall function address")
	emit("mov", "rbx", strconv.Itoa(argNo*8), "")
	emit("push", "rax", "", "Dummy for _syscall ")
	emit("call", "_syscall", "", "")
	emit("add", "rsp", strconv.Itoa(argNo*8+16), " Drop arguments")
	code.SetAx()
	return nil
}
