package main

import (
	"fmt"

	"github.com/jkvatne/jkv/code"
)

type VarDef struct {
	Typ         *TypeDef
	Value       ValueDef
	Name        string
	IsInputType bool // The variable is a formal parameter with the "in" specifier, meaning the function takes ownership.
	BlockLevel  int
	IsGlobal    bool
	Offset      int
}

var VarDefs map[string]*VarDef

func InitVardefs() {
	VarDefs = make(map[string]*VarDef)
	VarDefs["err"] = &VarDef{Name: "err", Typ: &I64Type, IsGlobal: true, Value: ValueDef{Typ: &I64Type}}
}

func init() {
	InitVardefs()
}

func MustFree() bool {
	for _, v := range VarDefs {
		if v.Value.Typ.Pt == code.TYP_STRING || v.Value.Typ.Pt == code.TYP_STRUCT {
			return true
		}
	}
	return false
}

func VarReset(s *State) {
	for _, v := range VarDefs {
		if v.Typ == nil {
			continue // panic("v.Typ is nil")
		}
		if v.Typ.Pt != code.TYP_ERROR && !v.IsGlobal {
			delete(VarDefs, v.Name)
		}
	}
	s.LocalVarCount = 0
}

func (v *VarDef) Size() int {
	return code.PrimaryTypeSizes[v.Typ.Pt]
}

func (v *VarDef) SetType(t *TypeDef) {
	v.Typ = t
	v.Value.Typ = t
}

// AddLocalPar is called from ParseFormalArgList
// The name "par" should be used only for formal parameters
func AddLocalPar(s *State, name string, typ *TypeDef) *VarDef {
	v := &VarDef{Name: name, Typ: typ}
	s.ParCount++
	v.Offset = 8 + s.ParCount*8
	VarDefs[name] = v
	return v
}

func AddGlobalConst(id string, typ *TypeDef) (*VarDef, error) {
	v := VarDefs[id]
	if v != nil {
		return nil, fmt.Errorf("constant '%s' is re-declared", id)
	}
	v = &VarDef{Name: id, Typ: typ, Value: ValueDef{Typ: typ, IsConst: true}, IsGlobal: true, BlockLevel: 0}
	VarDefs[id] = v
	return v, nil
}

func AddLocalVar(s *State, id string, typ *TypeDef) *VarDef {
	v := VarDefs[id]
	if v == nil {
		v = &VarDef{Name: id, Typ: typ, Value: ValueDef{Typ: typ, IsConst: false}, BlockLevel: s.BlockLevel}
		VarDefs[id] = v
		s.LocalVarCount++
		v.Offset = -s.LocalVarCount * 8 // First local variable is at rbp-16, the next at rpb-24
	}
	return v
}

func DeleteLocalVar(s *State, id string) {
	s.LocalVarCount--
	delete(VarDefs, id)
}

func FreeBlockVars(s *State) {
	for _, v := range VarDefs {
		if v.BlockLevel == s.BlockLevel {
			EmitPopAx("FreeBlockVars:  " + v.Name)
			// Hack to avoid double free:
			code.LocalSp++
		}
	}
}

func DeleteBlockVars(s *State) {
	n := 0
	for _, v := range VarDefs {
		if v.BlockLevel == s.BlockLevel {
			DeleteLocalVar(s, v.Name)
			n++
		}
	}
	if n > 0 {
		EmitAddToSp(-n, "Delete block vars")
	}
}

// ParseType expects that the current token is an id for a new or existing type
func ParseType(s *State) (*TypeDef, error) {
	var err error
	id := s.tokenString
	if s.found(TOK_STRUCT) {
		return ParseStructType(s, id)
	}
	nextToken(s)
	if id[0] > 'Z' {
		return nil, fmt.Errorf("types must start with a capital letter A..Z: '%s'", id)
	}
	typ, ok := TypeDefs[id]
	if !ok {
		return nil, fmt.Errorf("unknown type: %s", id)
	}
	return typ, err
}
