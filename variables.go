package main

import (
	"fmt"
	"log/slog"
	"strconv"
)

type VarLocation int

const (
	VAR_HEAP VarLocation = iota
	VAR_ARG
	VAR_STACK
)

/* value on stack */
type StackValue = struct {
	typ    TypeDef /* type */
	reg1   int     /* register + flags */
	reg2   int     /* second register, used for 'I64g' type. If not used, set to VT_CONST */
	symbol string  /* symbol, if (VT_SYM | VT_CONST), or if result of an identifier. */
}

type VarDef = struct {
	name     string
	typ      *TypeDef
	location VarLocation
	value    ValueDef
}

var VarDefs map[string]*VarDef

func VarInit() {
	VarDefs = make(map[string]*VarDef)
}

func AddVar(id string, typ *TypeDef) *VarDef {
	v := VarDefs[id]
	if v == nil {
		// New variable.
		v = &VarDef{name: id, typ: typ, value: ValueDef{typ: typ, hasValue: false}}
		VarDefs[id] = v
	}
	return v
}

func AddConst(s *State, id string, typ *TypeDef, value string) {
	EmitConst(s, id, value, PrimaryTypeNames[typ.pt])
}

func AddArg(s *State, funcName string, argName string, typ *TypeDef) {
	slog.Info("Arg list", "funcName", funcName, "ArgName", argName)
	AddVar(argName, typ)
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
				break
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

// ParseVar will parse a variable declaration
func ParseVar(s *State, isConst bool) error {
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
	v := AddVar(id, typ)
	v.typ.arraySize = arraySize
	if s.token == TOK_ASSIGN {
		nextToken(s)
		val = s.tokenString
		v.value.stringValue = val
		nextToken(s)
	}
	return err
}
