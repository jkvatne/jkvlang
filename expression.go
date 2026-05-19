package main

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/jkvatne/jkv/code"
)

func ParseType(s *State) (*TypeDef, error) {
	var err error
	if s.token == TOK_LBRACE {
		return nil, nil
	}
	id := s.tokenString
	if id[0] > 'Z' {
		return nil, fmt.Errorf("types must start with a capital letter: '%s'", id)
	}
	nextToken(s)
	typ, ok := TypeDefs[id]
	if !ok {
		return nil, fmt.Errorf("unknown type: %s", id)
	}
	/*
		typ := new(TypeDef)
		if s.token == TOK_LBRACK {
			nextToken(s)
			if s.token == TOK_ID {
				typ.arraySize, err = strconv.Atoi(s.tokenString)
				nextToken(s)
			}
			if s.token != TOK_RBRACK {
				return nil, fmt.Errorf("Invalid token %s", s.tokenString)
			}
			nextToken(s)
		}
	*/
	return typ, err
}

// ParseFormalParList parses the function definition and returns a list of formal parameters
func ParseFormalParList(s *State) ([]*VarDef, error) {
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
		nextToken(s)
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

func ParseArrayIndexes(s *State) error {
	// Assuming s.token==TOK_LBRACK
	for {
		nextToken(s)
		if s.token != TOK_RBRACK {
			break
		}
	}
	nextToken(s)
	return nil
}

func ParseLvalueList(s *State, id string) (lvalues []*VarDef, err error) {
	for {
		lvalue := VarDefs[id]
		if lvalue == nil {
			// We don't yet know the type, so just use nil as type
			lvalue = AddLocalVar(s, id, nil, false)
			// NB: Actual size is not known. Allocation must be delayed to the time we set the type
		}
		lvalues = append(lvalues, lvalue)
		if !s.found(TOK_COMMA) {
			break
		}
		if s.token != TOK_ID {
			break
		}
		nextToken(s)
		id = s.tokenString
	}
	return lvalues, err
}

// ParseActualArgList
// For each actual argument in the argument list, generate code in ArgCode and Value in valueList
func ParseActualArgList(s *State, f *FuncDef) (valueList []*ValueDef, err error) {
	parNo := 0
	for { // each argument in the actual argument list
		parNo++
		if s.token == TOK_RPAR {
			break
		}
		// Parse the argument and save the type of the result in the value list
		var value *ValueDef
		if parNo > 1 {
			code.PushArgCode()
		}
		value, err = ParseExpression(s)
		if err != nil {
			return nil, err
		}
		valueList = append(valueList, value)
		code.PushCleanupCode()
		if value.HasValue {
			// Constants/literals are passed as pointers on the stack by EmitPushStringLit() or EmitPushConst() or PushFloat()
			if value.Typ.Pt == TYP_STRING {
				EmitPushStringLit(value.StringLitNo, "Actual argument nr "+strconv.Itoa(parNo)+" is string literal")
				EmitPushTos(parNo, f.name)
				if f.name == "printf" {
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
			EmitPushTos(parNo, f.name)
			if f.name == "printf" {
				// We have a value on the stack (TOS). printf needs special handling.
				if value.Typ.Pt == TYP_STRING {
					EmitSkipLenCap()
					// If it was a local variable or a constant, we should not free it. (The constant case has already been handled)
					if !value.IsLocalVar {
						v := ""
						v += "   mov rax, rsp  ;  Cleanup arg " + strconv.Itoa(parNo) + "\n"
						v += "   add rax, rbx\n"
						v += "   sub rax, " + strconv.Itoa(parNo*8-8) + "\n"
						v += "   mov rax, [rax]\n"
						v += "   sub rax, 8\n" // The stack contains a C-string pointer, so adjust it back
						v += "   call _free_str\n\n"
						code.SetCleanupCode(v)
					}
				} else if value.Typ.Pt == TYP_F64 || value.Typ.Pt == TYP_F32 {
					EmitFlushRax("Float arg to printf")
				} else if value.Typ.Pt.IsInteger() {
					EmitFlushRax("Integer arg to printf")
				} else {
					return nil, fmt.Errorf("printf of arguments of type %s is not yet handled", value.Typ.Pt.Name())
				}
			} else if value.Typ.Pt.IsObject() {
				// We have a heap object pointer on top of the stack. If the formal parameter is not "in",
				// and it is the result of a function call, then we have to free it after the call.
				if !f.parameters[min(parNo, len(f.parameters))-1].IsInputType {
					// If it was a local variable or a constant, we should not free it. (The constant case has already been handled)
					if !value.IsLocalVar {
						str := fmt.Sprintf("   mov rax, rsp   ; Cleanup\n   add rax,%d\n   mov rax, [rax]\n   call _free_str   ; Call free arg %d\n", parNo*8-8, parNo)
						code.SetCleanupCode(str)
					}
				}

			} else {
				// We have a simple value on the stack. Just continue.
				slog.Debug("Simple value")
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
func ParseFuncCall(s *State, id string, returnSomething bool) ([]*ValueDef, error) {
	s.currentFuncCall = id
	EmitFlushRax("Push TOS before call")
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
	s.currentFuncCall = id
	nac := len(code.ArgCode)
	if len(values) == 0 && nac >= 1 && code.ArgCode[nac-1] == "" {
		code.ArgCode = code.ArgCode[0 : nac-1]
	}
	// Make space for return values. This code is added to the ArgCode stack.
	code.PushArgCode()
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
	var v []*ValueDef
	for _, t := range f.returnTypes {
		v = append(v, &ValueDef{Typ: t, IsReturned: true, IsTempObj: t.Pt.IsObject()})
	}
	// Function results are on stack and not in RAX.
	code.RaxIsTOS = false
	s.currentFuncCall = ""
	return v, nil
}

// ParseAssign - this might be the start of a lvalue list or a function call
func ParseAssign(s *State, id string) error {
	// Expect a list of lvalues
	lvalues, err := ParseLvalueList(s, id)
	if err != nil {
		return err
	}
	for _, v := range lvalues {
		if v.Typ == nil {
			VarDefs[v.Name].Value.Offset = EmitAllocLocalVar("Allocate local variable " + v.Name)
		}
	}

	op := s.token
	if s.found(TOK_ASSIGN, TOK_PLUS_ASGN, TOK_MINUS_ASGN, TOK_MULT_ASGN, TOK_DIV_ASGN) {
		if op == TOK_ASSIGN {
			// If there is an old object, we must free it first.
			for _, lv := range lvalues {
				if lv.MustFree && lv.Typ.Pt.IsObject() {
					err = EmitFreeLocalVariables(lv.Offset(), lv.Value.Typ.Pt, "Free "+lv.Name)
					if err != nil {
						return err
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
			if lvalues[i].Value.HasValue {
				return fmt.Errorf("%s is a constant and can not be assigned to", op.Name())
			}
			oldHasValue := lvalues[i].Value.HasValue
			err = GenerateAssignment(op, lvalues[i], value)
			if err != nil {
				return err
			}
			// Old constant values are no longer constant when assigned to.
			if oldHasValue && !value.HasValue {
				lvalues[i].Value.HasValue = false
			}
			lvalues[i].Value.IsTempObj = false
		}
		code.OutputArgCode()
	} else {
		return fmt.Errorf("unrecognized token \"%s\"", s.tokenString)
	}
	return nil
}

// ParseVarOrFunc is called for a unary function or variable.
// Called when en ID is encountered in an expression
func ParseVarOrFunc(s *State) (value *ValueDef, err error) {
	value = &ValueDef{}
	err = fmt.Errorf("unrecognized variable or function call")
	// We now have s.token == TOK_ID
	id := s.tokenString
	nextToken(s)
	if s.found(TOK_LPAR) {
		// It is a function call that should return values
		var values []*ValueDef
		values, err = ParseFuncCall(s, id, true)
		if err != nil {
			return &NoValue, err
		}
		if len(values) != 1 {
			return nil, fmt.Errorf("expected 1 value but got %d", len(values))
		}
		value = values[0]
		value.IsReturned = true
		return value, nil
	} else if s.token == TOK_LBRACK {
		// TODO: It is an array
		err = ParseArrayIndexes(s)
		return &NoValue, fmt.Errorf("arrays are not yet implemented")
	}
	// It is  a simple variable
	v, ok := VarDefs[id]
	if !ok {
		return &NoValue, fmt.Errorf("did not find variable \"%s\"", id)
	}
	if v.Typ == nil {
		return &NoValue, fmt.Errorf("no type for \"%s\"", id)
	}
	if v.Typ.Pt == TYP_NONE {
		return &NoValue, fmt.Errorf("no type for \"%s\"", id)
	}
	if !v.Value.HasValue {
		// This is a local variable, not a known constant
		if v.Name == "err" {
			EmitLoadErr()
		} else if v.Value.Typ.Pt == TYP_F64 {
			// Load value into xmm<sp>
			EmitLoadFloat64(8, v.Offset(), "Load float "+v.Name)
		} else {
			EmitLoad(v.Typ.Pt.Size(), v.Offset(), "Load variable "+v.Name)
		}
		value.IsLocalVar = true
	}
	value.Typ = v.Value.Typ
	return value, nil
}

// ParseUnary will parse a parenthesis term, a number, a string, a function call
func ParseUnary(s *State) (value *ValueDef, err error) {
	value = &ValueDef{}
	if s.token == TOK_ID {
		// An id can be either a variable or a function call. A func call must return one value
		value, err = ParseVarOrFunc(s)
	} else if s.token == TOK_LPAR {
		// Start of parenthesis term
		nextToken(s)
		EmitFlushRax("Begin parenthesis term")
		value, err = ParseExpression(s)
		EmitFlushRax("End parenthesis term")
		return value, Expect(s, TOK_RPAR)
	} else if s.token == TOK_INT {
		value, err = StringToValue(s.tokenString)
		if err != nil {
			return &NoValue, err
		}
		if value.Typ == nil {
			return &NoValue, fmt.Errorf("missing integer type")
		}
		nextToken(s)
	} else if s.token == TOK_FLOAT {
		floatLitNo := AddFloatLiteral(s.tokenFloatValue)
		value.Typ = TypeDefs["F64"]
		value.FloatValue = s.tokenFloatValue
		value.FloatLitNo = floatLitNo
		value.HasValue = true
		nextToken(s)
	} else if s.token == TOK_STRING {
		litNo := AddLiteral(s.tokenString)
		value.Typ = TypeDefs["String"]
		value.StringValue = s.tokenString
		value.StringLitNo = litNo
		value.HasValue = true
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
	} else {
		slog.Error("Unexpected", "token", s.tokenString)
		return &NoValue, fmt.Errorf("unexpected token %s", s.tokenString)
	}
	return value, err
}

func ParseProd(s *State) (value *ValueDef, err error) {
	value, err = ParseUnary(s)
	if err != nil {
		return value, err
	}
	var value2 *ValueDef
	for s.token == TOK_MULT || s.token == TOK_DIV || s.token == TOK_MOD {
		op := s.token
		nextToken(s)
		code.PushArgCode()
		value2, err = ParseUnary(s)
		if err != nil {
			return &NoValue, err
		}
		code.PushArgCode()
		value, err = GenerateOp(s, op, value, value2)
		code.ConsArgCode(3, false)
		if err != nil {
			return &NoValue, err
		}
		value.IsReturned = false
	}
	return value, nil
}

func ParseSumTerm(s *State) (*ValueDef, error) {
	// ParseProd should push rax and leave new result in rax
	value1, err := ParseProd(s)
	var value2 *ValueDef
	if err != nil {
		return &NoValue, err
	}
	if s.token == TOK_PLUS && value1.Typ.Pt == TYP_STRING {
		// Concatenation of two or more strings
		if value1.HasValue {
			EmitPushConstString(value1.StringLitNo)
		}
		// Loop through all strings that are concatenated
		for s.token == TOK_PLUS {
			nextToken(s)
			// ParseProd should push rax and leave new result in rax
			code.PushArgCode()
			value2, err = ParseProd(s)
			if err != nil {
				return &NoValue, err
			}
			if value2.HasValue {
				EmitPushStringLit(value2.StringLitNo, "Sum term push value2")
				value2.IsTempObj = false
			}
			if value2.Typ.Pt != TYP_STRING {
				return &NoValue, fmt.Errorf("String can only be concatenated with another string")
			}
			code.PushArgCode()
			EmitConcat(value1.IsTempObj, value2.IsTempObj)
			code.ConsArgCode(3, false)
			value1.IsTempObj = true
		}
		return &ValueDef{Typ: &StringType, IsTempObj: true}, nil
	}
	for s.token == TOK_PLUS || s.token == TOK_MINUS || s.token == TOK_AND || s.token == TOK_OR {
		op := s.token
		nextToken(s)
		code.PushArgCode()
		value2, err = ParseProd(s)
		if err != nil {
			return &NoValue, err
		}
		code.PushArgCode()
		value1, err = GenerateOp(s, op, value1, value2)
		code.ConsArgCode(3, false)
		if err != nil {
			return &NoValue, err
		}
		value1.IsReturned = false
	}
	return value1, nil
}

func ParseCompareTerm(s *State) (*ValueDef, error) {
	value1, err := ParseSumTerm(s)
	if err != nil {
		return &NoValue, err
	}
	if value1.Typ == nil {
		return &NoValue, fmt.Errorf("internal error, no type")
	}
	if s.token != TOK_LT && s.token != TOK_GT && s.token != TOK_EQ && s.token != TOK_GE && s.token != TOK_LE && s.token != TOK_NE {
		// Not a compare operation, return value1 immediately
		return value1, nil
	}
	value1.IsReturned = false
	op := s.token
	nextToken(s)
	code.PushArgCode()
	value2, err := ParseSumTerm(s)
	if err != nil {
		return &NoValue, err
	}
	EmitFlushRax("Push TOS value 2")
	code.PushArgCode()
	result, err := GenerateOp(s, op, value1, value2)
	code.ConsArgCode(3, false)
	if err != nil {
		return &NoValue, err
	}
	return result, err
}

func ParseExpression(s *State) (result *ValueDef, err error) {
	var value2 *ValueDef
	code.RaxIsTOS = false
	result, err = ParseCompareTerm(s)
	if err != nil {
		return &NoValue, err
	}
	if result.Typ == nil {
		return &NoValue, fmt.Errorf("expression type is nil - internal error")
	}
	endLabel := 0
	for s.token == TOK_LOG_AND || s.token == TOK_LOG_OR {
		result.IsReturned = false
		if endLabel == 0 {
			endLabel = code.NewLabel()
		}
		op := s.token
		if result.Typ.Pt != TYP_BOOL {
			return &NoValue, fmt.Errorf("%s requires boolean operands", s.tokenString)
		}
		nextToken(s)

		if op == TOK_LOG_OR {
			EmitJumpTrue(endLabel, "")
		} else if op == TOK_LOG_AND {
			EmitJumpFalse(endLabel, "")
		}

		value2, err = ParseCompareTerm(s)
		if err != nil {
			return &NoValue, err
		}
		if value2.Typ == nil {
			return &NoValue, fmt.Errorf("value2.typ is nil")
		}
		if value2.Typ.Pt != TYP_BOOL {
			return &NoValue, fmt.Errorf("%s requires boolean operands", s.tokenString)
		}
	}
	if endLabel != 0 {
		EmitLabel(endLabel, "")
	}
	if result.Typ == nil {
		return &NoValue, fmt.Errorf("value.type is nil - internal error")
	}
	return result, nil
}

// ParseExpressions will parse either a comma separated list of values,
// or a function call returning potentially many values.
// It is called from the ParseAssign() function
func ParseExpressions(s *State) (results []*ValueDef, err error) {
	var v *ValueDef
	results = make([]*ValueDef, 0, 4)
	code.PushArgCode()
	n := 0
	for {
		n++
		v, err = ParseExpression(s)
		if err != nil {
			return nil, err
		}
		results = append(results, v)
		if !s.found(TOK_COMMA) {
			break
		}
		code.PushArgCode()
	}
	code.ConsArgCode(n, false)
	return results, nil
}

func ParseBlock(s *State, isTrue bool) error {
	if isTrue {
		s.noCode++
	}
	if len(code.ArgCode) > 0 {
		panic("ParseBlock: ArgCode was not empty")
	}
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
	return nil
}

// ParseColonQmark will parse the code after '?' or ':'
func ParseColonQmark(s *State, value *ValueDef) (err error) {
	L1, L2 := 0, 0
	if !value.HasValue {
		L1 = code.NewLabel()
		EmitJumpFalse(L1, "Skip block 1 if false")
	}

	// Parse stm1 in if cond ? stm1 : stm2
	_, err = ParseStatement(s)
	if err != nil {
		return err
	}

	if s.found(TOK_COLON) {
		if !s.HasReturned && !value.HasValue {
			L2 = code.NewLabel()
			EmitJump(L2, "")
		}
		EmitLabel(L1, "")
		// Parse stm2 in if cond ? stm1 : stm2
		_, err = ParseStatement(s)
		if err != nil {
			return err
		}
		if !s.HasReturned && !value.HasValue {
			EmitLabel(L2, "")
		}
	} else {
		EmitLabel(L1, "")
	}
	return nil
}

// ParseIfElse will parse the code after "if cond {"
func ParseIfElse(s *State, value *ValueDef) (err error) {
	L1, L2 := 0, 0
	nextToken(s)
	if !value.HasValue {
		L1 = code.NewLabel()
		EmitJumpFalse(L1, "Skip block 1 if false")
	}

	// Parse stm1 in "if cond { stm1 } ..."
	err = ParseBlock(s, value.IsTrue())
	if err != nil {
		return err
	}

	if !s.found(TOK_RBRACE) {
		return fmt.Errorf("expected } after if clause, but got %s", s.tokenString)
	}

	for s.found(TOK_ELSE) {
		if !s.HasReturned && !value.HasValue {
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
			code.PushArgCode()
			value, err = ParseExpression(s)
			code.OutputArgCode()
			if err != nil {
				return err
			}
			if len(code.ArgCode) > 0 {
				panic("ParseIfElse has len(ArgCode)>0")
			}
			if value.Typ.Pt != TYP_BOOL {
				return fmt.Errorf("expected boolean but got %s", PrimaryTypeNames[value.Typ.Pt])
			}
			L1 = code.NewLabel()
			EmitJumpFalse(L1, "jump if condition was false")
			if s.token != TOK_LBRACE {
				return fmt.Errorf("expected { after if but got %s", s.tokenString)
			}
			nextToken(s)
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
		} else if s.token == TOK_LBRACE {
			nextToken(s)
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
	code.PushArgCode()
	// Parse the if condition
	value, err := ParseExpression(s)
	if err != nil {
		return err
	}
	if value.Typ.Pt != TYP_BOOL {
		return fmt.Errorf("expected boolean but got %s", PrimaryTypeNames[value.Typ.Pt])
	}
	code.OutputArgCode()
	if s.found(TOK_COLON) || s.found(TOK_QMARK) {
		return ParseColonQmark(s, value)
	} else if s.token == TOK_LBRACE {
		return ParseIfElse(s, value)
	}
	return fmt.Errorf("expected {, got %s", s.token.Name())
}

func ParseFuncDef(s *State) error {
	nextToken(s)
	code.LocalSp = 0
	if s.token != TOK_ID {
		return fmt.Errorf("expected function name but got %s", s.tokenString)
	}
	VarInit()
	fun := s.tokenString
	EmitFunction(fun)
	nextToken(s)
	if s.token != TOK_LPAR {
		return fmt.Errorf("expected left parenthesis but got %s", s.tokenString)
	}
	nextToken(s)
	s.VarCount = 0
	parList, err := ParseFormalParList(s)
	if err != nil {
		return err
	}
	code.RaxIsTOS = len(parList) > 1
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
	f, err = AddFunc(fun, parList, returnList, false)
	s.returnLbl = code.NewLabel()
	s.currentFuncDef = f
	if err != nil {
		return err
	}
	// Now parse all the statements in the function
	code.RaxIsTOS = len(parList) > 0
	s.DidReturn = false
	err = ParseStatements(s)
	if err != nil {
		return err
	}
	// CheckLocalSp(fun)

	// After all the statements in the function, we must have a right-brace "}".
	if s.token != TOK_RBRACE {
		return fmt.Errorf("function definition expected ending '}' but got %s", s.tokenString)
	}
	if !s.HasReturned && f != nil && len(f.returnTypes) > 0 {
		return fmt.Errorf("function definition does not return a value")
	}
	EmitLabel(s.returnLbl, "Return label for "+f.name)
	// Free local variables that have objects on the heap, if any
	if MustFree() {
		// Save ax because it might contain the returned value of the current function definition
		EmitPushAx("Save rax before freeing " + strconv.Itoa(len(VarDefs)) + " variables from " + fun)
		for _, v := range VarDefs {
			EmitComment("Free argument " + v.Name + " at " + strconv.Itoa(v.Offset()) + " MustFree=" + strconv.FormatBool(v.MustFree))
			if v.Value.Typ.Pt.IsObject() && v.MustFree {
				err = EmitFreeLocalVariables(v.Offset(), v.Value.Typ.Pt, "Free "+v.Name)
				if err != nil {
					return err
				}
			}
		}
		EmitPopAx("Restore rax after freeing local variables")
	}
	// Free local variables on the stack
	EmitComment("End of function. Drop local variables for " + f.name)
	EmitAddToSp(-code.LocalSp, "Drop local variables")
	EmitEpilogue(f.name)
	code.OutputArgCode()
	nextToken(s)
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
	if s.token != TOK_ASSIGN {
		return fmt.Errorf("expected \"=\" but got %s", s.tokenString)
	}
	nextToken(s)
	typ, err := ParseType(s)
	if err != nil {
		return err
	}
	AddType(id, typ)
	return nil
}

func ParseTypeDefs(s *State) error {
	var err error
	nextToken(s)
	if s.token == TOK_LPAR {
		nextToken(s)
		for s.token != TOK_RPAR {
			err = ParseTypeDef(s)
			if err != nil {
				break
			}
		}
		nextToken(s)
	} else {
		err = ParseTypeDef(s)
	}
	return err
}
