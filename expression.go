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
		// Add argument as local variable
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

func ParseActualArgList(s *State) (valueList []ValueDef, err error) {
	for {
		s.ArgCount++
		if s.ArgCount > len(s.ArgCode) {
			s.ArgCode = append(s.ArgCode, "")
		}
		s.ArgCode[s.ArgCount-1] = ""
		if s.token == TOK_RPAR {
			break
		}
		var value ValueDef
		value, err = ParseExpression(s)
		valueList = append(valueList, value)
		if err != nil {
			return
		}
		if value.HasValue {
			if value.Typ.Pt == TYP_STRING {
				EmitPushStringLit(s, value.StringLitNo)
			} else if value.Typ.Pt.IsInteger() {
				EmitPushConst(s, value.IntValue, "")
			} else if value.Typ.Pt == TYP_BOOL {
				if value.BoolValue {
					EmitPushConst(s, 1, "")
				} else {
					EmitPushConst(s, 0, "")
				}
			} else {
				return nil, fmt.Errorf("unknown constant: %s", value.Typ.Pt)
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

// ParseFuncCall parses a function call and its arguments
// This is the only location where arguments are evaluated
func ParseFuncCall(s *State, id string, returnSomething bool) (ValueDef, error) {
	f := FuncDefs[id]
	if f == nil {
		return NoValue, fmt.Errorf("expected a function name, got: %s", id)
	}
	// Make space for return values
	n := len(f.returnTypes)
	if n > 1 {
		EmitAddSp(s, n-1, "Make space for "+strconv.Itoa(n-1)+" extra return values in addition to AX")
	}
	// Save the starting point for arguments. Needed for nested function calls
	startArgNo := s.ArgCount
	// Parse the argument list and push each arg
	values, err := ParseActualArgList(s)
	if err != nil {
		return NoValue, err
	}
	// Now output the generated code for each argument, in reverse order
	i := len(s.ArgCode) - 1
	txt := ""
	for {
		txt += s.ArgCode[i]
		if i == startArgNo {
			break
		}
		txt += "   push rax                             ; Push argument " + strconv.Itoa(i-startArgNo+1) + " of " + id + "\n"
		s.localSp++
		i--
	}
	if startArgNo == 0 {
		// If this is a top level call, output txt
		EmitCode(s, txt)
	} else {
		// If it is a nested call, save the code
		s.ArgCode[startArgNo-1] = txt
		s.ArgCode = s.ArgCode[0:startArgNo]
	}

	s.ArgCount = startArgNo
	if f.builtin {
		id = "_" + id
	}
	EmitCall(s, id, len(values))
	if !returnSomething || len(f.returnTypes) == 0 {
		// The function call should be alone, so just continue
		return NoValue, nil
	}
	return ValueDef{Typ: f.returnTypes[0]}, nil
}

// ParseAssignOrCall - this might be the start of a lvalue list or a function call
func ParseAssignOrCall(s *State, id string) error {
	if s.found(TOK_LPAR) {
		// This is a function call that does not use any returned values (a procedure call)
		_, err := ParseFuncCall(s, id, false)
		if err != nil {
			return err
		}
		return nil
	}
	// if it was not a "(", then it must be a list of lvalues
	lvalues, err := ParseLvalueList(s, id)
	if err != nil {
		return err
	}

	op := s.token
	if s.found(TOK_ASSIGN, TOK_PLUS_ASGN, TOK_MINUS_ASGN, TOK_MULT_ASGN, TOK_DIV_ASGN) {
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
			if lvalues[i].IsConst {
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
func ParseVarOrFunc(s *State) (value ValueDef, err error) {
	// We now have s.token == TOK_ID
	id := s.tokenString
	nextToken(s)
	if s.token != TOK_LBRACK && s.token != TOK_LPAR {
		// It is  a simple variable
		v, ok := VarDefs[id]
		if !ok {
			return NoValue, fmt.Errorf("did not find variable \"%s\"", id)
		}
		if v.Typ == nil {
			return NoValue, fmt.Errorf("no type for \"%s\"", id)
		}
		if v.Typ.Pt == TYP_NONE {
			return NoValue, fmt.Errorf("no type for \"%s\"", id)
		}
		if !v.Value.HasValue && !s.RaxIsTOS || v.Offset != -8 {
			EmitLoad(s, v.Typ.Pt.Size(), v.Offset, "Load variable "+v.Name)
		}
		return v.Value, err
	} else if s.token == TOK_LBRACK {
		// It is an array
		err = ParseArrayIndexes(s)
		return NoValue, err
	} else if s.found(TOK_LPAR) {
		// It is a function call that should return values
		return ParseFuncCall(s, id, true)
	}
	return NoValue, fmt.Errorf("unrecognized variable or function call")
}

// ParseUnary will parse a parenthesis term, a number, a string, a function call
func ParseUnary(s *State) (value ValueDef, err error) {
	if s.token == TOK_ID {
		// An id can be either a variable or a function call
		value, err = ParseVarOrFunc(s)
	} else if s.token == TOK_LPAR {
		// Start of parenthesis term
		nextToken(s)
		value, err = ParseExpression(s)
		return value, Expect(s, TOK_RPAR)
	} else if s.token == TOK_INT {
		value, err = StringToValue(s.tokenString)
		if err != nil {
			return NoValue, err
		}
		if value.Typ == nil {
			return NoValue, fmt.Errorf("missing integer type")
		}
		nextToken(s)
	} else if s.token == TOK_FLOAT {
		value.Typ = TypeDefs["F64"]
		value.FloatValue = s.tokenFloatValue
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
		value = True
		nextToken(s)
	} else if s.token == TOK_FALSE {
		value = False
		nextToken(s)
	} else {
		slog.Error("Unexpected", "token", s.tokenString)
		return NoValue, fmt.Errorf("unexpected token %s", s.tokenString)
	}
	return value, err
}

func ParseProd(s *State) (value ValueDef, err error) {
	value, err = ParseUnary(s)
	if err != nil {
		return value, err
	}
	var value2 ValueDef
	for s.token == TOK_MULT || s.token == TOK_DIV || s.token == TOK_MOD {
		op := s.token
		nextToken(s)
		value2, err = ParseUnary(s)
		if err == nil {
			value, err = GenerateOp(s, op, value, value2)
		}
		if err != nil {
			return NoValue, err
		}
	}
	return value, nil
}

func ParseSumTerm(s *State) (ValueDef, error) {
	value1, err := ParseProd(s)
	if err != nil {
		return NoValue, err
	}
	if s.token == TOK_PLUS && value1.Typ.Pt == TYP_STRING {
		// Concatenation of two strings
		for s.token == TOK_PLUS || s.token == TOK_MINUS || s.token == TOK_AND || s.token == TOK_OR {
			nextToken(s)
			_value2, err := ParseProd(s)
			if err != nil {
				return NoValue, err
			}
			if _value2.Typ.Pt != TYP_STRING {
				return NoValue, fmt.Errorf("String can only be concatenated with another string")
			}
			EmitConcat(s)
			s.localSp--
		}
	} else {
		for s.token == TOK_PLUS || s.token == TOK_MINUS || s.token == TOK_AND || s.token == TOK_OR {
			op := s.token
			nextToken(s)
			value2, err := ParseProd(s)
			if err != nil {
				return NoValue, err
			}
			value1, err = GenerateOp(s, op, value1, value2)
			if err != nil {
				return NoValue, err
			}
		}
	}
	return value1, nil
}

func ParseCompareTerm(s *State) (ValueDef, error) {
	var value1, value2, result ValueDef
	var err error
	value1, err = ParseSumTerm(s)
	if err != nil {
		return NoValue, err
	}
	if value1.Typ == nil {
		return NoValue, fmt.Errorf("internal error, no type")
	}
	if s.token != TOK_LT && s.token != TOK_GT && s.token != TOK_EQ && s.token != TOK_GE && s.token != TOK_LE && s.token != TOK_NE {
		// Not a compare operation, return value1 immediately
		return value1, nil
	}
	op := s.token
	nextToken(s)
	value2, err = ParseSumTerm(s)
	if err != nil {
		return NoValue, err
	}
	result.Typ = TypeDefs["Bool"]
	return GenerateOp(s, op, value1, value2)
}

func ParseExpression(s *State) (result ValueDef, err error) {
	var value2 ValueDef
	result, err = ParseCompareTerm(s)
	if err != nil {
		return NoValue, err
	}
	if result.Typ == nil {
		return NoValue, fmt.Errorf("expression type is nil - internal error")
	}
	for s.token == TOK_LOG_AND || s.token == TOK_LOG_OR {
		if result.Typ.Pt != TYP_BOOL {
			return NoValue, fmt.Errorf("%s requires boolean operands", s.tokenString)
		}
		nextToken(s)
		value2, err = ParseCompareTerm(s)
		if err != nil {
			return NoValue, err
		}
		if value2.Typ == nil {
			return NoValue, fmt.Errorf("value2.typ is nil")
		}
		if value2.Typ.Pt != TYP_BOOL {
			return NoValue, fmt.Errorf("%s requires boolean operands", s.tokenString)
		}
		_, err = GenerateOp(s, s.token, result, value2)
		if err != nil {
			return NoValue, err
		}
	}
	if result.Typ == nil {
		return NoValue, fmt.Errorf("value.type is nil - internal error")
	}
	return result, nil
}

// ParseExpressions will parse either a comma separated list of values,
// or a function call returning potentially many values.
func ParseExpressions(s *State) (results []ValueDef, err error) {
	var v ValueDef
	results = make([]ValueDef, 0, 3)
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
				v = ValueDef{Typ: t}
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

// StartCond will increment noCode if the value is a constant equal to cond
func StartCond(s *State, value *ValueDef, cond bool) {
	if value.HasValue && (value.BoolValue != cond) {
		s.noCode++
	}
}

// EndCond will decrement noCode if the value is a constant equal to cond
func EndCond(s *State, value *ValueDef, cond bool) {
	if value.HasValue && (value.BoolValue != cond) {
		s.noCode--
		if s.noCode < 0 {
			panic("negative noCode")
		}
	}
}

func ParseIf(s *State) error {
	endLabelUsed := false
	nextToken(s)
	value, err := ParseExpression(s)
	if err != nil {
		return err
	}
	if value.Typ.Pt != TYP_BOOL {
		return fmt.Errorf("expected boolean but got %s", PrimaryTypeNames[value.Typ.Pt])
	}
	endLabel := NewLabel(s)
	elseLabel := NewLabel(s)
	if !value.HasValue {
		EmitJumpFalse(s, elseLabel, "")
	}

	if s.token == TOK_COLON || s.token == TOK_QMARK {
		nextToken(s)
		StartCond(s, &value, true)
		// Parse stm1 in if cond ? stm1 : stm2
		err = ParseStatements(s)
		if err != nil {
			return err
		}
		EndCond(s, &value, true)
		if !s.hasReturned {
			EmitJump(s, endLabel, "")
			endLabelUsed = true
		}
		if s.token == TOK_COLON {
			nextToken(s)
			EmitLabel(s, elseLabel)
			StartCond(s, &value, false)
			_, err = ParseStatement(s)
			if err != nil {
				return err
			}
			EndCond(s, &value, false)
		}

	} else if s.token == TOK_LBRACE {
		nextToken(s)
		StartCond(s, &value, true)
		EnterBlock(s)
		err = ParseStatements(s)
		ExitBlock(s)
		if err != nil {
			return err
		}
		EndCond(s, &value, true)
		if !s.hasReturned && !value.HasValue {
			EmitJump(s, endLabel, "")
			endLabelUsed = true
		}
		if s.token != TOK_RBRACE {
			return fmt.Errorf("expected } after if clause, but got %s", s.tokenString)
		}
		nextToken(s)
		for s.token == TOK_ELSE {
			EmitLabel(s, elseLabel)
			nextToken(s)
			if s.token == TOK_IF {
				nextToken(s)
				value, err = ParseExpression(s)
				if err != nil {
					return err
				}
				if value.Typ.Pt != TYP_BOOL {
					return fmt.Errorf("expected boolean but got %s", PrimaryTypeNames[value.Typ.Pt])
				}
				elseLabel = NewLabel(s)
				EmitJumpFalse(s, elseLabel, "")
				if s.token != TOK_LBRACE {
					return fmt.Errorf("expected { after if but got %s", s.tokenString)
				}
				nextToken(s)
				// Parsing 'else if' statements
				StartCond(s, &value, true)
				EnterBlock(s)
				err = ParseStatements(s)
				ExitBlock(s)
				if err != nil {
					return err
				}
				EndCond(s, &value, true)
				if !s.hasReturned {
					endLabelUsed = true
					EmitJump(s, endLabel, "")
				}
				if s.token != TOK_RBRACE {
					return fmt.Errorf("expected } after if clause, but got %s", s.tokenString)
				}
				ExitBlock(s)
				nextToken(s)
			} else if s.token == TOK_LBRACE {
				nextToken(s)
				// Else without if
				StartCond(s, &value, false)
				EnterBlock(s)
				err = ParseStatements(s)
				ExitBlock(s)
				EndCond(s, &value, false)
				nextToken(s)
			} else {
				// Else without {
				return fmt.Errorf("expected { after else but got %s", s.tokenString)
			}
		}
	} else {
		return fmt.Errorf("expected { or :  but got %s", s.tokenString)
	}
	if endLabelUsed {
		EmitLabel(s, endLabel)
	}
	return nil
}

func ParseFuncDef(s *State) error {
	nextToken(s)
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
	s.VarCount = [32]int{}
	s.level = 0
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
	s.currentFunc = f
	if err != nil {
		return err
	}
	// Now parse all the statements in the function
	s.RaxIsTOS = len(parList) > 0
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
	if !s.hasReturned {
		EmitReturn(s)
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
