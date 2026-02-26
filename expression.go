package main

import (
	"fmt"
	"log/slog"
	"math"
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

func ParseFormalArgList(s *State) ([]*TypeDef, error) {
	var argList []*TypeDef
	for {
		if s.token == TOK_RPAR {
			break
		}
		if s.token != TOK_ID {
			return argList, fmt.Errorf("expected argument name but got %s", s.tokenString)
		}
		id := s.tokenString
		nextToken(s)
		typ, err := ParseType(s)
		if err != nil {
			return argList, err
		}
		if typ == nil {
			return argList, fmt.Errorf("expected argument type but got nil")
		}
		// Add argument as local variable
		AddVar(id, typ, false)
		argList = append(argList, typ)
		if s.token == TOK_RPAR {
			break
		}
		if s.token != TOK_COMMA {
			return argList, fmt.Errorf("expected comma or right parenthesis but got %s", s.tokenString)
		}
		nextToken(s)
	}
	if s.token != TOK_RPAR {
		return argList, fmt.Errorf("expected ')' but got %s", s.tokenString)
	}
	nextToken(s)
	return argList, nil
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

func ParseArgumentList(s *State) (valueList []ValueDef, err error) {
	for {
		if s.token == TOK_RPAR {
			break
		}
		var value ValueDef
		value, err = ParseExpression(s)
		valueList = append(valueList, value)
		if err != nil {
			return
		}
		if value.hasValue {
			EmitPushConst(s, value)
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
			lvalue = AddVar(id, nil, false)
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

func DoAssignment(s *State, op Token, lvalue *VarDef, value ValueDef) error {
	// Set lvalue type if not already set. Needed for new variables.
	if lvalue.typ == nil {
		lvalue.typ = value.typ
		lvalue.value.typ = value.typ
	}
	// If the value is known (a compile time constant), we copy it into the lvalue
	if value.hasValue {
		if CanAssignConst(lvalue.typ.pt, value) {
			lvalue.value = value
			// The lvalue was a constant and was not on the stack, so we push it
			EmitPushConst(s, value)
		} else {
			return fmt.Errorf("cannot assign to variable \"%s\"", lvalue.name)
		}
	}
	// Check if assignment is possible
	if !CanAssign(lvalue.typ.pt, value.typ.pt) {
		return fmt.Errorf("expected type %s but got %s for %s", lvalue.typ.pt.Name(), value.typ.Name(), lvalue.name)
	}
	// Top of stack contains the value as a 64-bit number. Store it to the local variable with correct type
	if op == TOK_ASSIGN {
		EmitStore(s, lvalue.name, value.typ.pt.Name())
	} else {
		EmitModify(s, lvalue.name, op, value.typ.pt.Name())
	}
	return nil
}

func ParseFunctionCall(s *State, id string) error {
	f := FuncDefs[id]
	if f != nil {
		// Push 0 to make space for return values
		if len(f.returnTypes) > 0 {
			EmitComment(s, "Make space for return values from "+id)
		}
		for range len(f.returnTypes) {
			EmitPushConst(s, ZeroValue)
		}
		// Parse the argument list and push each arg
		values, err := ParseArgumentList(s)
		EmitCall(s, id, len(values))
		// The function call should be alone, so just continue
		return err
	}
	return fmt.Errorf("expected a function name, got: %s", id)
}

// ParseAssignOrCall - this might be the start of a lvalue list or a function call
func ParseAssignOrCall(s *State, id string) error {
	if s.found(TOK_LPAR) {
		// This is a function call
		err := ParseFunctionCall(s, id)
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
			if value.typ == nil {
				return fmt.Errorf("no type for \"%s\"", id)
			}
		}
		// Assign values to lvalues
		for i, value := range values {
			if lvalues[i].isConst {
				return fmt.Errorf("%s is a constant and can not be assigned to", op.Name())
			}
			oldHasValue := lvalues[i].value.hasValue
			err = DoAssignment(s, op, lvalues[i], value)
			if err != nil {
				return err
			}
			// Old constant values are no longer constant when assigned to.
			if oldHasValue {
				lvalues[i].value.hasValue = false
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
		if v.typ == nil {
			return NoValue, fmt.Errorf("no type for \"%s\"", id)
		}
		if v.typ.pt == TYP_NONE {
			return NoValue, fmt.Errorf("no type for \"%s\"", id)
		}
		if !v.value.hasValue {
			EmitPush(s, id, v.typ.pt.Name())
		}
		return v.value, err
	} else if s.token == TOK_LBRACK {
		// It is an array
		err = ParseArrayIndexes(s)
		return NoValue, err
	} else if s.found(TOK_LPAR) {
		// It is a function call
		err = ParseFunctionCall(s, id)
		return NoValue, err
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
		if value.typ == nil {
			return NoValue, fmt.Errorf("missing integer type")
		}
		nextToken(s)
	} else if s.token == TOK_FLOAT {
		value.typ = TypeDefs["F64"]
		value.floatValue = s.tokenFloatValue
		value.hasValue = true
		nextToken(s)
	} else if s.token == TOK_STRING {
		EmitPush(s, s.tokenString, "STRING")
		value.typ = TypeDefs["String"]
		value.stringValue = s.tokenString
		value.hasValue = false
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
	var value2 ValueDef
	value, err = ParseUnary(s)
	if err != nil {
		return value, err
	}
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

func ParseSumTerm(s *State) (value ValueDef, err error) {
	var value2 ValueDef
	value, err = ParseProd(s)
	if err != nil {
		return NoValue, err
	}
	for s.token == TOK_PLUS || s.token == TOK_MINUS || s.token == TOK_AND || s.token == TOK_OR {
		op := s.token
		nextToken(s)
		value2, err = ParseProd(s)
		if err != nil {
			return NoValue, err
		}
		ct := CommonType(value.typ.pt, value2.typ.pt)
		if value.typ.pt != ct && !value.hasValue {
			emit(s, "   NOS "+value.typ.pt.Name(), "TO "+ct.Name())
		}
		if value2.typ.pt != ct && !value2.hasValue {
			emit(s, "   TOS "+value2.typ.pt.Name(), "TO "+ct.Name())
		}
		value, err = GenerateOp(s, op, value, value2)
	}
	return value, nil
}

// CompareConsts assumes v1 and v2 both have values.
func CompareConsts(op Token, v1 ValueDef, v2 ValueDef) (bool, error) {
	if v1.typ.pt == TYP_STRING || v2.typ.pt == TYP_STRING {
		if v1.typ.pt != TYP_STRING || v2.typ.pt != TYP_STRING {
			return false, fmt.Errorf("comparing string constant with another type is not allowed")
		}
		x1 := v1.stringValue
		x2 := v2.stringValue
		if op == TOK_EQ {
			return x1 == x2, nil
		} else if op == TOK_NE {
			return x1 == x2, nil
		} else if op == TOK_LT {
			return x1 < x2, nil
		} else if op == TOK_LE {
			return x1 <= x2, nil
		} else if op == TOK_GT {
			return x1 > x2, nil
		} else if op == TOK_GE {
			return x1 >= x2, nil
		}
		return false, fmt.Errorf("unexpected operation on constants, op=%s", TokenNames[op])
	}
	x1 := v1.floatValue
	if v1.typ.pt.IsInteger() {
		x1 = float64(v1.intValue)
	} else if v1.typ.pt == TYP_BOOL && v1.boolValue {
		x1 = 1.0
	}

	x2 := v1.floatValue
	if v1.typ.pt.IsInteger() {
		x2 = float64(v2.intValue)
	} else if v2.typ.pt == TYP_BOOL && v2.boolValue {
		x2 = 1.0
	}
	if op == TOK_EQ {
		return math.Abs(x1-x2)/max(x1, x2, 1e-30) < 1e-7, nil
	} else if op == TOK_NE {
		return math.Abs(x1-x2)/max(x1, x2, 1e-30) >= 1e-7, nil
	} else if op == TOK_LT {
		return x1 < x2, nil
	} else if op == TOK_LE {
		return x1 <= x2, nil
	} else if op == TOK_GT {
		return x1 > x2, nil
	} else if op == TOK_GE {
		return x1 >= x2, nil
	}
	return false, fmt.Errorf("unexpected operation on constants,op %s", TokenNames[op])
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
	default:
		return op
	}
}

func ParseCompareTerm(s *State) (result ValueDef, err error) {
	var value1, value2 ValueDef
	value1, err = ParseSumTerm(s)
	if err != nil {
		return NoValue, err
	}
	result = value1
	if value1.typ == nil {
		slog.Error("ParseCompareTerm: No type")
	}
	if s.token == TOK_LT || s.token == TOK_GT || s.token == TOK_EQ || s.token == TOK_GE || s.token == TOK_LE || s.token == TOK_NE {
		op := s.token
		nextToken(s)
		value2, err = ParseSumTerm(s)
		result.typ = TypeDefs["Bool"]
		if err != nil {
			return NoValue, err
		}
		if value1.hasValue && value2.hasValue {
			result.boolValue, err = CompareConsts(op, value1, value2)
			result.hasValue = true
		} else if value2.hasValue {
			EmitPushConst(s, value2)
			return GenerateOp(s, op, value1, value2)
		} else if value1.hasValue {
			EmitPushConst(s, value1)
			return GenerateOp(s, Inverse(op), value1, value2)
		}
		return result, nil
	}
	if result.typ == nil {
		slog.Error("result.type is nil")
	}
	return result, err
}

func ParseExpression(s *State) (result ValueDef, err error) {
	var value2 ValueDef
	result, err = ParseCompareTerm(s)
	if err != nil {
		return NoValue, err
	}
	if result.typ == nil {
		return NoValue, fmt.Errorf("expression type is nil - internal error")
	}
	for s.token == TOK_LOG_AND || s.token == TOK_LOG_OR {
		if result.typ.pt != TYP_BOOL {
			return NoValue, fmt.Errorf("%s requires boolean operands", s.tokenString)
		}
		op := s.token
		ops := s.tokenString
		nextToken(s)
		value2, err = ParseCompareTerm(s)
		if err != nil {
			return NoValue, err
		}
		if value2.typ == nil {
			return NoValue, fmt.Errorf("value2.typ is nil")
		}
		if value2.typ.pt != TYP_BOOL {
			return NoValue, fmt.Errorf("%s requires boolean operands", ops)
		}
		_, err = GenerateOp(s, op, result, value2)
		if err != nil {
			return NoValue, err
		}
	}
	if result.typ == nil {
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
			err = ParseFunctionCall(s, id)
			if err != nil {
				return nil, err
			}
			for _, t := range f.returnTypes {
				v = ValueDef{typ: t}
				results = append(results, v)
			}
		} else {
			v, err = ParseExpression(s)
			results = append(results, v)
		}
		if !s.found(TOK_COMMA) {
			break
		}
	}
	// if expectRpar && !s.found(TOK_RPAR) {
	//	return nil, fmt.Errorf("expected ) after function, found: %s", s.tokenString)
	// }
	return results, nil
}

func NewLabel(s *State) int {
	s.labelNo++
	return s.labelNo
}

// StartCond will increment noCode if the value is a constant equal to cond
func StartCond(s *State, value *ValueDef, cond bool) {
	if value.hasValue && (value.boolValue != cond) {
		s.noCode++
	}
}

// EndCond will decrement noCode if the value is a constant equal to cond
func EndCond(s *State, value *ValueDef, cond bool) {
	if value.hasValue && (value.boolValue != cond) {
		s.noCode--
	}
}

func ParseIf(s *State) error {
	endLabelUsed := false
	nextToken(s)
	value, err := ParseExpression(s)
	if err != nil {
		return err
	}
	if value.typ.pt != TYP_BOOL {
		return fmt.Errorf("expected boolean but got %s", PrimaryTypeNames[value.typ.pt])
	}
	endLabel := NewLabel(s)
	elseLabel := NewLabel(s)
	if !value.hasValue {
		EmitJumpFalse(s, elseLabel)
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
			EmitJump(s, endLabel)
			endLabelUsed = true
		}
		if s.token == TOK_COLON {
			nextToken(s)
			EmitLabel(s, elseLabel)
			StartCond(s, &value, false)
			err = ParseStatements(s)
			if err != nil {
				return err
			}
			EndCond(s, &value, false)
		}

	} else if s.token == TOK_LBRACE {
		nextToken(s)
		StartCond(s, &value, true)
		err = ParseStatements(s)
		if err != nil {
			return err
		}
		EndCond(s, &value, true)
		if !s.hasReturned && !value.hasValue {
			EmitJump(s, endLabel)
			endLabelUsed = true
		}
		if s.token != TOK_RBRACE {
			return fmt.Errorf("expected } after if clause, but got %s", s.tokenString)
		}
		nextToken(s)
		for s.token == TOK_ELSE {
			EmitLineNo(s)
			EmitLabel(s, elseLabel)
			nextToken(s)
			if s.token == TOK_IF {
				nextToken(s)
				value, err = ParseExpression(s)
				if err != nil {
					return err
				}
				if value.typ.pt != TYP_BOOL {
					return fmt.Errorf("expected boolean but got %s", PrimaryTypeNames[value.typ.pt])
				}
				elseLabel = NewLabel(s)
				EmitJumpFalse(s, elseLabel)
				if s.token != TOK_LBRACE {
					return fmt.Errorf("expected { after if but got %s", s.tokenString)
				}
				nextToken(s)
				// Parsing 'else if' statements
				StartCond(s, &value, true)
				err = ParseStatements(s)
				if err != nil {
					return err
				}
				EndCond(s, &value, true)
				if !s.hasReturned {
					endLabelUsed = true
					EmitJump(s, endLabel)
				}
				if s.token != TOK_RBRACE {
					return fmt.Errorf("expected } after if clause, but got %s", s.tokenString)
				}
				nextToken(s)
			} else if s.token == TOK_LBRACE {
				nextToken(s)
				// Else without if
				StartCond(s, &value, false)
				err = ParseStatements(s)
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

func ParseFunctionDefinition(s *State) error {
	EmitLineNo(s)
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
	argList, err := ParseFormalArgList(s)
	if err != nil {
		return err
	}
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
	f, err = AddFunc(fun, argList, returnList)
	s.currentFunc = f
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
