package main

import (
	"fmt"
	"log/slog"
	"strconv"
)

type VarLocation int

//goland:noinspection ALL,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage
const (
	VAR_HEAP VarLocation = iota
	VAR_ARG
	VAR_STACK
)

type VarDef = struct {
	name     string
	typ      *TypeDef
	location VarLocation
	value    ValueDef
	isConst  bool
}

var VarDefs map[string]*VarDef

func VarInit() {
	VarDefs = make(map[string]*VarDef)
}

func AddVar(id string, typ *TypeDef, isConst bool) *VarDef {
	v := VarDefs[id]
	if v == nil {
		// New variable.
		v = &VarDef{name: id, typ: typ, isConst: isConst, value: ValueDef{typ: typ, hasValue: isConst}}
		VarDefs[id] = v
	}
	return v
}

func AddArg(s *State, funcName string, argName string, typ *TypeDef) {
	slog.Info("Arg list", "funcName", funcName, "ArgName", argName)
	AddVar(argName, typ, false)
}

// ParseVars parses a parenthesis var declaration
func ParseVars(s *State) error {
	var err error
	nextToken(s)
	if s.token == TOK_LPAR {
		nextToken(s)
		for s.token != TOK_RPAR {
			err = ParseVar(s, false)
			if err != nil {
				return err
			}
		}
		nextToken(s)
	} else {
		err = ParseVar(s, false)
	}
	return err
}

func ParseConsts(s *State) error {
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

// ParseVar will parse a variable or constant declaration
func ParseVar(s *State, isConst bool) error {
	var val string
	var err error
	var arraySize int

	if s.token != TOK_ID {
		return fmt.Errorf("expected id but got %s", s.tokenString)
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
			return fmt.Errorf("expected ], got %s", s.tokenString)
		}
		nextToken(s)
	}
	typ, err := ParseType(s)
	if err != nil {
		return err
	}
	v := AddVar(id, typ, isConst)
	v.typ.arraySize = arraySize
	if s.token == TOK_ASSIGN {
		nextToken(s)
		val = s.tokenString
		v.value.stringValue = val
		nextToken(s)
	}
	return err
}
