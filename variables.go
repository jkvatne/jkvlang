package main

import (
	"fmt"
)

type VarLocation int

type VarDef = struct {
	Typ     *TypeDef
	Size    int
	Value   *ValueDef
	Name    string
	Offset  int
	IsConst bool
	ArgNo   int
}

var VarDefs map[string]*VarDef

func VarInit() {
	VarDefs = make(map[string]*VarDef)
}

func AddLocalArg(s *State, name string, typ *TypeDef) {
	v := &VarDef{Name: name, Typ: typ, IsConst: false, Value: ValueDef{Typ: typ}}
	s.ArgCount++
	s.LocalArgSize += 8
	v.Offset = s.LocalArgSize
	v.ArgNo = s.ArgCount
	VarDefs[name] = v
}

func AddLocalVar(s *State, id string, typ *TypeDef, isConst bool) *VarDef {
	v := VarDefs[id]
	if v == nil {
		// New variable.
		v = &VarDef{Name: id, Typ: typ, IsConst: isConst, Value: ValueDef{Typ: typ, HasValue: isConst}}
		EmitPushConst(s, 0, "New variable "+id)
		s.localSp++
		// Local variables are at negative offset. The first on -8.
		v.Offset = -s.localSp * 8
		VarDefs[id] = v
		s.VarCount[s.level]++
	}
	return v
}

func EnterBlock(s *State) {
	s.level++
}

func ExitBlock(s *State) {
	EmitPop(s, s.VarCount[s.level], "")
	s.level--
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
	if s.token != TOK_ID {
		return fmt.Errorf("expected id but got %s", s.tokenString)
	}
	id := s.tokenString
	nextToken(s)
	if s.token == TOK_LBRACK {
		nextToken(s)
		// TODO: Parse array size
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
	v := AddLocalVar(s, id, typ, isConst)
	if s.token == TOK_ASSIGN {
		nextToken(s)
		val = s.tokenString
		v.Value.stringValue = val
		nextToken(s)
	}
	return err
}
