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
		if value.Typ.Pt == TYP_U8 || value.Typ.Pt == TYP_U16 || value.Typ.Pt == TYP_I16 {
			// Default to I32 when assigning smaller types to a local variable
			lvalue.SetType(&I32Type)
		} else {
			lvalue.SetType(value.Typ)
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

	// If the value is known (a compile time constant)
	if value.HasValue() {
		t := lvalue.Typ.Pt
		if t == TYP_STRUCT {
			if lvalue.FieldType != nil {
				t = lvalue.FieldType.Pt
			}
		}
		if CanAssignConst(t, value) {
			if t == TYP_STRING {
				if lvalue.IsIndirect {
					EmitFlushRax("Before AssignIndirectStrLit")
					EmitAssignIndirectStrLit(value.StringLitNo, lvalue.Typ.Pt.Size(), "")
				} else if lvalue.Typ.Pt == TYP_STRUCT {
					err = EmitOpAssignStringLitToField(lvalue.Offset(), lvalue.FieldOfs, value.StringLitNo)
				} else {
					err = EmitOpAssignString(lvalue.Offset(), value.StringLitNo)
				}
			} else if t.IsInteger() {
				if lvalue.IsIndirect {
					EmitFlushRax("Before AssignIndirectInt")
					EmitAssignIndirectInt(value.Typ.Pt.Size(), value.IntValue, "")
				} else if lvalue.Name == "err" {
					EmitStoreErr(int(value.IntValue))
				} else {
					if lvalue.Offset() == 0 {
						return fmt.Errorf("Test")
					}
					err = EmitOpAssign(op, lvalue.Offset(), lvalue.Typ.Pt.Size(), value.IntValue, "")
				}
			} else if t == TYP_F64 {
				if value.FloatLitNo == 0 {
					value.FloatLitNo = AddFloatLiteral(value.FloatValue)
					err = EmitOpAssignFloat(op, lvalue.Offset(), value.FloatLitNo, "")
				} else {
					err = EmitOpAssignFloat(op, lvalue.Offset(), value.FloatLitNo, "")
				}
			} else if t == TYP_BOOL {
				EmitStoreConst(1, value.IntValue, lvalue.Offset(), "Assign bool")
			} else {
				err = fmt.Errorf("unimplemented assignment of %s", t.Name())
			}

			if err != nil {
				return err
			}

		} else {
			return fmt.Errorf("cannot assign const to variable \"%s\"", lvalue.Name)
		}
	} else if value.Typ.Pt.IsInteger() || value.Typ.Pt == TYP_PTR {
		// The value is on the top of the stack (rax). Save it to the lvalue.
		if !code.AxIsTos() {
			EmitPopAx("Assigning TOS to lvalue")
			code.SetAx()
		}
		if lvalue.Value.Typ.Pt == TYP_STRUCT {
			EmitStoreIndirect(TokenOp[op], lvalue.Typ.Pt.Size())
		} else {
			EmitStoreToLocal(TokenOp[op], lvalue.Typ.Pt.Size(), lvalue.Offset(), "Assign int to "+lvalue.Name)
		}
		code.SetUndef()
	} else if value.Typ.Pt == TYP_F64 {
		EmitAssertTosInRax("Pop TOS into rax before assignment of F64")
		EmitStoreF64(lvalue.Offset(), "Assign F64 to "+lvalue.Name)
		code.SetUndef()
	} else if value.Typ.Pt == TYP_STRING {
		EmitAssertTosInRax("Pop TOS into rax before assignment")
		EmitStoreToLocal(TokenOp[op], lvalue.Typ.Pt.Size(), lvalue.Offset(), "Assign string to "+lvalue.Name)
		code.SetUndef()
	} else if value.Typ.Pt == TYP_STRUCT && op == TOK_ASSIGN {
		EmitAssertTosInRax("Pop TOS into rax before assignment")
		// Free old value if it exists
		if !wasNew {
			EmitFreeIfExists(lvalue.Offset(), lvalue.Typ.size, "Free if "+lvalue.Name+" exists")
		}
		EmitStoreToLocal("mov", lvalue.Typ.Pt.Size(), lvalue.Offset(), "Assign struct to "+lvalue.Name)
		code.SetUndef()
	} else if value.Typ.Pt == TYP_BOOL {
		code.SetUndef()
		EmitStoreToLocal(TokenOp[op], lvalue.Typ.Pt.Size(), lvalue.Offset(), "Assign int to "+lvalue.Name)
	} else {
		return fmt.Errorf("cannot assign to variable \"%s\"", lvalue.Name)
	}
	return nil
}

func ParseStruct(s *State, id string) (*TypeDef, error) {
	if !s.found(TOK_LBRACE) {
		return nil, fmt.Errorf("expected {, found " + s.tokenString)
	}
	t := &TypeDef{TypeName: id, Pt: TYP_STRUCT}
	t.Fields = make(map[string]*TypeDef)
	t.Offsets = make(map[string]int)
	count := 0
	for {
		fieldName := s.tokenString
		_, ok := t.Fields[fieldName]
		if ok {
			return nil, fmt.Errorf("field \"%s\" already defined", fieldName)
		}
		s.next()
		fieldTypeName := s.tokenString
		ft, ok := TypeDefs[fieldTypeName]
		if !ok {
			return nil, fmt.Errorf("unknown type \"%s\"", fieldTypeName)
		}
		count++
		t.Fields[fieldName] = ft
		// fmt.Printf("name %s, type %s\n", fieldName, fieldTypeName)
		s.next()
		if s.token == TOK_RBRACE {
			break
		}
	}
	ofs := 0
	for fn, f := range t.Fields {
		if f.Pt.Size() == 8 {
			t.Offsets[fn] = ofs
			ofs += 8
		}
	}
	for fn, f := range t.Fields {
		if f.Pt.Size() == 4 {
			t.Offsets[fn] = ofs
			ofs += 4
		}
	}
	for fn, f := range t.Fields {
		if f.Pt.Size() == 2 {
			t.Offsets[fn] = ofs
			ofs += 2
		}
	}
	for fn, f := range t.Fields {
		if f.Pt.Size() == 1 {
			t.Offsets[fn] = ofs
			ofs += 1
		}
	}
	t.size = (ofs + 7) & 0xFFFFFFF8
	s.next()
	AddType(id, t)
	return t, nil
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

// ParseStructField will evaluate the address
// Called just after dot. Token should be a field name
func ParseStructField(s *State, v *VarDef) (*VarDef, error) {
	vt := v.Typ
	v = &VarDef{Typ: vt, Value: v.Value}
	for {
		fieldName := s.tokenString
		s.next()
		ofs, ok := vt.Offsets[fieldName]
		vt = vt.Fields[fieldName]
		v.Typ = vt
		if !ok {
			return nil, fmt.Errorf("expected field name of the struct %s but but got %s", v.Name, s.tokenString)
		}
		EmitAddToRsi(ofs)
		if s.token != TOK_DOT && s.token != TOK_LBRACK {
			break
		}
		// EmitStoreIndirect(vt)  ????
		return nil, fmt.Errorf("internal error in ParseStructField")
	}
	// Now rax is the address of the value
	return v, nil
}

// ParseLvalueList parses a list of lvalues to the left of = , += etc.
// The first identifier is given in parameter id.
func ParseLvalueList(s *State, id string) (lvalues []*VarDef, err error) {
	for {
		lvalue := VarDefs[id]
		if s.found(TOK_DOT) {
			if s.token == TOK_ID {
				EmitLoadEa(lvalue.Offset())
				lvalue, err = ParseStructField(s, lvalue)
				if err != nil {
					return nil, err
				}
				lvalue.IsIndirect = true
			} else {
				return nil, fmt.Errorf("expected field name of the struct %s (after dot) but but got %s", id, s.tokenString)
			}
		} else if s.found(TOK_LBRACK) {
			// Calculate offset into rax
			// TODO
			// var v *ValueDef
			// v, err = ParseIndex(s)
			// Load variable address into SI
			// EmitLoadEa(lvalue.Offset())
			// if err != nil {
			//	return nil, err
			// }
		} else if lvalue == nil {
			// New local variable,we don't yet know the type, so just use nil
			lvalue = AddLocalVar(s, id, nil)
			// NB: Actual size is not known. Allocation must be delayed to the time we set the type
		}
		lvalues = append(lvalues, lvalue)

		if !s.found(TOK_COMMA) {
			break
		}
		if s.token != TOK_ID {
			return nil, fmt.Errorf("expected variable name after comma, but but got %s", s.tokenString)
		}
		id = s.tokenString
		nextToken(s)
	}
	for _, v := range lvalues {
		if v.Typ == nil {
			VarDefs[v.Name].Value.Offset = EmitAllocLocalVar("Allocate local variable " + v.Name)
		}
	}
	return lvalues, err
}

// ParseActualArgList
// For each actual argument in the argument list, generate code in ArgCode and Value in valueList
// Assumes the ( is consumed already
func ParseActualArgList(s *State, f *FuncDef) (valueList []*ValueDef, err error) {
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
		values, err1 := ParseExpression(s)
		if err1 != nil {
			return nil, err1
		}
		valueList = append(valueList, values...)
		code.PushCleanupCode()
		value := values[0]
		if value.HasValue() {
			// Constants/literals are passed as pointers on the stack by EmitPushStringLit() or EmitPushConst() or PushFloat()
			if value.Typ.Pt == TYP_STRING {
				EmitPushStringLit(value.StringLitNo, "Actual argument nr "+strconv.Itoa(parNo)+" is string literal")
				EmitPushTos(parNo, f.name)
				if f.name == "printf" || f.name == "print" {
					EmitSkipLenCap()
				}
			} else if value.Typ.Pt.IsInteger() {
				EmitPushConst(value.IntValue, "")
				EmitPushTos(parNo, f.name)
			} else if value.Typ.Pt == TYP_BOOL {
				if value.BoolValue {
					EmitPushConst(1, "")
				} else {
					EmitPushConst(0, "")
				}
				EmitPushTos(parNo, f.name)
			} else if value.Typ.Pt == TYP_F64 {
				EmitPushFloatLit(value.FloatLitNo)
			} else {
				// TODO: Handle F32 etc.
				return nil, fmt.Errorf("constant arguments of type %s is not yet handled", value.Typ.Pt.Name())
			}
		} else {
			if parNo == 1 && f.name == "print" {
				// Check that the first parameter is a constant string literal
				return nil, fmt.Errorf("print's first parameter must be a constant string")
			}
			EmitPushTos(parNo, f.name)
			if f.name == "printf" || f.name == "print" {
				// We have a value on the stack (TOS). printf needs special handling.
				if value.Typ.Pt == TYP_STRING {
					EmitSkipLenCap()
					// If it was a local variable or a constant, we should not free it.
					// (The constant case has already been handled)
					// But if it was a function result, it can be a pointer to a literal.
					if value.LocalVar == nil {
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
				} else if value.Typ.Pt == TYP_F64 || value.Typ.Pt == TYP_F32 {
					EmitFlushRax("Float arg to printf")
				} else if value.Typ.Pt.IsInteger() {
					EmitFlushRax("Integer arg to printf")
				} else if value.Typ.Pt == TYP_STRUCT {
					EmitFlushRax("Struct field arg to printf")
				} else if value.Typ.Pt == TYP_PTR {
					EmitFlushRax("Ptr field arg to printf")
				} else if value.Typ.Pt == TYP_BOOL {
					EmitFlushRax("Bool arg to printf")
				} else {
					return nil, fmt.Errorf("printf arguments of type %s is not yet handled", value.Typ.Pt.Name())
				}
			} else if value.Typ.Pt.IsObject() {
				// We have a heap object pointer on top of the stack. If the formal parameter is not "in",
				// and it is the result of a function call, then we have to free it after the call.
				if !f.parameters[min(parNo, len(f.parameters))-1].IsInputType {
					// If it was a local variable or a constant, we should not free it. (The constant case has already been handled)
					if value.LocalVar == nil {
						str := fmt.Sprintf("   mov rax, rsp   ; Cleanup\n   add rax,%d\n   mov rax, [rax]\n   call _free_str   ; Call free arg %d\n", parNo*8-8, parNo)
						code.SetCleanupCode(str)
					}
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
	s.currentFuncCall = id
	f := FuncDefs[id]
	if f == nil {
		s.currentFuncCall = ""
		return nil, fmt.Errorf("expected a function name, got: %s", id)
	}

	// Parse the argument list and push each arg
	// -------------------------------------------------------
	values, err := ParseActualArgList(s, f)
	if err != nil {
		s.currentFuncCall = ""
		return nil, err
	}
	if !f.VarArg && len(values) != len(f.parameters) {
		return nil, fmt.Errorf("expected %d arguments, got %d", len(f.parameters), len(values))
	}
	s.currentFuncCall = id
	nac := len(code.ArgCode)
	if len(values) == 0 && nac >= 1 && code.ArgCode[nac-1] == "" {
		code.ArgCode = code.ArgCode[0 : nac-1]
	}
	// Make space for return values. This code is added to the ArgCode stack.
	code.NewArgCode()
	EmitAddToSp(len(f.returnTypes), "Make space for return values from "+f.name)

	code.ConsArgCode(len(values)+1, true)

	// Do actual call
	// ----------------------------------
	EmitCall(id, len(values), f.builtin)

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
				if lv.Typ != nil && lv.Typ.Pt == TYP_STRING {
					// Need to have pointer in rax
					EmitLoad(8, lv.Offset(), "Load ptr to string")
					// emit("mov", "rax", BpRel(lv.Offset()), "Load ptr to string")
					EmitFreeString("Free old string when assigning new")
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
			if lvalues[i].Value.HasValue() {
				return fmt.Errorf("%s is a constant and can not be assigned to", op.Name())
			}
			oldHasValue := lvalues[i].Value.HasValue()
			err = GenerateAssignment(op, lvalues[i], value)
			if err != nil {
				return err
			}
			// Old constant values are no longer constant when assigned to.
			if oldHasValue && !value.HasValue() {
				lvalues[i].Value.IsConst = false
			}
			lvalues[i].Value.IsTempObj = value.IsTempObj
		}
		code.OutputArgCode()
	} else {
		return fmt.Errorf("expected assignment, got \"%s\"", s.tokenString)
	}
	return nil
}

func ParseTypeConversion(s *State, t1 *TypeDef) (values []*ValueDef, err error) {
	// This is a type conversion. First parse value to convert
	if code.AxIsTos() {
		EmitPushAx("ParseVarOrFunc")
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
	} else if t1.Pt == TYP_I64 && t2.Pt == TYP_F64 {
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
	ofs := v.Offset()
	EmitGetAddrOfLocal(ofs)
	return []*ValueDef{&PtrValue}, nil
}

func ParseIndex(s *State, id string) (values []*ValueDef, err error) {
	v, ok := VarDefs[id]
	if !ok {
		return nil, fmt.Errorf("did not find variable \"%s\"", id)
	}
	if v.Typ.Pt != TYP_STRING {
		return nil, fmt.Errorf("expected string or array, got %s", v.Typ.Pt.Name())
	}
	values, err = ParseExpression(s)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("expected 1 value but got %d", len(values))
	}
	if !s.found(TOK_RBRACK) {
		return nil, fmt.Errorf("expected ']' but got %s", s.tokenString)
	}
	if values[0].HasValue() {
		// Index is a constant. Push local var address
		code.SetAx()
		emit("mov", "rax", BpRel(v.Offset()), "")
		emit("add", "rax", strconv.Itoa(int(values[0].IntValue)+8), "")
		values[0].IsConst = false
		values[0].Typ = &U8Type
	} else {

	}
	if v.Typ.Pt == TYP_STRING {
		emit("mov", "al", "[rax]", "")
		emit("and", "rax", strconv.Itoa(255), "")
	}
	return values, nil
}

func ParseSimpleVar(s *State, id string) (values []*ValueDef, err error) {
	// It is  a simple variable
	var value = &ValueDef{}
	v, ok := VarDefs[id]
	if !ok {
		return nil, fmt.Errorf("did not find variable \"%s\"", id)
	}
	if v.Typ == nil {
		return nil, fmt.Errorf("no type for \"%s\"", id)
	}
	if v.Typ.Pt == TYP_NONE {
		return nil, fmt.Errorf("no type for \"%s\"", id)
	}
	if !v.Value.HasValue() {
		// This is a local variable, not a known constant
		if v.Typ == nil {
			return nil, fmt.Errorf("no type for \"%s\"", id)
		}
		if v.Name == "err" {
			EmitLoadErr()
		} else if v.Value.Typ.Pt == TYP_F64 {
			// Load value into xmm<sp>
			EmitLoadFloat(8, v.Offset(), "Load float "+v.Name)
		} else if v.Value.Typ.Pt == TYP_F32 {
			// Load value into xmm<sp>
			EmitLoadFloat(4, v.Offset(), "Load float "+v.Name)
		} else if v.Value.Typ.Pt == TYP_STRUCT {
			if s.found(TOK_DOT) {
				if s.token != TOK_ID {
					return nil, fmt.Errorf("expected field name after dot")
				}
				fn := s.tokenString
				f, isOk := v.Typ.Fields[fn]
				if !isOk {
					return nil, fmt.Errorf("field \"%s\" not found", fn)
				}
				// A struct field
				ofs, ok := v.Typ.Offsets[fn]
				if !ok {
					return nil, fmt.Errorf("field \"%s\" not found", fn)
				}
				EmitLoadField(f.Pt.Size(), v.Value.Offset, ofs)
				value.Typ = f
				s.next()
				return []*ValueDef{value}, nil
			}
			// It is a struct name. Return the address in rax
			emit("mov", "rax", BpRel(v.Value.Offset), "")
			code.SetAx()
		} else {
			EmitLoad(v.Typ.Pt.Size(), v.Offset(), "Load variable "+v.Name)
		}
		value.LocalVar = v
	} else {
		value = &v.Value
	}
	value.Typ = v.Value.Typ
	return []*ValueDef{value}, nil
}

// ParseVarOrFunc is called for a unary function or variable.
// Called when an identifier is encountered in an expression
func ParseVarOrFunc(s *State) (values []*ValueDef, err error) {
	err = fmt.Errorf("unrecognized variable or function call")
	// We now have s.token == TOK_ID
	id := s.tokenString
	nextToken(s)
	if s.found(TOK_LPAR) {
		// t1 is the type we convert to
		t1, ok := TypeDefs[id]
		if ok {
			return ParseTypeConversion(s, t1)
		} else if id == "ptr" {
			return ParsePointer(s, id)
		} else {
			return ParseFuncCall(s, id, true)
		}
	} else if s.found(TOK_LBRACK) {
		// It is an array
		return ParseIndex(s, id)
	} else {
		return ParseSimpleVar(s, id)
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
		floatLitNo := AddFloatLiteral(value.FloatValue)
		value.Typ = TypeDefs["F64"]
		value.FloatLitNo = floatLitNo
		value.IsConst = true
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
		if values[0].Typ.Pt != TYP_BOOL {
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
		if id == "String" {
			if !s.found(TOK_COMMA) {
				return nil, fmt.Errorf("new string must have a given capacity")
			}
			v, err1 := ParseExpression(s)
			if err1 != nil {
				return nil, err
			}
			if v[0].IsConst {
				EmitPushConst(v[0].IntValue, "")
			}
			EmitNewString()
		} else {
			EmitNewStruct(t)
		}
		if !s.found(TOK_RPAR) {
			return nil, fmt.Errorf("expected right parenthesis")
		}
		value.IsTempObj = true

	} else if s.token == TOK_MINUS {
		s.next()
		v, err3 := ParseUnary(s, true)
		if err3 != nil {
			return nil, err3
		}
		if len(v) != 1 {
			return nil, fmt.Errorf("unary minus on invalid value")
		}
		if v[0].HasValue() {
			v[0].IntValue = -v[0].IntValue
			v[0].FloatValue = -v[0].FloatValue
		} else {
			EmitNegate()
		}
		value = v[0]
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
	if s.token == TOK_PLUS && values1[0].Typ.Pt == TYP_STRING {
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
			if values2[0].Typ.Pt != TYP_STRING {
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
		results[0].IsReturned = false
		if endLabel == 0 {
			endLabel = code.NewLabel()
		}
		op := s.token
		if results[0].Typ.Pt != TYP_BOOL {
			return nil, fmt.Errorf("%s requires boolean operands", s.tokenString)
		}
		nextToken(s)

		if op == TOK_LOG_OR {
			EmitJumpTrue(endLabel, "")
		} else if op == TOK_LOG_AND {
			EmitJumpFalse(endLabel, "")
		}

		values2, err2 := ParseCompareTerm(s)
		if err2 != nil {
			return nil, err2
		}
		if values2[0].Typ.Pt != TYP_BOOL {
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
		EmitJumpFalse(L1, "Skip block 1 if false")
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
		EmitJumpFalse(L1, "Skip block 1 if false")
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
			if value.Typ.Pt != TYP_BOOL {
				return fmt.Errorf("expected boolean but got %s", PrimaryTypeNames[value.Typ.Pt])
			}
			L1 = code.NewLabel()
			EmitJumpFalse(L1, "jump if condition was false")
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
	if values[0].Typ.Pt != TYP_BOOL {
		return fmt.Errorf("expected boolean but got %s", PrimaryTypeNames[values[0].Typ.Pt])
	}
	code.OutputArgCode()
	if s.found(TOK_COLON) || s.found(TOK_QMARK) {
		return ParseColonQmark(s, values[0])
	} else if s.token == TOK_LBRACE {
		return ParseIfElse(s, values[0])
	}
	return fmt.Errorf("expected {, ? or : but got %s", s.token.Name())
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
	EmitFunction(fun)
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
	f, err = AddFunc(fun, parList, returnList, false, false)
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
	EmitLabel(s.returnLbl, "Return label for "+f.name)
	// Free local variables that have objects on the heap, if any
	mustFree := false
	for _, v := range VarDefs {
		if (v.Value.Typ.Pt == TYP_STRING || v.Value.Typ.Pt == TYP_STRUCT) && v.Value.IsTempObj {
			mustFree = true
		}
	}

	if mustFree {
		// Save ax because it might contain the returned value of the current function definition
		EmitPushAx("Save rax before freeing " + strconv.Itoa(len(VarDefs)) + " variables from " + fun)
		for _, v := range VarDefs {
			code.SetSp()
			if v.Value.Typ.Pt == TYP_STRING && v.Value.IsTempObj {
				EmitLoad(8, v.Offset(), "Free local variable string "+v.Name)
				EmitFreeString("")
			} else if v.Value.Typ.Pt == TYP_STRUCT && v.Value.IsTempObj {
				EmitLoad(8, v.Offset(), "Free local struct "+v.Name)
				EmitFreeStruct(v.Typ.Size(), "")
			}
		}
		EmitPopAx("Restore rax after freeing local variables")
	}
	// Free local variables on the stack
	DeleteBlockVars(s)
	EmitEpilogue(f.name)
	if code.LocalSp != 0 {
		// return fmt.Errorf("Stack error")
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
	var val string
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
		v.Value.Offset = EmitAllocLocalVar("Allocate local variable " + v.Name)
	}

	if s.token == TOK_ASSIGN {
		nextToken(s)
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
		v.Value.StringValue = val
		v.Value.IntValue = int64(s.ConstValue.Bits)
		v.Value.FloatValue = math.Float64frombits(s.ConstValue.Bits)
		nextToken(s)
	}
	return nil
}
