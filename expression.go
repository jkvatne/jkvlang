package main

import (
	"fmt"
	"log/slog"
)

func ParseType(s *State) (*TypeDef, error) {
	var err error
	if s.token == TOK_LBRACE {
		return nil, nil
	}
	id := s.tokenString
	slog.Info("Parsing type", "id", id)
	nextToken(s)
	typ, ok := TypeDefs[id]
	if !ok {
		return nil, fmt.Errorf("Unknown type: %s", s.tokenString)
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

func ParseFormalArgList(s *State, funcName string) error {
	for {
		if s.token == TOK_RPAR {
			break
		}
		if s.token != TOK_ID {
			return fmt.Errorf("Expected argument name but got %s", s.tokenString)
		}
		id := s.tokenString
		nextToken(s)
		typ, err := ParseType(s)
		if err != nil {
			return err
		}
		if typ == nil {
			return fmt.Errorf("Expected argument type but got nil")
		}
		AddArg(s, funcName, id, typ)
		if s.token == TOK_RPAR {
			break
		}
		if s.token != TOK_COMMA {
			return fmt.Errorf("Expected comma or reight parantesis but got %s", s.tokenString)
		}
		nextToken(s)
	}
	if s.token != TOK_RPAR {
		return fmt.Errorf("Expected ')' but got %s", s.tokenString)
	}
	nextToken(s)
	return nil
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

func ParseArgumentList(s *State) (argCount int, err error) {
	for {
		argCount++
		if s.token == TOK_RPAR {
			break
		}
		var value ValueDef
		value, err = ParseExpression(s)
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
		return argCount, fmt.Errorf("Expected right parantesis but got %s", s.tokenString)
	}
	// Skip the final )
	nextToken(s)
	return argCount, nil
}

func ParseAssignOrCall(s *State) (value ValueDef, err error) {
	// We now have s.token == TOK_ID
	id := s.tokenString
	nextToken(s)
	lvalue, ok := VarDefs[id]
	if !ok {
		if s.token == TOK_ASSIGN {
			// If it is an assign statement we must create the variable if it does not exist
			// We don't yet know the type, so just use nil as type
			lvalue = AddVar(id, nil)
		} else if s.token == TOK_LPAR {
			// Unknown function. Add it
			_ = AddVar(id, &FuncType)
		} else {
			return NoValue, fmt.Errorf("Line %d: Did not find variable \"%s\"", s.lineNum, id)
		}
	}
	if s.token == TOK_LBRACK {
		// TODO: This is an array. For now just skip it
		for s.token != TOK_RBRACK {
			nextToken(s)
		}
		nextToken(s)
	}
	if s.token == TOK_LPAR {
		// This is a function call. Parse the argument list
		nextToken(s)
		var argNo int
		argNo, err = ParseArgumentList(s)
		EmitCall(s, id, argNo)
		// The function call should be alone, so just continue
		nextToken(s)
	} else if s.token == TOK_ASSIGN || s.token == TOK_PLUS_ASGN || s.token == TOK_MINUS_ASGN || s.token == TOK_MULT_ASGN || s.token == TOK_DIV_ASGN {
		// Now parse the expression to find the value
		op := s.token
		nextToken(s)
		value, err = ParseExpression(s)
		if err != nil {
			return value, err
		}
		if value.typ == nil {
			return NoValue, fmt.Errorf("Line %d: No type for \"%s\"", s.lineNum, id)
		}
		if lvalue.typ == nil {
			lvalue.typ = value.typ
		}
		if value.hasValue {
			if CanAssingConst(lvalue.typ.pt, value) {
				lvalue.value = value
				return value, nil
			}
			return NoValue, fmt.Errorf("Line %d: Cannot assign to variable \"%s\"", s.lineNum, id)
		}
		ct := CommonType(lvalue.typ.pt, value.typ.pt)
		if ct != value.typ.pt {
			if ct != lvalue.typ.pt {
				return NoValue, fmt.Errorf("Expected type of left side variable ")
			}
			// Convert expression's type to variable's type
			emit(s, "   TOS "+value.typ.pt.Name()+" TO "+ct.Name(), "")
		}
		// Assign type if not known
		if lvalue.typ.pt == TYP_NONE {
			lvalue.typ.pt = ct
		}
		if !CanAssign(lvalue.typ.pt, value.typ.pt) {
			return NoValue, fmt.Errorf("Expected type %s but got %s for %s", lvalue.typ.pt.Name(), value.typ.Name(), id)
		}
		slog.Info("Store lvalue <op> TOS to", "lvalue", id)
		if op == TOK_ASSIGN {
			EmitStore(s, id, value.typ.pt.Name())
		} else {
			EmitModify(s, id, op, value.typ.pt.Name())
		}
	} else {
		return NoValue, fmt.Errorf(No(s) + " Unrecognized assignment or function call")
	}
	// Statements have no value
	return NoValue, nil
}

// ParseVarOrFunc is called for an unary function or variable.
// Called when en ID is encountered in an expression
func ParseVarOrFunc(s *State) (value ValueDef, err error) {
	// We now have s.token == TOK_ID
	id := s.tokenString
	nextToken(s)
	if s.token != TOK_LBRACK && s.token != TOK_LPAR {
		// It is  a simple variable
		v, ok := VarDefs[id]
		if !ok {
			return NoValue, fmt.Errorf("Line %d: Did not find variable \"%s\"", s.lineNum, id)
		}
		if v.typ == nil {
			return NoValue, fmt.Errorf("No type for \"%s\"", s.lineNum, id)
		}
		if v.typ.pt == TYP_NONE {
			return NoValue, fmt.Errorf("Line %d: No type for \"%s\"", s.lineNum, id)
		}
		if !v.value.hasValue {
			EmitPush(s, id, v.name)
		}
		return v.value, err
	} else if s.token == TOK_LBRACK {
		// It is an array
		err = ParseArrayIndexes(s)
		return NoValue, err
	} else if s.token == TOK_LPAR {
		// It is a function call. Parse argument list
		nextToken(s)
		var argNo int
		argNo, err = ParseArgumentList(s)
		EmitCall(s, id, argNo)
		nextToken(s)
		return NoValue, nil
	}
	return NoValue, fmt.Errorf(No(s) + " Unrecognized variable or function call")
}

// ParseUnary will parse a parantesis term, a number, a string, a function call
func ParseUnary(s *State) (value ValueDef, err error) {
	slog.Info("ParseUnary variable/function/array", "Token", s.tokenString)
	id := s.tokenString
	if s.token == TOK_ID {
		// An id can be either a variable or a function call
		value, err = ParseVarOrFunc(s)
	} else if s.token == TOK_LPAR {
		// Start of parantesis term
		nextToken(s)
		value, err = ParseExpression(s)
		return value, Expect(s, TOK_RPAR)
	} else if s.token == TOK_INT {
		value, err = StringToValue(s.tokenString)
		if value.typ == nil {
			return NoValue, fmt.Errorf(No(s) + " Missing integer type")
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
		EmitPush(s, s.tokenString, "BOOL")
		value = True
		nextToken(s)
	} else if s.token == TOK_FALSE {
		EmitPush(s, s.tokenString, "BOOL")
		value = False
		nextToken(s)
	} else {
		slog.Error("Unexpected token ", s.tokenString)
		return NoValue, fmt.Errorf("Unexpected token %s", s.tokenString)
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
		if value.typ.pt != ct {
			emit(s, "   NOS "+value2.typ.pt.Name(), "TO "+ct.Name())
		}
		if value2.typ.pt != ct {
			emit(s, "   TOS "+value2.typ.pt.Name(), "TO "+ct.Name())
		}
		value, err = GenerateOp(s, op, value, value2)
	}
	return value, nil
}

func ParseCompareTerm(s *State) (result ValueDef, err error) {
	var value1 ValueDef
	var value2 ValueDef
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
			if op == TOK_EQ {
				if value1.stringValue == value2.stringValue {
					return True, nil
				} else {
					return False, nil
				}
			} else if op == TOK_NE {
				if value1.stringValue != value2.stringValue {
					return True, nil
				} else {
					return False, nil
				}
			} else if op == TOK_LE {
				if value1.stringValue <= value2.stringValue {
					return True, nil
				} else {
					return False, nil
				}
			} else if op == TOK_LT {
				if value1.stringValue < value2.stringValue {
					return True, nil
				} else {
					return False, nil
				}
			} else if op == TOK_GE {
				if value1.stringValue >= value2.stringValue {
					return True, nil
				} else {
					return False, nil
				}
			} else if op == TOK_GT {
				if value1.stringValue > value2.stringValue {
					return True, nil
				} else {
					return False, nil
				}
			}
		} else {
			GenerateOp(s, op, value1, value2)
			return result, nil
		}
	}
	if result.typ == nil {
		slog.Error("result.type is nil")
	}
	return result, err
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
		return fmt.Errorf("Line %d: Expected boolean but got %s", s.lineNum, PrimaryTypeNames[typ.typ.pt])
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
			return fmt.Errorf("Expected } after if clause, but got %s", s.tokenString)
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
					return fmt.Errorf("Line %d: Expected boolean but got %s", s.lineNum, PrimaryTypeNames[typ.typ.pt])
				}
				elseLabel = NewLabel(s)
				EmitJumpFalse(s, elseLabel)
				if s.token != TOK_LBRACE {
					return fmt.Errorf("Line %d: Expected { after if but got %s", s.lineNum, s.tokenString)
				}
				nextToken(s)
				slog.Debug("Parsing 'else if' statements")
				err = ParseStatements(s)
				EmitJump(s, endLabel)
				if err != nil {
					return err
				}
				if s.token != TOK_RBRACE {
					return fmt.Errorf("Line %d: Expected } after if clause, but got %s", s.lineNum, s.tokenString)
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
		return fmt.Errorf("Expected { or :  but got %s", s.tokenString)
	}
	slog.Debug("ParseIf end - emitting lagel")
	EmitLabel(s, endLabel)
	return nil
}

func ParseFunctionDefinition(s *State) error {
	EmitLineNo(s)
	nextToken(s)
	if s.token != TOK_ID {
		return fmt.Errorf("Expected function name but got %s", s.tokenString)
	}
	VarInit()
	fun := s.tokenString
	s.currentFunc = fun
	slog.Info("Parsing function definition", "name", fun)
	EmitFunction(s, fun)
	nextToken(s)
	if s.token != TOK_LPAR {
		return fmt.Errorf("Expected left parantesis but got %s", s.tokenString)
	}
	nextToken(s)
	slog.Info("Compiling", "function", fun)
	err := ParseFormalArgList(s, fun)
	if err != nil {
		return err
	}
	// Parse the type of the function (after arguments)
	_, err = ParseType(s)
	if err != nil {
		return err
	}
	// After the type, we must have a {
	if s.token != TOK_LBRACE {
		return fmt.Errorf("Funcion definition expected '{' but got %s", s.tokenString)
	}
	nextToken(s)
	// Now parse all the statements in the function
	err = ParseStatements(s)
	if err != nil {
		return err
	}
	// After all the statements in the function, we must have a right-brace }.
	if s.token != TOK_RBRACE {
		return fmt.Errorf("Function definition expected ending '}' but got %s", s.tokenString)
	}
	if !s.hasReturned {
		EmitReturn(s)
	}
	nextToken(s)
	s.currentFunc = fun
	return nil
}

func ParseTypeDef(s *State) error {
	slog.Info("ParseTypeDef", "id", s.tokenString)
	if s.token != TOK_ID {
		return fmt.Errorf("Expected id but got %s", s.tokenString)
	}
	if s.tokenString[0] > 'Z' {
		return fmt.Errorf("All types must start with uppercase, got %s", s.tokenString)
	}
	id := s.tokenString
	nextToken(s)
	if s.token != TOK_ASSIGN {
		return fmt.Errorf("Expected \"=\" but got %s", s.tokenString)
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

func ParseExpression(s *State) (result ValueDef, err error) {
	var value2 ValueDef
	result, err = ParseCompareTerm(s)
	if result.typ == nil {
		return NoValue, fmt.Errorf("Expression type is nil - internal error")
	}
	if err != nil {
		return NoValue, err
	}
	for s.token == TOK_LOG_AND || s.token == TOK_LOG_OR {
		if result.typ.pt != TYP_BOOL {
			return NoValue, fmt.Errorf("&& requires boolean operands!")
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
			return NoValue, fmt.Errorf("&& requires boolean operands!")
		}
		GenerateOp(s, op, result, value2)
	}
	if result.typ == nil {
		return NoValue, fmt.Errorf("value.type is nil - internal error")
	}
	return result, nil
}
