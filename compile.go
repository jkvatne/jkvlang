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

func ParseType(s *state) (*typeDef, error) {
	var err error
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

func addDef(id string, typ *typeDef, value *valueDef, isConst bool) error {
	return nil
}

func ParseVar(s *state, isConst bool) error {
	var val *valueDef
	var err error
	if s.token != TOK_ID {
		return fmt.Errorf("Expected id but got %s", s.tokenString)
	}
	id := s.tokenString
	nextToken(s)
	typ, err := ParseType(s)
	if err != nil {
		return err
	}
	if s.token == TOK_EQ {
		val, err = ParseConstValue(s)
	}
	err = addDef(id, typ, val, isConst)
	return err
}

func addArg(funcName string, argName string, typ *typeDef) error {
	slog.Info("Arg list", "ArgName", argName, "type", typ.name)
	return nil
}

func ParseArgList(s *state, funcName string) error {
	for {
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

func ParseExpression(s *state, funcName string) error {
	slog.Info("Parsing expression", "id", id1, "func", funcName)
	if s.token == TOK_LPAR {
		nextToken(s)
		err := ParseExpression(x, funcName)
		if err != nil {
			return err
		}
		if s.token != TOK_RPAR {
			return fmt.Errorf("Expected ')' but got %s", s.tokenString)
		}
		nextToken(s)
	} else if s.token == TOK_ID {
		id1 := s.tokenString
		nextToken(s)
		if s.token != TOK_LPAR {
			slog.Info("Evaluate", "function", id1)
			for {
				err := ParseExpression(s, funcName)
				if err != nil {
					return err
				}
				if s.token == TOK_RPAR {
					nextToken(s)
					break
				}
				if s.token != TOK_COMMA {
					return fmt.Errorf("Expected comma or right parentesis but got %s", s.tokenString)
				}
				nextToken(s)
			}
		}
	}
	return nil
}

func ParseStatements(s *state, funcName string) error {
	if s.token == TOK_RETURN {
		nextToken(s)
		err := ParseExpression(s, funcName)
		if err != nil {
			return err
		}
	} else if s.token == TOK_FOR {
		nextToken(s)
	}
	return nil
}

func CompileFile(s *state, workdir string) error {
	InitTypes()
	nextToken(s)
	for s.token != TOK_EOF {
		if s.token == TOK_FUNC {
			nextToken(s)
			if s.token != TOK_ID {
				return fmt.Errorf("Expected function name but got %s", s.tokenString)
			}
			fun := s.tokenString
			nextToken(s)
			if s.token != TOK_LPAR {
				return fmt.Errorf("Expected left parantesis but got %s", s.tokenString)
			}
			nextToken(s)
			slog.Info("Compiling", "function", fun)
			err := ParseArgList(s, fun)
			if err != nil {
				return err
			}
			ParseType(s)
			if s.token != TOK_LBRACE {
				return fmt.Errorf("Funcion definition expected '{' but got %s", s.tokenString)
			}
			nextToken(s)
			ParseStatements(s, fun)
			if s.token != TOK_LBRACE {
				return fmt.Errorf("Funcion definition expected ending '}' but got %s", s.tokenString)
			}

		} else if s.token == TOK_CONST {
			nextToken(s)
			if s.token == TOK_LPAR {
				for s.token != TOK_RPAR {
					err := ParseVar(s, true)
					if err != nil {
						return err
					}
				}
			} else {
				err := ParseVar(s, true)
				if err != nil {
					return err
				}
			}
		} else if s.token == TOK_VAR {
			nextToken(s)
			if s.token == TOK_LPAR {
				nextToken(s)
				for s.token != TOK_RPAR {
					err := ParseVar(s, false)
					if err != nil {
						return err
					}
				}
				nextToken(s)
			} else {
				err := ParseVar(s, false)
				if err != nil {
					return err
				}
			}

		} else {
			return fmt.Errorf("Unexpected token %s", s.tokenString)
		}
	}
	return nil
}
