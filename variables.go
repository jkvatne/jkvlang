package main

import (
	"fmt"
)

type VarKind int

const (
	ParVar VarKind = iota
	LocalVar
	StructField
	RetVar
	TempVar
	ErrorVar
)

type VarDef struct {
	Typ         *TypeDef
	Value       ValueDef
	Name        string
	level       int
	IsInputType bool // The variable is a formal parameter with the "in" specifier, meaning the function takse ownership.
	MustFree    bool
	Kind        VarKind
	FieldOfs    int
	FieldType   *TypeDef
	IsIndirect  bool
}

var VarDefs map[string]*VarDef

func MustFree() bool {
	for _, v := range VarDefs {
		if v.MustFree && (v.Value.Typ.Pt == TYP_STRING || v.Value.Typ.Pt == TYP_STRUCT) {
			return true
		}
	}
	return false
}

func VarInit() {
	VarDefs = make(map[string]*VarDef)
	VarDefs["err"] = &VarDef{Name: "err", Typ: &I64Type, Kind: ErrorVar, Value: ValueDef{Typ: &I64Type}}
}

func (v *VarDef) Offset() int {
	return v.Value.Offset
}

func (v *VarDef) Size() int {
	return PrimaryTypeSizes[v.Typ.Pt]
}

func (v *VarDef) SetType(t *TypeDef) {
	v.Typ = t
	v.Value.Typ = t
}

// AddLocalPar is called from ParseFormalParList
// The name "par" should be used only for formal parameters
func AddLocalPar(s *State, name string, typ *TypeDef) *VarDef {
	v := &VarDef{Name: name, Typ: typ, Kind: ParVar}
	s.ParCount++
	v.Value.Offset = 8 + s.ParCount*8
	v.Value.Typ = typ
	VarDefs[name] = v
	return v
}

func AddLocalVar(s *State, id string, typ *TypeDef, isConst bool) *VarDef {
	v := VarDefs[id]
	if v == nil {
		// New variable.
		v = &VarDef{Name: id, Typ: typ, Value: ValueDef{Typ: typ, HasValue: isConst}, Kind: LocalVar}
		VarDefs[id] = v
		s.VarCount++
		v.Value.Offset = -s.VarCount * 8 // First local variable is at rbp-16, the next at rpb-24
		// fmt.Printf("AddLocalVar(%s)  offs=%d  s.localSp=%d\n", v.Name, v.Offset, s.localSp)
	}
	return v
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
	typ, err := ParseType(s, id)
	if err != nil {
		return err
	}
	v := AddLocalVar(s, id, typ, isConst)
	v.Value.Offset = EmitAllocLocalVar("Allocate local variable " + v.Name)

	if s.token == TOK_ASSIGN {
		nextToken(s)
		val = s.tokenString
		v.Value.StringValue = val
		nextToken(s)
	}
	return err
}
