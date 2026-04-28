package main

import (
	"fmt"
	"log/slog"
	"strconv"
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

// ParseFormalParList parses the function definition and retrurns a list of formal parameters
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

func OutputCleanupCode(s *State, startArgNo int) {
	txt := ""
	for i := len(s.CleanupCode) - 1; i >= startArgNo; i-- {
		txt += s.CleanupCode[i]
	}
	s.CleanupCode[startArgNo] = txt
	s.CleanupCode = s.CleanupCode[0 : startArgNo+1]
}

func OutputArgCode(s *State, startArgNo int, values []*ValueDef) {
	// Now output the generated code for each argument, in reverse order
	txt := ""
	for i := len(s.ArgCode) - 1; i >= startArgNo; i-- {
		txt += s.ArgCode[i]
		if len(values) > 0 {
			if values[i-startArgNo].Typ.Pt == TYP_F64 {
				txt += "   movq rax, xmm0\n"
			}
			if i > startArgNo {
				txt += "   push rax\n"
				s.localSp++
			}
		}
	}
	s.ArgCode[startArgNo] = txt
	s.ArgCode = s.ArgCode[0 : startArgNo+1]
}

// ParseActualArgList
// For each actual argument in the argument list, generate code in ArgCode and Value in valueList
func ParseActualArgList(s *State, f *FuncDef) (startArgNo int, valueList []*ValueDef, floatParCount int, err error) {
	// Make sure we have an empty last entry in ArgCode. Will exist for nested functions.
	if len(s.ArgCode) == 0 {
		s.ArgCode = append(s.ArgCode, "")
		s.CleanupCode = append(s.CleanupCode, "")
	}
	if s.nesting == 0 {
		s.ArgCode[0] = ""
		s.CleanupCode[0] = ""
	}
	if s.ArgCode[len(s.ArgCode)-1] != "" {
		panic("ArgCode[i] should be blank")
	}
	// Save the starting point for arguments. Needed for nested function calls
	startArgNo = len(s.ArgCode) - 1
	if len(s.ArgCode) > 0 {
		// Remove the last entry in ArgCode. A new will be added by ParseActualArgList
		s.ArgCode = s.ArgCode[:len(s.ArgCode)-1]
		s.CleanupCode = s.CleanupCode[:len(s.CleanupCode)-1]
	}
	parNo := 0
	for { // each agrument in the actual argument list
		parNo++
		s.RaxIsTOS = false
		// A new argument. Append "" to the ArgCode slice
		s.ArgCode = append(s.ArgCode, "")
		s.CleanupCode = append(s.CleanupCode, "")
		if s.token == TOK_RPAR {
			break
		}

		// Parse the argument and save the type of the result in the value list
		var value *ValueDef
		value, err = ParseExpression(s)
		if err != nil {
			return 0, nil, 0, err
		}
		valueList = append(valueList, value)

		if value.HasValue {
			// Constants/literals are passed as pointers on the stack by EmitPushStringLit() or EmitPushConst() or PushFloat()
			if value.Typ.Pt == TYP_STRING {
				EmitPushStringLit(s, value.StringLitNo, "Actual argument is string literal")
			} else if value.Typ.Pt.IsInteger() {
				EmitPushConst(s, value.IntValue, "")
			} else if value.Typ.Pt == TYP_BOOL {
				if value.BoolValue {
					EmitPushConst(s, 1, "")
				} else {
					EmitPushConst(s, 0, "")
				}
			} else if value.Typ.Pt == TYP_F64 {
				floatParCount++
				EmitPushFloat(s, value.FloatLitNo)
			} else {
				// TODO: Handle F32 etc.
				return 0, nil, 0, fmt.Errorf("Constant arguments of type %s is not yet handled", value.Typ.Pt)
			}
		} else {
			if f.name == "printf" {
				// We have a value on the stack (TOS). printf needs special handling.
				if value.Typ.Pt == TYP_STRING && parNo > 1 {
					EmitSkipLenCap(s)
				} else if value.Typ.Pt == TYP_F64 || value.Typ.Pt == TYP_F32 {
					emit(s, "movq", "rax", xmm(s.XmmSp-1), "printf argument")
				} else {
					return 0, nil, 0, fmt.Errorf("prinf of arguments of type %s is not yet handled", value.Typ.Pt)
				}
			} else if value.Typ.Pt.IsObject() {
				// We have a heap object pointer on top of the stack. If the formal parameter is not "in",
				// and it is the result of a function call, then we have to free it after the call.
				if !f.parameters[min(parNo, len(f.parameters))-1].IsInputType {
					// If it was a local variable or a constant, we should not free it. (The constant case has already been handled)
					if !value.IsLocalVar {
						// value.Offset = EmitAllocLocalVar(s, "Temporary variable for parameter "+strconv.Itoa(parNo))
						s.CleanupCode[len(s.CleanupCode)-1] = "; Call free"
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
		return 0, nil, 0, fmt.Errorf("expected right parenthesis but got %s", s.tokenString)
	}
	// Skip the final ")"
	nextToken(s)
	OutputArgCode(s, startArgNo, valueList)
	return startArgNo, valueList, floatParCount, nil
}

// ParseFuncCall parses a function call and its arguments
// This is the only location where arguments are evaluated
func ParseFuncCall(s *State, id string, returnSomething bool) ([]*ValueDef, error) {
	if s.RaxIsTOS {
		emit(s, "push", "rax", "", "")
	}

	f := FuncDefs[id]
	if f == nil {
		return nil, fmt.Errorf("expected a function name, got: %s", id)
	}

	s.nesting++

	// Make space for return values
	n := len(f.returnTypes)
	if n > 1 {
		EmitAddSp(s, n-1, "Make space for "+strconv.Itoa(n-1)+" extra return values in addition to AX")
	}

	// Parse the argument list and push each arg
	// -------------------------------------------------------
	startArgNo, values, floatParCount, err := ParseActualArgList(s, f)
	if err != nil {
		return nil, err
	}

	// Do actual call
	// ----------------------------------
	EmitCall(s, id, len(values), f.builtin)

	OutputCleanupCode(s, startArgNo)
	EmitSubStack(s, len(values))

	s.nesting--
	s.XmmSp -= floatParCount
	if s.nesting == 0 {
		_, _ = Write(s, s.ArgCode[0], true)
		s.ArgCode[0] = ""
	}
	if !returnSomething || len(f.returnTypes) == 0 {
		// The function call should be alone, so just continue
		return nil, nil
	}
	s.RaxIsTOS = true
	var v []*ValueDef
	for _, t := range f.returnTypes {
		v = append(v, &ValueDef{Typ: t, IsReturned: true, IsTempObj: t.Pt.IsObject()})
	}
	return v, nil
}

// ParseAssign - this might be the start of a lvalue list or a function call
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
				return fmt.Errorf("no type for \"%s\"", id)
			}
		}
		// Assign values to lvalues
		for i, value := range values {
			if lvalues[i].Value.HasValue {
				return fmt.Errorf("%s is a constant and can not be assigned to", op.Name())
			}
			oldHasValue := lvalues[i].Value.HasValue
			err = GenertateAssignment(s, op, lvalues[i], value)
			if err != nil {
				return err
			}
			// Old constant values are no longer constant when assigned to.
			if oldHasValue && !value.HasValue {
				lvalues[i].Value.HasValue = false
			}
		}
	} else {
		return fmt.Errorf("unrecognized token \"%s\"", s.tokenString)
	}
	return nil
}

// ParseVarOrFunc is called for a unary function or variable.
// Called when en ID is encountered in an expression
func ParseVarOrFunc(s *State) (value *ValueDef, err error) {
	value = &NoValue
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
		return &NoValue, fmt.Errorf("Arras are not yet implemented")
	} else {
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
				emit(s, "mov", "rax", "r15", "Load err")
				s.RaxIsTOS = true
			} else if v.Value.Typ.Pt == TYP_F64 {
				// Load value into xmm<sp>
				EmitLoadFloat64(s, 8, v.Offset(), "Load float "+v.Name)
			} else {
				EmitLoad(s, v.Typ.Pt.Size(), v.Offset(), "Load variable "+v.Name)
			}
			v.Value.IsLocalVar = true
		}
		return &v.Value, nil
	}
}

// ParseUnary will parse a parenthesis term, a number, a string, a function call
func ParseUnary(s *State) (value *ValueDef, err error) {
	value = &ValueDef{}
	if s.token == TOK_ID {
		// An id can be either a variable or a function call. A func call must returne one value
		value, err = ParseVarOrFunc(s)
	} else if s.token == TOK_LPAR {
		// Start of parenthesis term
		nextToken(s)
		value, err = ParseExpression(s)
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
		value2, err = ParseUnary(s)
		if err == nil {
			value, err = GenerateOp(s, op, value, value2)
		}
		value.IsReturned = false
		if err != nil {
			return &NoValue, err
		}
	}
	return value, nil
}

func FreeTemporaryObject(s *State, value *ValueDef) {
	if value.IsTempObj {
		EmitComment(s, "Free temporary object ")
	}
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
			EmitPushConstString(s, value1.StringLitNo)
		}
		// Loop through all strings that are concatenated
		for s.token == TOK_PLUS {
			nextToken(s)
			// ParseProd should push rax and leave new result in rax
			value2, err = ParseProd(s)
			if err != nil {
				return &NoValue, err
			}
			if value2.HasValue {
				if s.RaxIsTOS {
					s.localSp++
					emit(s, "push", "rax", "", "")
				}
				// Push constant string
				emit(s, "mov", "rax", "str"+strconv.Itoa(value2.StringLitNo), "")
			}
			if value2.Typ.Pt != TYP_STRING {
				return &NoValue, fmt.Errorf("String can only be concatenated with another string")
			}
			EmitConcat(s, value1.IsTempObj, value2.IsTempObj)
		}
		return &ValueDef{Typ: &StringType, IsTempObj: true}, nil
	} else {
		for s.token == TOK_PLUS || s.token == TOK_MINUS || s.token == TOK_AND || s.token == TOK_OR {
			op := s.token
			nextToken(s)
			value2, err = ParseProd(s)
			if err != nil {
				return &NoValue, err
			}
			value1, err = GenerateOp(s, op, value1, value2)
			if err != nil {
				return &NoValue, err
			}
			value1.IsReturned = false
		}
		return value1, nil
	}
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
	value2, err := ParseSumTerm(s)
	if err != nil {
		return &NoValue, err
	}
	result, err := GenerateOp(s, op, value1, value2)
	if err != nil {
		return &NoValue, err
	}
	// Free possible temporary objects in value1 or value2
	FreeTemporaryObject(s, value1)
	FreeTemporaryObject(s, value2)
	return result, err
}

func ParseExpression(s *State) (result *ValueDef, err error) {
	var value2 *ValueDef
	result, err = ParseCompareTerm(s)
	if err != nil {
		return &NoValue, err
	}
	if result.Typ == nil {
		return &NoValue, fmt.Errorf("expression type is nil - internal error")
	}
	endlbl := 0
	for s.token == TOK_LOG_AND || s.token == TOK_LOG_OR {
		result.IsReturned = false
		if endlbl == 0 {
			endlbl = NewLabel(s)
		}
		op := s.token
		if result.Typ.Pt != TYP_BOOL {
			return &NoValue, fmt.Errorf("%s requires boolean operands", s.tokenString)
		}
		nextToken(s)

		if op == TOK_LOG_OR {
			EmitJumpTrue(s, endlbl, "")
		} else if op == TOK_LOG_AND {
			EmitJumpFalse(s, endlbl, "")
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
	if endlbl != 0 {
		EmitLabel(s, endlbl, "")
	}
	if result.Typ == nil {
		return &NoValue, fmt.Errorf("value.type is nil - internal error")
	}
	return result, nil
}

// ParseExpressions will parse either a comma separated list of values,
// or a function call returning potentially many values.
func ParseExpressions(s *State) (results []*ValueDef, err error) {
	var v *ValueDef
	results = make([]*ValueDef, 0, 3)
	// expectRpar := s.found(TOK_LPAR)
	for {
		id := s.tokenString
		f := FuncDefs[s.tokenString]
		if s.token == TOK_ID && f != nil && len(f.returnTypes) > 0 {
			nextToken(s)
			if !s.found(TOK_LPAR) {
				return nil, fmt.Errorf("expected ( after function, found: %s", s.tokenString)
			}
			_, err = ParseFuncCall(s, id, true)
			if err != nil {
				return nil, err
			}
			for _, t := range f.returnTypes {
				v = &ValueDef{Typ: t}
				results = append(results, v)
			}
		} else {
			v, err = ParseExpression(s)
			if err != nil {
				return nil, err
			}
			results = append(results, v)
		}
		if !s.found(TOK_COMMA) {
			break
		}
	}
	return results, nil
}

func NewLabel(s *State) int {
	s.labelNo++
	return s.labelNo
}

func ParseBlock(s *State, isTrue bool) error {
	if isTrue {
		s.noCode++
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
		L1 = NewLabel(s)
		EmitJumpFalse(s, L1, "Skip block 1 if false")
	}

	// Parse stm1 in if cond ? stm1 : stm2
	_, err = ParseStatement(s)
	if err != nil {
		return err
	}

	if s.found(TOK_COLON) {
		if !s.hasReturned && !value.HasValue {
			L2 = NewLabel(s)
			EmitJump(s, L2, "")
		}
		EmitLabel(s, L1, "")
		// Parse stm2 in if cond ? stm1 : stm2
		_, err = ParseStatement(s)
		if err != nil {
			return err
		}
		if !s.hasReturned && !value.HasValue {
			EmitLabel(s, L2, "")
		}
	} else {
		EmitLabel(s, L1, "")
	}
	return nil
}

// ParseIfElse will parse the code after "if cond {"
func ParseIfElse(s *State, value *ValueDef) (err error) {
	L1, L2 := 0, 0
	nextToken(s)
	if !value.HasValue {
		L1 = NewLabel(s)
		EmitJumpFalse(s, L1, "Skip block 1 if false")
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
		if !s.hasReturned && !value.HasValue {
			L2 = NewLabel(s)
			EmitJump(s, L2, "Skip else block")
		}
		EmitLabel(s, L1, "")
		L1 = 0
		if s.token == TOK_IF {
			nextToken(s)
			value, err = ParseExpression(s)
			if err != nil {
				return err
			}
			if value.Typ.Pt != TYP_BOOL {
				return fmt.Errorf("expected boolean but got %s", PrimaryTypeNames[value.Typ.Pt])
			}
			L1 = NewLabel(s)
			EmitJumpFalse(s, L1, "jump if condition was false")
			if s.token != TOK_LBRACE {
				return fmt.Errorf("expected { after if but got %s", s.tokenString)
			}
			nextToken(s)
			// Parsing 'else if' statements
			err = ParseBlock(s, value.IsFalse())
			if err != nil {
				return err
			}
			if !s.hasReturned {
				EmitJump(s, L2, "jump to end of else block")
			}
			if s.token != TOK_RBRACE {
				return fmt.Errorf("expected } after if clause, but got %s", s.tokenString)
			}
			if L2 != 0 {
				EmitLabel(s, L2, "Skipped else block")
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
		EmitLabel(s, L1, "")
	}
	if L2 != 0 {
		EmitLabel(s, L2, "")
	}
	return nil
}

func ParseIf(s *State) error {
	nextToken(s)

	// Parse the if condition
	value, err := ParseExpression(s)
	if err != nil {
		return err
	}
	if value.Typ.Pt != TYP_BOOL {
		return fmt.Errorf("expected boolean but got %s", PrimaryTypeNames[value.Typ.Pt])
	}

	if s.found(TOK_COLON) || s.found(TOK_QMARK) {
		return ParseColonQmark(s, value)
	} else if s.token == TOK_LBRACE {
		return ParseIfElse(s, value)
	} else {
		return fmt.Errorf("Expected {, got %s", s.token.Name())
	}

}

func ParseFuncDef(s *State) error {
	nextToken(s)
	s.localSp = 0
	if s.token != TOK_ID {
		return fmt.Errorf("expected function name but got %s", s.tokenString)
	}
	VarInit()
	fun := s.tokenString
	EmitFunction(s, fun)
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
	s.RaxIsTOS = len(parList) > 1
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
	s.returnLbl = NewLabel(s)
	s.currentFunc = f
	if err != nil {
		return err
	}
	// Now parse all the statements in the function
	s.RaxIsTOS = len(parList) > 0
	s.DidReturn = false
	err = ParseStatements(s)
	if err != nil {
		return err
	}
	// After all the statements in the function, we must have a right-brace "}".
	if s.token != TOK_RBRACE {
		return fmt.Errorf("function definition expected ending '}' but got %s", s.tokenString)
	}
	if !s.hasReturned && f != nil && len(f.returnTypes) > 0 {
		return fmt.Errorf("function definition does not return a value")
	}
	EmitLabel(s, s.returnLbl, "Return label for "+f.name)
	// Free local variables that have objects on the heap, if any
	if MustFree() {
		// Save ax because it might contain the returned value of the current function definition
		emit(s, "push", "rax", "", "Save rax")
		for _, v := range VarDefs {
			if v.Value.Typ.Pt.IsObject() && v.MustFree {
				// EmitComment(s, "Free argument "+v.Name+" at "+strconv.Itoa(v.Offset()))
				err = EmitFreeLocalVariables(s, v.Offset(), v.Value.Typ.Pt, "Free "+v.Name)
				if err != nil {
					return err
				}
			}
		}
		emit(s, "pop", "rax", "", "Restore rax")
	}
	// Set return values if more than one. If only one, it is already in rax
	if s.LocalRetSize > 1 {
		for range len(s.currentFunc.returnTypes) - 1 {
			// TODO emit(s, "pop", "rax", "", "Return value no "+strconv.Itoa(i))
			s.localSp--
		}
	}
	// Remove local variables from stack
	if s.localSp > 0 {
		emit(s, "add", "rsp", strconv.Itoa(s.localSp*8), "")
		s.localSp = 0
	}

	// Return exit code from main
	if s.currentFunc.name == "main" {
		EmitPrintSp(s)
		// Print remaining allocation
		emit(s, "mov", "rax", "[allocation_count]", "")
		emit(s, "push", "rax", "", "")
		emit(s, "mov", "rax", "alloc_size_str", "")
		emit(s, "mov", "rbx", "8", "")
		emit(s, "call", "_printf", "", "")
		emit(s, "call", "_fflush", "", "")
		emit(s, "mov", "rax", "r15", "Get error code")
		emit(s, "call", "_exit", "", "")
	} else {
		// Function epilogue. Restore frame pointer and exit
		emit(s, "leave", "", "", "")
		emit(s, "ret", "", "", "return from "+s.currentFunc.name)
	}

	nextToken(s)
	s.currentFunc = nil
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
