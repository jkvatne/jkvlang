package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/jkvatne/jkv/code"
)

func GenerateAssignment(op Token, lvalue *VarDef, value *ValueDef) (err error) {
	// Set lvalue type if not already set. Needed for new variables.
	wasNew := false
	if lvalue.Typ == nil && op == TOK_ASSIGN {
		if value.Typ.Pt == code.TYP_U8 || value.Typ.Pt == code.TYP_U16 || value.Typ.Pt == code.TYP_I16 {
			// Default to I32 when assigning smaller types to a local variable
			lvalue.Typ = &I32Type
		} else {
			lvalue.Typ = value.Typ
			wasNew = true
		}
	}
	if lvalue.Typ == nil {
		return fmt.Errorf("new variable not allowed before op-assignment")
	}

	// Check types to see if the value can be assigned to the lvalue
	if !CanAssignToVar(lvalue, value.Typ.Pt) {
		return fmt.Errorf("assignment expected type %s but got %s", lvalue.Typ.Pt.Name(), value.Typ.Name())
	}
	EmitComment(">>> GenerateAssignment")
	// If the value is known (a compile time constant)
	if value.HasValue() {
		t := lvalue.Typ.Pt
		if t == code.TYP_STRUCT {
			if lvalue != nil {
				t = lvalue.Typ.Pt
			}
		}
		if CanAssignConst(t, value) {
			if t == code.TYP_STRING {
				if lvalue.IsIndirect {

					EmitAssignIndirectStrLit(value.StringLitNo, lvalue.Typ.Pt.Size(), "")
				} else if lvalue.Typ.Pt == code.TYP_STRUCT {
					// err = EmitOpAssignStringLitToField(lvalue.Offset(), lvalue.FieldOfs, value.StringLitNo)
					panic("Test")
				} else {
					err = EmitOpAssignString(lvalue.Offset, value.StringLitNo)
				}
			} else if t.IsInteger() {
				if lvalue.IsIndirect {
					EmitAssignIndirectConstInt(lvalue.Typ.Pt.Size(), false, value.IntValue, "Assign to field")
				} else if lvalue.Name == "err" {
					EmitStoreErr(int(value.IntValue))
				} else {
					if lvalue.Offset == 0 {
						return fmt.Errorf("GenerateAssignment with offset=0")
					}
					err = EmitOpAssign(op, lvalue.Offset, lvalue.Typ.Pt.Size(), value.IntValue, "")
				}
			} else if t == code.TYP_F64 {
				err = EmitOpAssignF64(op, lvalue.Offset, value.FloatValue, "")
			} else if t == code.TYP_F32 {
				err = EmitOpAssignF32(op, lvalue.Offset, float32(value.FloatValue), "")
			} else if t == code.TYP_BOOL {
				EmitStoreConst(1, value.IntValue, lvalue.Offset, "Assign bool")
			} else {
				err = fmt.Errorf("unimplemented assignment of %s", t.Name())
			}
			code.SetUndef()
			if err != nil {
				return err
			}

		} else {
			return fmt.Errorf("cannot assign const to variable \"%s\"", lvalue.Name)
		}
	} else if value.Typ.Pt.IsInteger() || value.Typ.Pt == code.TYP_PTR {
		// The value is in TOS/rax. Save it to the lvalue.
		EmitAssertTosInRax("Assigning TOS to lvalue, assert righthand the value is in rax")
		if lvalue.IsIndirect {
			EmitStoreIndirect(TokenOp[op], lvalue.Typ.Pt.Size())
		} else if value.Offset != 0 {
			EmitStoreToLocal(TokenOp[op], lvalue.Typ.Pt.Size(), lvalue.Offset, "Assign int to "+lvalue.Name)
		}
		code.SetUndef()
	} else if value.Typ.Pt == code.TYP_F64 {
		EmitAssertTosInRax("Pop TOS into rax before assignment of F64")
		EmitStoreF64(lvalue.Offset, "Assign F64 to "+lvalue.Name)
		code.SetUndef()
	} else if value.Typ.Pt == code.TYP_F32 {
		EmitAssertTosInRax("Pop TOS into rax before assignment of F64")
		EmitStoreF32(lvalue.Offset, "Assign F32 to "+lvalue.Name)
		code.SetUndef()
	} else if value.Typ.Pt == code.TYP_STRING {
		EmitAssertTosInRax("Pop TOS into rax before assignment of string")
		EmitStoreToLocal(TokenOp[op], lvalue.Typ.Pt.Size(), lvalue.Offset, "Assign string to "+lvalue.Name)
		code.SetUndef()
	} else if value.Typ.Pt == code.TYP_SLICE {
		EmitAssertTosInRax("Pop TOS into rax before assignment of slice")
		EmitIndirectAssignment(lvalue.Name)
	} else if value.Typ.Pt == code.TYP_STRUCT && op == TOK_ASSIGN {
		if lvalue.Offset != 0 {
			EmitAssertTosInRax("Pop TOS into rax before assignment")
			// Free old value if it exists
			if !wasNew {
				EmitFreeIfExists(lvalue.Offset, lvalue.Typ.StructSize, "Free if "+lvalue.Name+" exists")
			}
			EmitStoreToLocal("mov", lvalue.Typ.Pt.Size(), lvalue.Offset, "Assign struct to "+lvalue.Name)
			code.SetUndef()
		} else {
			EmitAssertTosInRax("Pop TOS into rax before indirect assignment")
			EmitIndirectAssignment(lvalue.Name)
		}
	} else if value.Typ.Pt == code.TYP_BOOL {
		EmitAssertTosInRax("Pop TOS into rax before assignment")
		EmitStoreToLocal(TokenOp[op], lvalue.Typ.Pt.Size(), lvalue.Offset, "Assign bool to "+lvalue.Name)
	} else {
		return fmt.Errorf("cannot assign to variable \"%s\"", lvalue.Name)
	}
	return nil
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
		nextToken(s)
	}
	if s.token != TOK_RPAR {
		return parList, fmt.Errorf("expected ')' but got %s", s.tokenString)
	}
	nextToken(s)
	return parList, nil
}

// ParseLvalue will scan up to next comma. The lvalue can consist of either a
// simple variable, a struct variable followed by a dot and field, or an indexed variable
// followed by a bracket expression. We can have a row of indexes/fields
func ParseLvalue(s *State, id string) (*VarDef, error) {
	lvalue := VarDefs[id]
	if lvalue != nil && lvalue.Destroyed {
		return nil, fmt.Errorf("cannot modify destroyed local variable \"%s\"", lvalue.Name)
	}
	var ok bool
	// Loop over field access or indexed access.
	for {
		if s.found(TOK_DOT) && lvalue.Typ.Pt == code.TYP_STRUCT && s.token == TOK_ID {
			if lvalue.IsIndirect {
				EmitLoadTosIndirect(8, lvalue.Name)
			}
			// The id was followed by a dot and a field id, indicated field access.
			fieldName := s.tokenString
			s.next()
			v := &VarDef{}
			v.Typ, ok = lvalue.Typ.Fields[fieldName]
			if !ok {
				return nil, fmt.Errorf("expected field name of the struct %s but was not found", fieldName)
			}
			v.Name = fieldName
			fieldOfs := lvalue.Typ.Offsets[fieldName]
			EmitLoadField(lvalue.Offset, lvalue.IsIndirect, fieldOfs, lvalue.Name, fieldName)
			v.IsIndirect = true
			lvalue = v
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
				return nil, fmt.Errorf("Local var address is 0")
			}

			if lvalue.IsIndirect && lvalue.Typ.Pt == code.TYP_STRING && index.IsConst {
				EmitModifyConstIndexedCharIndirect(int(index.IntValue))
			} else if lvalue.Typ.Pt == code.TYP_STRING && index.IsConst {
				if lvalue.Offset == 0 {
					return nil, fmt.Errorf("Local var address is 0")
				}
				EmitModifyConstIndexedChar(lvalue.Offset, int(index.IntValue))
			} else if lvalue.IsIndirect && lvalue.Typ.Pt == code.TYP_STRING {
				EmitModifyIndexedCharIndirect()
			} else if lvalue.Typ.Pt == code.TYP_STRING {
				EmitAssertTosInRax("")
				EmitModifyIndexedChar(lvalue.Offset)
			} else if lvalue.Typ.Pt == code.TYP_SLICE && index.IsConst {
				if !lvalue.IsIndirect {
					EmitLea(lvalue.Offset, "Load indirect var address")
				}
				EmitModifyConstIndexedSlice(int(index.IntValue), lvalue.Typ.Element.Size(), s.returnLbl)
			} else if lvalue.Typ.Pt == code.TYP_SLICE {
				if !lvalue.IsIndirect {
					EmitLea(lvalue.Offset, "Load indirect var address")
				}
				EmitModifyIndexedSlice(lvalue.Typ.Element.Size(), s.returnLbl)
			}
			// Multiply by element size
			if err != nil {
				return nil, err
			}
			if lvalue.Typ.Pt == code.TYP_STRING {
				lvalue = &VarDef{Typ: &U8Type, IsIndirect: true}
			} else if lvalue.Typ.Pt == code.TYP_SLICE {
				lvalue = &VarDef{Typ: lvalue.Typ.Element, IsIndirect: true}
			}
		} else if lvalue == nil {
			// New local variable,we don't yet know the type, so just use nil
			lvalue = AddLocalVar(s, id, nil)
			// NB: Actual size is not known. Allocation must be delayed to the time we set the type
			EmitAllocLocalVar("Assign stack space for " + id)
		} else {
			break
		}
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
	} else {
		panic("Size of element must be 1,2,4 or 8 bytes")
	}
}

// ParseLvalueList parses a list of lvalues to the left of = , += etc.
// The first identifier is given in parameter id.
func ParseLvalueList(s *State, id string) (lvalues []*VarDef, err error) {
	EmitComment(">>>>> Start parsing Lvalue List")

	// For each lvalue, separated by comma. The identifier is already in the id variable
	for {
		lvalue, err2 := ParseLvalue(s, id)
		if err2 != nil {
			return nil, err2
		}
		lvalues = append(lvalues, lvalue)
		if !s.found(TOK_COMMA) {
			break
		}
		// ALl lvalues must start with an identifier
		if s.token != TOK_ID {
			return nil, fmt.Errorf("expected variable name after comma, but but got %s", s.tokenString)
		}
		id = s.tokenString
		nextToken(s)
	}
	// Create new vardefs for new local variables with unknown type.
	for _, v := range lvalues {
		if v.Typ == nil {
			// VarDefs[v.Name].Offset = EmitAllocLocalVar("Allocate local variable " + v.Name)
		}
	}
	txt := fmt.Sprintf(">>>> ParseLvalueList done, len(lvalues)=%d, indirect=%v, offset=%d", len(lvalues), lvalues[0].IsIndirect, lvalues[0].Offset)
	EmitComment(txt)
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
			// Constants/literals are passed as pointers on the stack by EmitPushStringLit() or EmitPushConst() or PushFloat()
			if value.Typ.Pt == code.TYP_STRING {
				EmitPushStringLit(value.StringLitNo, "Actual argument nr "+strconv.Itoa(parNo)+" is string literal")
				EmitPushTos(parNo, funcname)
				if funcname == "printf" || funcname == "print" {
					EmitSkipLenCapForPrint()
				}
			} else if value.Typ.Pt.IsInteger() {
				EmitPushConst(value.IntValue, "")
				EmitPushTos(parNo, funcname)
			} else if value.Typ.Pt == code.TYP_BOOL {
				if value.BoolValue {
					EmitPushConst(1, "")
				} else {
					EmitPushConst(0, "")
				}
				EmitPushTos(parNo, funcname)
			} else if value.Typ.Pt == code.TYP_F64 {
				EmitPushF64Lit(value.FloatValue)
			} else if value.Typ.Pt == code.TYP_F32 {
				if funcname == "printf" || funcname == "print" {
					// Must convert to F64 for print/fprint
					EmitPushF64Lit(value.FloatValue)
				} else {
					EmitPushF32Lit(float32(value.FloatValue))
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
		nextToken(s)
	}
	if s.token != TOK_RPAR {
		return nil, fmt.Errorf("expected right parenthesis but got %s", s.tokenString)
	}
	// Skip the final ")"
	nextToken(s)
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
	f := FindFuncDef(id, TypeListVal(values))
	if f == nil {
		return nil, fmt.Errorf("Function %s with wrong parameters", id)
	}

	for parNo, value := range values {
		if value.Typ.Pt.IsObject() {
			// We have a heap object pointer on top of the stack. If the formal parameter is not "var",
			// and it is the result of a function call, then we have to free it after the call.
			// if !f.parameters[min(i, len(f.parameters))-1].IsInputType {
			// If it was a local variable or a constant, we should not free it.
			// TODO: FInd a better way than checking for names
			if value.IsTempObj && id != "cptr" && id != "lptr" {
				lbl := code.NewLabel()
				str := "   mov rax, rsp   ; Cleanup\n"
				str += fmt.Sprintf("   add rax,%d\n", parNo*8)
				str += "   mov rax, [rax]  ; Get pointer to content (string cap/len) on stack\n"
				str += "   mov rbx, [rax]  ; Get len/cap\n"
				str += "   shr rbx, 32\n"
				str += "   or rbx, rbx\n"
				str += "   jz .L" + strconv.Itoa(lbl) + "\n"
				str += fmt.Sprintf("   call _free_str   ; Call free arg %d of %s\n", parNo+1, id)
				str += ".L" + strconv.Itoa(lbl) + ":\n"
				code.SetCleanupCode(str)
			}
			// }
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
	EmitAddToSp(len(f.returnTypes), "** Make space for return values from "+f.name)

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

	if !returnSomething || len(f.returnTypes) == 0 {
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
		if op == TOK_ASSIGN {
			// If there is an old object, we must free it first.
			for _, lv := range lvalues {
				if lv.Typ != nil && lv.Typ.Pt == code.TYP_STRING {
					if !lv.IsIndirect {
						EmitLoad(8, lv.Offset, "Load ptr to string")
						EmitFreeString("Free old string when assigning new")
					}
				}
			}
		}
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
				return fmt.Errorf("no type for \"%s\"", id)
			}
		}
		// Assign values to lvalues
		for i, value := range values {
			err = GenerateAssignment(op, lvalues[i], value)
			if err != nil {
				return err
			}
		}
		// Destroy local variables that are pointers (destrucive read).
		for _, value := range values {
			if value.localVar != nil {
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

func ParseTypeConversion(s *State, t1 *TypeDef) (values []*ValueDef, err error) {
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
	// t2 is the type we convert from
	t2 := value.Typ
	if CanAssign(t1.Pt, t2.Pt) {
		value.Typ = t1
	} else if t1.Pt == code.TYP_I64 && t2.Pt == code.TYP_F64 {
		value.Typ = t1
	} else if (t1.Pt == code.TYP_F64 || t1.Pt == code.TYP_F32) && (t2.Pt == code.TYP_F64 || t2.Pt == code.TYP_F32) {
		value.Typ = t1
	} else {
		err = fmt.Errorf("can not convert from %s to %s", t1.Pt.Name(), t2.Pt.Name())
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

// LoadIndexedVar assumes the index value is TOS (or a constant in value.IntValue if value.IsConst
// It will multiply the index by the size (which can be 1,2,4 or 8) and add it to the address.
// This function is for local variables only, so the address is given by the offset from the frame pointer (rbp).
// We want:  RBX = (RCX * 4) + RAX + 16
// Assembly: lea rbx, [rax + rcx * 4 + 16]
func LoadIndexedVar(size int, frameOffset int, index *ValueDef) (*ValueDef, error) {
	if index.HasValue() {
		// Index is a constant. Load ea and add size*index
		EmitLoadIndexedVar(frameOffset, index.IntValue, size)
		return &ValueDef{Typ: &U8Type}, nil
	} else {
		return nil, fmt.Errorf("Not implemented")
	}
}

// ParseArrayOrStruct handles a string of array or field references.
// F.ex. a[14].f.r[i]. It emits code to load the resulting value.
// Current token is either TOK_DOT or TOK_LBRACK when this function is called
func ParseArrayOrStruct(s *State, id string) ([]*ValueDef, error) {
	var isIndirect bool
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
			if vp.IsGlobal {
				EmitLoadGlobal(id, size, int(index.IntValue), index.IsConst)
			} else {
				LoadIndexedValue(isIndirect, index.IsConst, v.Offset, index.IntValue, size)
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

// ParseVarOrFunc is called for a unary function or variable.
// Called when an identifier is encountered in an expression
// We now have s.token == TOK_ID
func ParseVarOrFunc(s *State) (values []*ValueDef, err error) {
	err = fmt.Errorf("unrecognized variable or function call")
	id := s.tokenString
	s.next()
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
		// It is an array or a struct field. Handle them in a loop in the
		// ParseArrayOrStruct function
		return ParseArrayOrStruct(s, id)

	} else {
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
			EmitLoadFloat(localVar.Typ.Size(), localVar.Offset, localVar.Name, s.currentFuncCall)
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
		nextToken(s)
		// EmitFlushRax("Begin parenthesis term")
		values, err2 := ParseExpression(s)
		if err2 != nil {
			return nil, err2
		}
		// EmitFlushRax("End parenthesis term")
		return values, Expect(s, TOK_RPAR)
	} else if s.token == TOK_INT {
		value = &ValueDef{IsConst: true}
		value.IntValue = int64(s.ConstValue.Bits)
		value.UintValue = s.ConstValue.Bits
		value.Typ = TypeDefs[s.ConstValue.Pt.Name()]
		if hasUnaryMinus {
			s.ConstValue.Bits = uint64(-int64(s.ConstValue.Bits))
		}
		if value.Typ == nil {
			return nil, fmt.Errorf("missing integer type")
		}
		nextToken(s)
	} else if s.token == TOK_FLOAT {
		if hasUnaryMinus {
			v := -math.Float64frombits(s.ConstValue.Bits)
			s.ConstValue.Bits = math.Float64bits(v)
		}
		value.FloatValue = math.Float64frombits(s.ConstValue.Bits)
		value.Typ = TypeDefs["F64"]
		value.IsConst = true
		nextToken(s)
	} else if s.token == TOK_CHAR {
		value = &ValueDef{IsConst: true}
		value.IntValue = int64(s.ConstValue.Bits)
		value.UintValue = s.ConstValue.Bits
		value.Typ = TypeDefs["U8"]
		nextToken(s)
	} else if s.token == TOK_STRING {
		litNo := AddLiteral(s.tokenString)
		value.Typ = TypeDefs["String"]
		value.StringValue = s.tokenString
		value.StringLitNo = litNo
		value.IsConst = true
		nextToken(s)
	} else if s.token == TOK_LBRACK {
		for {
			nextToken(s)
			if s.token == TOK_RBRACK {
				nextToken(s)
				break
			}
		}
	} else if s.token == TOK_TRUE {
		value = &True
		nextToken(s)
	} else if s.token == TOK_FALSE {
		value = &False
		nextToken(s)
	} else if s.token == TOK_NOT {
		nextToken(s)
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
			EmitPushConst(v[0].IntValue, "")
		}
		hasLen := false
		if s.found(TOK_COMMA) {
			// Has length also
			v2, err2 := ParseExpression(s)
			if err2 != nil {
				return nil, err
			}
			if v2[0].IsConst {
				EmitPushConst(v2[0].IntValue, "")
			}
			hasLen = true
		}
		if t.Pt == code.TYP_STRING {
			EmitNewString(hasLen)
		} else if t.Pt == code.TYP_SLICE {
			EmitNewSlice(t, t.Element.Size(), hasLen)
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
			v[0].IntValue = -v[0].IntValue
			// v[0].FloatValue = -v[0].FloatValue
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
		return nil, fmt.Errorf("unexpected token %s", s.tokenString)
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
		nextToken(s)
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
			nextToken(s)
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
		nextToken(s)
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
	nextToken(s)
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

func ParseExpression(s *State) ([]*ValueDef, error) {
	results, err := ParseCompareTerm(s)
	if err != nil {
		return nil, err
	}
	endLabel := 0
	for s.token == TOK_LOG_AND || s.token == TOK_LOG_OR {
		if len(results) != 1 {
			return nil, fmt.Errorf("+ and - can only operate on 1 value but got %d", len(results))
		}
		if results[0].IsConst {
			EmitLoadBool(results[0].BoolValue)
		}
		results[0].IsReturned = false
		if endLabel == 0 {
			endLabel = code.NewLabel()
		}
		op := s.token
		if results[0].Typ.Pt != code.TYP_BOOL {
			return nil, fmt.Errorf("%s requires boolean operands", s.tokenString)
		}
		nextToken(s)

		if op == TOK_LOG_OR {
			EmitJumpTrue("al", endLabel, "")
		} else if op == TOK_LOG_AND {
			EmitJumpFalse("al", endLabel, "")
		}

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
	if results[0].Typ == nil {
		return nil, fmt.Errorf("value.type is nil - internal error")
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
	if isTrue {
		s.noCode--
		if s.noCode < 0 {
			panic("negative noCode")
		}
	}
	DeleteBlockVars(s)
	s.BlockLevel--
	return nil
}

// ParseColonQmark will parse the code after '?' or ':'
func ParseColonQmark(s *State, value *ValueDef) (err error) {
	L1, L2 := 0, 0
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
	return nil
}

// ParseIfElse will parse the code after "if cond {"
func ParseIfElse(s *State, value *ValueDef) error {
	L1, L2 := 0, 0
	nextToken(s)
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
		if !s.HasReturned && !value.HasValue() {
			L2 = code.NewLabel()
			EmitJump(L2, "Skip else block")
		}
		EmitLabel(L1, "")
		L1 = 0
		if s.token == TOK_IF {
			nextToken(s)
			if len(code.ArgCode) > 0 {
				panic("ParseIfElse has len(ArgCode)>0")
			}
			code.NewArgCode()
			values, err := ParseExpression(s)
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
			nextToken(s)
		} else if s.found(TOK_LBRACE) {
			// Else without if
			err = ParseBlock(s, value.IsFalse())
			if err != nil {
				return err
			}
			if s.token != TOK_RBRACE {
				return fmt.Errorf("expected } after else clause, but got %s", s.tokenString)
			}
			nextToken(s)
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
	nextToken(s)
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

func FreeStruct(t *TypeDef) {
	for i, f := range t.Fields {
		if f.Pt != code.TYP_STRUCT && f.Pt != code.TYP_SLICE && f.Pt != code.TYP_STRING {
			continue
		}
		ofs := t.Offsets[i]
		EmitLoadWithOffset(ofs, "Free struct field "+f.Name())
		lbl := code.NewLabel()
		// Check that pointer is not null
		EmitJumpFalse("rax", lbl, "")
		if f.Pt == code.TYP_STRUCT {
			FreeStruct(f)
		} else if f.Pt == code.TYP_SLICE {
			EmitFreeSlice(f)
		} else if f.Pt == code.TYP_STRING {
			EmitFreeString("")
		}
		EmitLabel(lbl, "")
		EmitPopAx("")
	}
	EmitFreeStruct(t.StructSize, "")
}

func ParseFuncDef(s *State) error {
	s.BlockLevel++
	startLevel := s.BlockLevel
	nextToken(s)
	code.LocalSp = 0
	if s.token != TOK_ID {
		return fmt.Errorf("expected function name but got %s", s.tokenString)
	}
	VarReset(s)
	fun := s.tokenString
	if fun[0] != '-' && fun != "main" {
		n := FuncCount(fun)
		EmitFunction(fun + "_" + strconv.Itoa(n+1))
	} else {
		EmitFunction(fun)
	}
	nextToken(s)
	if s.token != TOK_LPAR {
		return fmt.Errorf("expected left parenthesis but got %s", s.tokenString)
	}
	nextToken(s)
	s.LocalVarCount = 0
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
	s.DidReturn = false
	err = ParseStatements(s)
	if err != nil {
		return err
	}

	// After all the statements in the function, we must have a right-brace "}".
	if s.token != TOK_RBRACE {
		return fmt.Errorf("function definition expected ending '}' but got %s", s.tokenString)
	}
	if !s.HasReturned && f != nil && len(f.returnTypes) > 0 {
		// return fmt.Errorf("function definition does not return a value")
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
			if v.Typ.Pt == code.TYP_STRING {
				EmitLoad(8, v.Offset, "Free local variable string '"+v.Name+"'")
				EmitFreeString("")
			} else if v.Typ.Pt == code.TYP_STRUCT {
				// Load local var pointer into rax
				EmitLoad(8, v.Offset, "Free local struct "+v.Name)
				FreeStruct(v.Typ)
			} else if v.Typ.Pt == code.TYP_SLICE && !v.IsGlobal {
				// Load local var pointer into rax
				EmitLoad(8, v.Offset, "Free local slice "+v.Name)
				EmitFreeSlice(v.Typ)
			}
		}
		EmitPopAx("Restore rax after freeing local variables")
	}
	// Free local variables on the stack
	DeleteBlockVars(s)
	EmitEpilogue(f.name)
	if code.LocalSp != 0 {
		fmt.Printf("Stack error - localstack=%d\n", code.LocalSp)
		EmitComment("Stack error - localstack=" + strconv.Itoa(code.LocalSp))
		return fmt.Errorf("Stack error at end of %s,  localstack=%d", fun, code.LocalSp)
	}
	code.OutputArgCode()
	nextToken(s)
	// Delete parameters
	for _, p := range parList {
		DeleteLocalVar(s, p.Name)
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
	if s.tokenString[0] > 'Z' {
		return fmt.Errorf("all types must start with uppercase, got %s", s.tokenString)
	}
	id := s.tokenString
	nextToken(s)
	if !s.found(TOK_ASSIGN) {
		return fmt.Errorf("expected \"=\" but got %s", s.tokenString)
	}
	typ, err := ParseType(s)
	if err != nil {
		return err
	}
	typ.TypeName = id
	AddType(id, typ)
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
			err = ParseVar(s, false)
			if err != nil {
				return err
			}
		}
		s.next()
	} else {
		err = ParseVar(s, false)
	}
	return err
}

func ParseConsts(s *State) error {
	var err error
	if s.token == TOK_LPAR {
		s.next()
		for s.token != TOK_RPAR {
			err = ParseVar(s, true)
			if err != nil {
				break
			}
		}
		s.next()
	} else {
		err = ParseVar(s, true)
	}
	return err
}

// ParseVar will parse a variable or constant declaration
func ParseVar(s *State, isGlobal bool) error {
	var err error
	if s.token != TOK_ID {
		return fmt.Errorf("expected id but got %s", s.tokenString)
	}
	id := s.tokenString
	nextToken(s)
	if s.token == TOK_LBRACK {
		nextToken(s)
		// TODO: Parse array size
		nextToken(s)
		if !s.found(TOK_RBRACK) {
			return fmt.Errorf("expected ], got %s", s.tokenString)
		}
	}
	typ, err := ParseType(s)
	if err != nil {
		return err
	}
	var v *VarDef
	if isGlobal {
		v, err = AddGlobalConst(id, typ)
		if err != nil {
			return err
		}
	} else {
		v = AddLocalVar(s, id, typ)
		v.Offset = EmitAllocLocalVar("Allocate local variable " + v.Name)
	}

	if s.token == TOK_ASSIGN {
		nextToken(s)
		val := ""
		if s.token == TOK_MINUS {
			nextToken(s)
			if s.token != TOK_INT && s.token != TOK_FLOAT {
				return fmt.Errorf("expected int or float, got %s", s.tokenString)
			}
			val = "-" + s.tokenString
			if s.ConstValue.Pt.IsFloat() {
				f := -math.Float64frombits(s.ConstValue.Bits)
				s.ConstValue.Bits = math.Float64bits(f)
			} else {
				s.ConstValue.Bits = uint64(-int64(s.ConstValue.Bits))
			}
		} else {
			val = s.tokenString
		}
		if v == nil {
			return fmt.Errorf("internal error in ParseVar")
		}
		v.constValue = val
		nextToken(s)
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
		return fmt.Errorf("expected indirect value")
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
			EmitPushConst(value[0].IntValue, "")
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
	emit("push", "rax", "", "Dummy for _syscall "+Sp(1))
	emit("call", "_syscall", "", "")
	emit("add", "rsp", strconv.Itoa(argNo*8+16), Sp(-argNo-2)+" Drop arguments")
	code.SetAx()
	return nil
}
