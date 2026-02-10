package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
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

type PrimaryType int

type typeDef struct {
	name string
	size int
	pt   PrimaryType
}
type valueDef struct {
	name string
}

const (
	TYP_NULL PrimaryType = iota
	TYP_I8
	TYP_I16
	TYP_I32
	TYP_I64
	TYP_U8
	TYP_U16
	TYP_U32
	TYP_U64
	TYP_F32
	TYP_F64
	TYP_STRUCT
	TYP_MAP
	TYP_FUNC
	TYP_ARRAY
)

var TypeDefs map[string]*typeDef

func InitTypes() {
	TypeDefs = make(map[string]*typeDef)
	TypeDefs["I32"] = &typeDef{name: "I32", pt: TYP_I32}
	TypeDefs["I64"] = &typeDef{name: "I64", pt: TYP_I64}
	TypeDefs["U32"] = &typeDef{name: "U32", pt: TYP_U32}
	TypeDefs["U64"] = &typeDef{name: "U64", pt: TYP_U64}
	TypeDefs["F32"] = &typeDef{name: "F32", pt: TYP_F32}
	TypeDefs["F64"] = &typeDef{name: "F64", pt: TYP_F64}
	TypeDefs["struct"] = &typeDef{name: "struct", pt: TYP_STRUCT}
	TypeDefs["func"] = &typeDef{name: "func", pt: TYP_FUNC}
	TypeDefs["array"] = &typeDef{name: "array", pt: TYP_ARRAY}
}

/* value on stack */
type StackValue = struct {
	typ    typeDef /* type */
	reg1   int     /* register + flags */
	reg2   int     /* second register, used for 'I64g' type. If not used, set to VT_CONST */
	symbol string  /* symbol, if (VT_SYM | VT_CONST), or if result of unary() for an identifier. */
}

func ParseType(s *state) (*typeDef, error) {
	var err error
	if s.token == TOK_LBRACE {
		return nil, nil
	}
	slog.Info("Parsing type", "id", s.tokenString)
	v := TypeDefs[s.tokenString]
	nextToken(s)
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

func ParseConstValue(s *state) (*valueDef, error) {
	return nil, nil
}

func addDef(id string, typ *typeDef, value string, size string, isConst bool) error {
	return nil
}

func ParseVar(s *state, isConst bool) error {
	var val string
	var err error
	var size string
	if s.token != TOK_ID {
		return fmt.Errorf("Expected id but got %s", s.tokenString)
	}
	id := s.tokenString
	nextToken(s)
	slog.Info("ParseVar", "id", id)
	if s.token == TOK_LBRACK {
		nextToken(s)
		if s.token == TOK_INT {
			size = s.tokenString
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
	if s.token == TOK_ASSIGN {
		nextToken(s)
		val = s.tokenString
		nextToken(s)
	}
	err = addDef(id, typ, val, size, isConst)
	return err
}

func addArg(funcName string, argName string, typ *typeDef) error {
	slog.Info("Arg list", "ArgName", argName, "type", typ.name)
	return nil
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
		addArg(funcName, id, typ)
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
func ParseUnary(s *state) error {
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
				err := ParseExpression(s)
				if err != nil {
					return err
				}
				if s.token != TOK_COMMA {
					break
				}
				nextToken(s)
			}
			if s.token != TOK_RPAR {
				return fmt.Errorf("Expected right parantesis but got %s", s.tokenString)
			}
			slog.Info("Emit CALL", "function", id)
			EmitCall(id)
			nextToken(s)
		} else if s.token == TOK_ASSIGN {
			nextToken(s)
			err := ParseExpression(s)
			if err != nil {
				return err
			}
			slog.Info("Pop", "lvalue", id)
			EmitPop(id)
		} else {
			EmitPush(id)
		}

	} else if s.token == TOK_LPAR {
		// Parantesis term
		nextToken(s)
		err := ParseExpression(s)
		if err != nil {
			return err
		}
		if s.token != TOK_RPAR {
			return fmt.Errorf("Expected ')' but got %s", s.tokenString)
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
		EmitPush(id)
	}
	return nil
}

func ParseProd(s *state) error {
	err := ParseUnary(s)
	if err != nil {
		return err
	}
	for s.token == TOK_MULT || s.token == TOK_DIV || s.token == TOK_MOD {
		op := s.token
		nextToken(s)
		err := ParseUnary(s)
		if err == nil {
			GenerateOp(s, op)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func ParseSumTerm(s *state) error {
	err := ParseProd(s)
	if err != nil {
		return err
	}
	for s.token == TOK_PLUS || s.token == TOK_MINUS {
		op := s.token
		nextToken(s)
		err = ParseProd(s)
		if err != nil {
			return err
		}
		GenerateOp(s, op)
		if err != nil {
			return err
		}
	}
	return nil
}

func ParseCompareTerm(s *state) error {
	err := ParseSumTerm(s)
	if err != nil {
		return err
	}
	for s.token == TOK_LT || s.token == TOK_GT || s.token == TOK_EQ || s.token == TOK_GE || s.token == TOK_LE {
		op := s.token
		nextToken(s)
		err = ParseSumTerm(s)
		if err != nil {
			return err
		}
		GenerateOp(s, op)
		if err != nil {
			return err
		}
	}
	return nil
}

func ParseExpression(s *state) error {
	err := ParseCompareTerm(s)
	if err != nil {
		return err
	}
	for s.token == TOK_LOG_AND || s.token == TOK_LOG_OR {
		op := s.token
		nextToken(s)
		err = ParseCompareTerm(s)
		if err != nil {
			return err
		}
		GenerateOp(s, op)
		if err != nil {
			return err
		}
	}
	return nil
}

func ParseStatements(s *state) error {
	nextToken(s)
	for s.token != TOK_RBRACE {
		err := ParseStatement(s)
		if err != nil {
			return err
		}
		if s.token == TOK_RBRACE {
			break
		}
		nextToken(s)
	}
	return nil
}

func NewLabel(s *state) int {
	s.labelNo++
	return s.labelNo
}

func ParseIf(s *state) error {
	nextToken(s)
	err := ParseExpression(s)
	if err != nil {
		return err
	}
	label := NewLabel(s)
	EmitJumpFalse(s.labelNo)
	if s.token != TOK_LBRACE {
		return fmt.Errorf("Expected { after if but got %s", s.tokenString)
	}
	err = ParseStatements(s)
	if err != nil {
		return err
	}
	if s.token != TOK_RBRACE {
		return fmt.Errorf("Expected } after if clause, but got %s", s.tokenString)
	}
	nextToken(s)
	for s.token == TOK_ELSE {
		EmitLabel(label)
		nextToken(s)
		if s.token == TOK_IF {
			nextToken(s)
			err = ParseExpression(s)
			if err != nil {
				return err
			}
			label = NewLabel(s)
			EmitJump(label)
			if s.token != TOK_LBRACE {
				return fmt.Errorf("Expected { after if but got %s", s.tokenString)
			}
			err = ParseStatements(s)
			if err != nil {
				return err
			}
			if s.token != TOK_RBRACE {
				return fmt.Errorf("Expected } after if clause, but got %s", s.tokenString)
			}
		}
		if s.token != TOK_LBRACE {
			return fmt.Errorf("Expected { after else, but got %s", s.tokenString)
		}
		EmitLabel(label)
		nextToken(s)
	}
	return nil
}

func ParseStatement(s *state) error {
	if s.token == TOK_RETURN {
		nextToken(s)
		err := ParseExpression(s)
		if err != nil {
			return err
		}
		EmitReturn()
	} else if s.token == TOK_IF {
		err := ParseIf(s)
		if err != nil {
			return err
		}
	} else if s.token == TOK_FOR {
		nextToken(s)
	} else if s.token == TOK_ID {
		err := ParseExpression(s)
		if err != nil {
			return err
		}
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
	fun := s.tokenString
	slog.Info("Parsing function definition", "name", fun)
	EmitFunction(fun)
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
	nextToken(s)
	return nil
}

func CompileFile(s *state, workdir string) error {
	InitTypes()
	nextToken(s)
	var err error
	for s.token != TOK_EOF {
		if s.token == TOK_FUNC {
			err = ParseFunctionDefinition(s)
		} else if s.token == TOK_CONST {
			nextToken(s)
			if s.token == TOK_LPAR {
				nextToken(s)
				for s.token != TOK_RPAR {
					err = ParseVar(s, true)
					if err != nil {
						break
					}
					nextToken(s)
				}
				nextToken(s)
			} else {
				err = ParseVar(s, true)
			}
		} else if s.token == TOK_VAR {
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
		} else {
			return fmt.Errorf("Unexpected token %s", s.tokenString)
		}
		if err != nil {
			return err
		}

	}
	return nil
}
