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
	slog.Info("Parsing type", "id", id)
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

func ParseFormalArgList(s *State, funcName string) ([]*TypeDef, error) {
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
		AddArg(s, funcName, id, typ)
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
	slog.Info("DUMMY: Parse array indexes")
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
			lvalue = AddVar(id, nil)
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

func ParseAssignment(s *State, op Token, lvalue *VarDef, value ValueDef) error {
	if lvalue.typ == nil {
		lvalue.typ = value.typ
		lvalue.value.typ = value.typ
	}
	if value.hasValue {
		if CanAssingConst(lvalue.typ.pt, value) {
			lvalue.value = value
		} else {
			return fmt.Errorf("cannot assign to variable \"%s\"", lvalue.name)
		}
	}
	ct := CommonType(lvalue.typ.pt, value.typ.pt)
	if ct != value.typ.pt {
		if ct != lvalue.typ.pt {
			return fmt.Errorf("incompatible types, %s and %s", ct.Name(), lvalue.typ.Name())
		}
		// Convert expression's type to variable's type
		emit(s, "   TOS "+value.typ.pt.Name()+" TO "+ct.Name(), "")
	}
	// Assign type if not known
	if lvalue.typ.pt == TYP_NONE {
		lvalue.typ.pt = ct
	}
	if !CanAssign(lvalue.typ.pt, value.typ.pt) {
		return fmt.Errorf("expected type %s but got %s for %s", lvalue.typ.pt.Name(), value.typ.Name(), lvalue.name)
	}
	slog.Info("Store lvalue <op> TOS to", "lvalue", lvalue.name)
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
			err := ParseAssignment(s, op, lvalues[i], value)
			if err != nil {
				return err
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
	slog.Info("ParseUnary variable/function/array", "Token", s.tokenString)
	id := s.tokenString
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
	} else if s.token == TOK_F32 {
		value.typ = TypeDefs["F32"]
		value.floatValue = s.tokenFloatValue
		value.hasValue = true
		nextToken(s)
	} else if s.token == TOK_F64 {
		value.typ = TypeDefs["F64"]
		value.floatValue = s.tokenFloatValue
		value.hasValue = true
		nextToken(s)
	} else if s.token == TOK_STRING {
		EmitPush(s, s.tokenString, "STRING")
		value.typ = TypeDefs["String"]
		value.stringValue = s.tokenString
		value.hasValue = true
		nextToken(s)
	} else if s.token == TOK_LBRACK {
		slog.Info("Unary: Evaluate array indexes for ", "function", id)
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
func CompareConsts(op Token, v1 ValueDef, v2 ValueDef) bool {
	if v1.typ.pt == TYP_STRING {
		x1 := v1.stringValue
		x2 := v2.stringValue
		if op == TOK_EQ {
			return x1 == x2
		} else if op == TOK_NE {
			return x1 == x2
		} else if op == TOK_LT {
			return x1 < x2
		} else if op == TOK_LE {
			return x1 <= x2
		} else if op == TOK_GT {
			return x1 > x2
		} else if op == TOK_GE {
			return x1 >= x2
		}
		slog.Error("Unexpected operation on constants", "op", op)
		return false
	}
	x1 := v1.floatValue
	if v1.typ.pt != TYP_F32 && v1.typ.pt != TYP_F64 {
		x1 = float64(v1.intValue)
	}
	x2 := v2.floatValue
	if v2.typ.pt != TYP_F32 && v2.typ.pt != TYP_F64 {
		x2 = float64(v1.intValue)
	}
	if op == TOK_EQ {
		return math.Abs(x1-x2)/max(x1, x2, 1e-30) < 1e-7
	} else if op == TOK_NE {
		return math.Abs(x1-x2)/max(x1, x2, 1e-30) >= 1e-7
	} else if op == TOK_LT {
		return x1 < x2
	} else if op == TOK_LE {
		return x1 <= x2
	} else if op == TOK_GT {
		return x1 > x2
	} else if op == TOK_GE {
		return x1 >= x2
	}
	slog.Error("Unexpected operation on constants ", "Op", op)
	return false
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
			result.boolValue = CompareConsts(op, value1, value2)
			result.hasValue = true
		} else if value2.hasValue {
			EmitPushConst(s, value2)
			return GenerateOp(s, op, value1, value2)
		} else if value1.hasValue {
			EmitPushConst(s, value2)
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
			return NoValue, fmt.Errorf("&& requires boolean operands")
		}
		op := s.token
		nextToken(s)
		value2, err = ParseCompareTerm(s)
		if err != nil {
			return NoValue, err
		}
		if value2.typ == nil {
			return NoValue, fmt.Errorf("value2.typ is nil")
		}
		if value2.typ.pt != TYP_BOOL {
			return NoValue, fmt.Errorf("&& requires boolean operands")
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
	expectRpar := s.found(TOK_LPAR)
	for {
		id := s.tokenString
		f := FuncDefs[s.tokenString]
		if s.token == TOK_ID && f != nil && len(f.returnTypes) > 0 {
			nextToken(s)
			if !s.found(TOK_LPAR) {
				return nil, fmt.Errorf("expected ( after function, found: %s", s.tokenString)
			}
			if len(f.returnTypes) == 1 {
				v, err = ParseVarOrFunc(s)
				if err != nil {
					return nil, err
				}
				results = append(results, v)
			} else if len(f.returnTypes) > 1 {
				// The expression is a function call with more than one result
				err = ParseFunctionCall(s, id)
				for _, t := range f.returnTypes {
					v = ValueDef{typ: t}
					results = append(results, v)
				}
			}
		} else {
			v, err = ParseExpression(s)
			results = append(results, v)
		}
		if !s.found(TOK_COMMA) {
			break
		}
	}
	if expectRpar && !s.found(TOK_RPAR) {
		return nil, fmt.Errorf("expected ) after function, found: %s", s.tokenString)
	}
	return results, nil
}

func NewLabel(s *State) int {
	s.labelNo++
	return s.labelNo
}

func ParseIf(s *State) error {
	slog.Debug("ParseIf")
	nextToken(s)
	typ, err := ParseExpression(s)
	if err != nil {
		return err
	}
	if typ.typ.pt != TYP_BOOL {
		return fmt.Errorf("expected boolean but got %s", PrimaryTypeNames[typ.typ.pt])
	}
	endLabel := NewLabel(s)
	elseLabel := NewLabel(s)
	EmitJumpFalse(s, elseLabel)

	if s.token == TOK_COLON || s.token == TOK_QMARK {
		nextToken(s)
		err = ParseStatements(s)
		EmitJump(s, endLabel)
		if err != nil {
			return err
		}
		if s.token == TOK_COLON {
			nextToken(s)
			EmitLabel(s, elseLabel)
			err = ParseStatements(s)
			if err != nil {
				return err
			}
		}

	} else if s.token == TOK_LBRACE {
		slog.Debug("Parse if statements")
		nextToken(s)
		err = ParseStatements(s)
		EmitJump(s, endLabel)
		if err != nil {
			return err
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
				slog.Debug("Parsing else if", "line", s.lineNum)
				nextToken(s)
				typ, err = ParseExpression(s)
				if err != nil {
					return err
				}
				if typ.typ.pt != TYP_BOOL {
					return fmt.Errorf("expected boolean but got %s", PrimaryTypeNames[typ.typ.pt])
				}
				elseLabel = NewLabel(s)
				EmitJumpFalse(s, elseLabel)
				if s.token != TOK_LBRACE {
					return fmt.Errorf("expected { after if but got %s", s.tokenString)
				}
				nextToken(s)
				slog.Debug("Parsing 'else if' statements")
				err = ParseStatements(s)
				EmitJump(s, endLabel)
				if err != nil {
					return err
				}
				if s.token != TOK_RBRACE {
					return fmt.Errorf("expected } after if clause, but got %s", s.tokenString)
				}
				nextToken(s)
			} else if s.token == TOK_LBRACE {
				nextToken(s)
				slog.Debug("Else without if")
				err = ParseStatements(s)
				nextToken(s)
			} else {
				slog.Info("Else without {")
				_, err = ParseStatement(s)
			}
		}
	} else {
		return fmt.Errorf("expected { or :  but got %s", s.tokenString)
	}
	slog.Debug("ParseIf end - emitting label")
	EmitLabel(s, endLabel)
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
	slog.Info("Parsing function definition", "name", fun)
	EmitFunction(s, fun)
	nextToken(s)
	if s.token != TOK_LPAR {
		return fmt.Errorf("expected left parenthesis but got %s", s.tokenString)
	}
	nextToken(s)
	slog.Info("Compiling", "function", fun)
	argList, err := ParseFormalArgList(s, fun)
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
		if !expectRpar || !s.found(TOK_RPAR) {
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
	slog.Info("ParseTypeDef", "id", s.tokenString)
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
	AddType(s, id, typ)
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
