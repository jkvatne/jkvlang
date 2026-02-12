package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func Compile(workdir string, inputPath string, outputName string) error {
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return fmt.Errorf("Fatal error " + err.Error())
	}
	s := new(state)
	s.lineNum = 1
	for _, entry := range entries {
		if !entry.IsDir() {
			slog.Info("Compiling", "filename", entry.Name())
			s.text, err = os.ReadFile(filepath.Join(inputPath, entry.Name()))
			s.unitName = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			if err != nil {
				slog.Error("Could not open file %s : %s", entry.Name(), err.Error())
			}
			// CheckFile(s, workdir)
			err = CompileFile(s, workdir)
		}
	}
	return err
}

func CheckFile(s *state, workdir string) {
	for s.token != TOK_EOF {
		nextToken(s)
		slog.Info("Token", "Lno", s.lineNum, "Value", s.token, "String", s.tokenString)
		usedToken[s.token] = true
	}
	for i, t := range usedToken {
		if t == false && i > 0 {
			slog.Error("Missing", "token", i)
		}
	}
}

/* value on stack */
type StackValue = struct {
	typ    TypeDef /* type */
	reg1   int     /* register + flags */
	reg2   int     /* second register, used for 'I64g' type. If not used, set to VT_CONST */
	symbol string  /* symbol, if (VT_SYM | VT_CONST), or if result of unary() for an identifier. */
}

func ParseType(s *state) (*TypeDef, error) {
	var err error
	if s.token == TOK_LBRACE {
		return nil, nil
	}
	id := s.tokenString
	slog.Info("Parsing type", "id", id)
	nextToken(s)
	v := TypeDefs[id]
	if v == nil {
		v = UserTypes[id]
	}
	if v == nil {
		return nil, fmt.Errorf("Unknown type: %s", s.tokenString)
	}
	if s.token == TOK_LBRACK {
		nextToken(s)
		if s.token == TOK_ID {
			v.size, err = strconv.Atoi(s.tokenString)
			nextToken(s)
		}
		if s.token != TOK_RBRACK {
			return nil, fmt.Errorf("Invalid token %s", s.tokenString)
		}
		nextToken(s)

	}
	return v, err
}

type VarLocation int

const (
	VAR_HEAP VarLocation = iota
	VAR_ARG
	VAR_STACK
)

type VarDef = struct {
	name     string
	typ      *TypeDef
	location VarLocation
}

var Variables map[string]*VarDef

func VarInit() {
	Variables = make(map[string]*VarDef)
}

func AddConst(s *state, id string, typ *TypeDef, value string) {
	EmitConst(s, id, value, TypeName[typ.pt])
}

func AddVar(s *state, id string, typ *TypeDef, value string, arraysize int) {
	EmitVar(s, id, value, TypeName[typ.pt])
	v := Variables[id]
	if v == nil {
		Variables[id] = &VarDef{name: id, typ: typ}
	}
}

func AddArg(s *state, funcName string, argName string, typ *TypeDef) {
	slog.Info("Arg list", "funcName", funcName, "ArgName", argName)
	EmitVar(s, argName, "", TypeName[typ.pt])
	Variables[argName] = &VarDef{name: argName, typ: typ}
}

func ParseFormalArgList(s *state, funcName string) error {
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

// ParseUnary will parse a parantesis term, a number, a string, a function call
func ParseUnary(s *state) (typ *TypeDef, err error) {
	slog.Info("ParseUnary variable/function/array", "Token", s.tokenString)
	id := s.tokenString
	if s.token == TOK_ID {
		nextToken(s)
		if s.token == TOK_LBRACK {
			slog.Info("Parse array indexes for ", "array", id)
			for {
				if s.token != TOK_RBRACK {
					break
				}
				nextToken(s)
			}
		}
		if s.token == TOK_LPAR {
			// Argument list
			nextToken(s)
			for {
				if s.token == TOK_RPAR {
					break
				}
				typ, err = ParseExpression(s)
				if err != nil {
					return typ, err
				}
				if s.token != TOK_COMMA {
					break
				}
				nextToken(s)
			}
			if s.token != TOK_RPAR {
				return typ, fmt.Errorf("Expected right parantesis but got %s", s.tokenString)
			}
			slog.Info("Emit CALL", "function", id)
			EmitCall(s, id)
			nextToken(s)
		} else if s.token == TOK_ASSIGN {
			nextToken(s)
			typ, err = ParseExpression(s)
			if TypeDefs[id] == nil {
				EmitVar(s, id, "", "")
			}
			if err != nil {
				return typ, err
			}
			slog.Info("Store top of stack to", "lvalue", id)
			EmitStore(s, id)
		} else if s.token == TOK_PLUS_ASGN || s.token == TOK_MINUS_ASGN || s.token == TOK_MULT_ASGN || s.token == TOK_DIV_ASGN {
			op := s.token
			nextToken(s)
			typ, err = ParseExpression(s)
			if err != nil {
				return typ, err
			}
			slog.Info("Store lvalue op tos to", "lvalue", id)
			EmitModify(s, id, op)
		} else {
			v := Variables[id]
			if v == nil {
				return nil, fmt.Errorf("Did not find variable \"%s\"", id)
			}
			typ = v.typ
			EmitPush(s, id)
		}

	} else if s.token == TOK_LPAR {
		// Parantesis term
		nextToken(s)
		typ, err = ParseExpression(s)
		if err != nil {
			return typ, err
		}
		if s.token != TOK_RPAR {
			return typ, fmt.Errorf("Expected ')' but got %s", s.tokenString)
		}
		nextToken(s)
	} else if s.token == TOK_INT {
		PushInt(s, s.tokenString)
		nextToken(s)
	} else if s.token == TOK_FLOAT {
		PushFloat(s, s.tokenString)
		nextToken(s)
	} else if s.token == TOK_STRING {
		PushString(s, s.tokenString)
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
	} else {
		slog.Info("Unary: Got a variable", "name", id)
		EmitPush(s, id)
	}
	return typ, nil
}

func ParseProd(s *state) (typ *TypeDef, err error) {
	typ, err = ParseUnary(s)
	if err != nil {
		return typ, err
	}
	for s.token == TOK_MULT || s.token == TOK_DIV || s.token == TOK_MOD {
		op := s.token
		nextToken(s)
		typ, err = ParseUnary(s)
		if err == nil {
			GenerateOp(s, op)
		}
		if err != nil {
			return typ, err
		}
	}
	return typ, nil
}

func ParseSumTerm(s *state) (*TypeDef, error) {
	typ, err := ParseProd(s)
	if err != nil {
		return nil, err
	}
	for s.token == TOK_PLUS || s.token == TOK_MINUS || s.token == TOK_AND || s.token == TOK_OR {
		op := s.token
		nextToken(s)
		typ, err = ParseProd(s)
		if err != nil {
			return typ, err
		}
		GenerateOp(s, op)
	}
	return typ, nil
}

func ParseCompareTerm(s *state) (*TypeDef, error) {
	typ, err := ParseSumTerm(s)
	if err != nil {
		return typ, err
	}
	for s.token == TOK_LT || s.token == TOK_GT || s.token == TOK_EQ || s.token == TOK_GE || s.token == TOK_LE || s.token == TOK_NE {
		op := s.token
		nextToken(s)
		typ, err = ParseSumTerm(s)
		if err != nil {
			return typ, err
		}
		GenerateOp(s, op)
	}
	return TypeDefs["bool"], nil
}

func ParseExpression(s *state) (*TypeDef, error) {
	typ, err := ParseCompareTerm(s)
	if err != nil {
		return typ, err
	}
	for s.token == TOK_LOG_AND || s.token == TOK_LOG_OR {
		op := s.token
		nextToken(s)
		typ, err = ParseCompareTerm(s)
		if err != nil {
			return typ, err
		}
		GenerateOp(s, op)
	}
	return typ, nil
}

func ParseStatements(s *state) error {
	nextToken(s)
	for s.token != TOK_RBRACE {
		err := ParseStatement(s)
		if err != nil {
			return err
		}
		if s.token == TOK_RBRACE || s.token == TOK_COLON {
			break
		}
		nextToken(s)
		if s.token == TOK_SEMICOLON {
			nextToken(s)
		}
	}
	return nil
}

func NewLabel(s *state) int {
	s.labelNo++
	return s.labelNo
}

func ParseIf(s *state) error {
	nextToken(s)
	typ, err := ParseExpression(s)
	if err != nil {
		return err
	}
	if typ.pt != TYP_BOOL {
		return fmt.Errorf("Expected boolean but got %s", TypeName[typ.pt])
	}
	endLabel := NewLabel(s)
	elseLabel := NewLabel(s)
	EmitJumpFalse(s, elseLabel)

	if s.token == TOK_COLON || s.token == TOK_QMARK {
		err = ParseStatements(s)
		EmitJump(s, endLabel)
		if err != nil {
			return err
		}
		if s.token == TOK_COLON {
			EmitLabel(s, elseLabel)
			err = ParseStatements(s)
			if err != nil {
				return err
			}
		}
	} else if s.token == TOK_LBRACE {
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
			EmitLabel(s, elseLabel)
			nextToken(s)
			if s.token == TOK_IF {
				nextToken(s)
				typ, err = ParseExpression(s)
				if err != nil {
					return err
				}
				if typ.pt != TYP_BOOL {
					return fmt.Errorf("Expected boolean but got %s", TypeName[typ.pt])
				}
				elseLabel = NewLabel(s)
				EmitJumpFalse(s, elseLabel)
				if s.token != TOK_LBRACE {
					return fmt.Errorf("Expected { after if but got %s", s.tokenString)
				}
				err = ParseStatements(s)
				EmitJump(s, endLabel)
				if err != nil {
					return err
				}
				if s.token != TOK_RBRACE {
					return fmt.Errorf("Expected } after if clause, but got %s", s.tokenString)
				}
				nextToken(s)
			} else {
				err = ParseStatements(s)
				nextToken(s)
			}
		}
		if s.token != TOK_RBRACE {
			return fmt.Errorf("Expected { after else, but got %s", s.tokenString)
		}
	} else {
		return fmt.Errorf("Expected { or :  but got %s", s.tokenString)
	}
	EmitLabel(s, endLabel)

	return nil
}

func ParseStatement(s *state) error {
	if s.token == TOK_RETURN {
		nextToken(s)
		_, err := ParseExpression(s)
		if err != nil {
			return err
		}
		s.returned = true
		EmitReturn(s)
	} else if s.token == TOK_IF {
		err := ParseIf(s)
		if err != nil {
			return err
		}
	} else if s.token == TOK_FOR {
		nextToken(s)
	} else if s.token == TOK_ID {
		_, err := ParseExpression(s)
		if err != nil {
			return err
		}
	} else if s.token == TOK_SEMICOLON {
		// Ignore
		nextToken(s)
	} else if s.token == TOK_VAR {
		return ParseVars(s)
	} else {
		return fmt.Errorf("Unknown statement starting with %s", s.tokenString)
	}
	return nil
}
func ParseFunctionDefinition(s *state) error {
	nextToken(s)
	if s.token != TOK_ID {
		return fmt.Errorf("Expected function name but got %s", s.tokenString)
	}
	VarInit()
	s.returned = false
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
	_, err = ParseType(s)
	if err != nil {
		return err
	}

	if s.token != TOK_LBRACE {
		return fmt.Errorf("Funcion definition expected '{' but got %s", s.tokenString)
	}
	nextToken(s)
	for {
		err = ParseStatement(s)
		if err != nil {
			return err
		}
		if s.token == TOK_RBRACE {
			break
		}
	}
	if s.token != TOK_RBRACE {
		return fmt.Errorf("Funcion definition expected ending '}' but got %s", s.tokenString)
	}
	if fun == "main" {
		EmitExit(s)
	}
	if !s.returned {
		EmitReturn(s)
	}
	nextToken(s)
	s.currentFunc = fun
	return nil
}

func ParseTypeDef(s *state) error {
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

func ParseTypeDefs(s *state) error {
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

func ParseVar(s *state, isConst bool) error {
	var val string
	var err error
	var arraySize int
	if s.token != TOK_ID {
		return fmt.Errorf("Expected id but got %s", s.tokenString)
	}
	id := s.tokenString
	nextToken(s)
	slog.Info("ParseVar", "id", id)
	if s.token == TOK_LBRACK {
		nextToken(s)
		if s.token == TOK_INT {
			arraySize, err = strconv.Atoi(s.tokenString)
		}
		nextToken(s)
		if s.token != TOK_RBRACK {
			return fmt.Errorf("Expected ], got %s", s.tokenString)
		}
		nextToken(s)
	}
	typ, err := ParseType(s)
	if err != nil {
		return err
	}
	if s.token == TOK_ASSIGN || s.token == TOK_MINUS_ASGN || s.token == TOK_PLUS_ASGN || s.token == TOK_MULT_ASGN || s.token == TOK_DIV_ASGN {
		nextToken(s)
		val = s.tokenString
		nextToken(s)
	}
	AddVar(s, id, typ, val, arraySize)
	return err
}

func ParseVars(s *state) error {
	var err error
	nextToken(s)
	if s.token == TOK_LPAR {
		nextToken(s)
		for s.token != TOK_RPAR {
			err = ParseVar(s, false)
			if err != nil {
				break
			}
		}
		nextToken(s)
	} else {
		err = ParseVar(s, false)
	}
	return err
}

func ParseConsts(s *state) error {
	var err error
	nextToken(s)
	if s.token == TOK_LPAR {
		nextToken(s)
		for s.token != TOK_RPAR {
			err = ParseVar(s, true)
			if err != nil {
				break
			}
		}
		nextToken(s)
	} else {
		err = ParseVar(s, true)
	}
	return err
}

func CompileFile(s *state, workdir string) error {
	err := EmitTo(s, workdir)
	if err != nil {
		return err
	}
	InitTypes()
	nextToken(s)
	for s.token != TOK_EOF {
		if s.token == TOK_FUNC {
			err = ParseFunctionDefinition(s)
		} else if s.token == TOK_CONST {
			ParseConsts(s)
		} else if s.token == TOK_TYPE {
			err = ParseTypeDefs(s)
		} else {
			return fmt.Errorf("Unexpected token \"%s\"", s.tokenString)
		}
		if err != nil {
			return err
		}

	}
	return nil
}
