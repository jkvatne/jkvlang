package main

import (
	"fmt"
)

type VarDef struct {
	Typ     *TypeDef
	Value   ValueDef
	Name    string
	Offset  int
	IsConst bool
	ArgNo   int
}

var VarDefs map[string]*VarDef

func VarInit() {
	VarDefs = make(map[string]*VarDef)
}

func (v *VarDef) Size() int {
	return PrimaryTypeSizes[v.Typ.Pt]
}
func (v *VarDef) SetType(t *TypeDef) {
	v.Typ = t
	v.Value.Typ = t
}

func AddLocalArg(s *State, name string, typ *TypeDef) *VarDef {
	v := &VarDef{Name: name, Typ: typ, IsConst: false}
	s.ArgCount++
	s.LocalArgSize += 8
	v.Offset = s.LocalArgSize + 8
	v.ArgNo = s.ArgCount
	v.Value.Typ = typ
	VarDefs[name] = v
	return v
}

func AddLocalVar(s *State, id string, typ *TypeDef, isConst bool) *VarDef {
	v := VarDefs[id]
	if v == nil {
		// New variable.
		v = &VarDef{Name: id, Typ: typ, IsConst: isConst, Value: ValueDef{Typ: typ, HasValue: isConst}}
		VarDefs[id] = v
		s.VarCount[s.level]++
	}
	return v
}

func EnterBlock(s *State) {
	s.level++
}

func ExitBlock(s *State) {
	EmitAddSp(s, s.VarCount[s.level], "")
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
		v.Value.StringValue = val
		nextToken(s)
	}
	return err
}
