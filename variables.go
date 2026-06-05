package main

import (
	"fmt"

	"github.com/jkvatne/jkv/code"
)

type VarKind int

const (
	ErrorVar VarKind = iota
	GlobalConst
	ParVar
	LocalVar
	StructField
	RetVar
	TempVar
)

type VarDef struct {
	Typ         *TypeDef
	Value       ValueDef
	Name        string
	IsInputType bool // The variable is a formal parameter with the "in" specifier, meaning the function takes ownership.
	Kind        VarKind
	FieldOfs    int
	FieldType   *TypeDef
	IsIndirect  bool
	BlockLevel  int
}

var VarDefs map[string]*VarDef

func init() {
	VarDefs = make(map[string]*VarDef)
	VarDefs["err"] = &VarDef{Name: "err", Typ: &I64Type, Kind: ErrorVar, Value: ValueDef{Typ: &I64Type}}
}

func HasLocalVars() bool {
	for _, v := range VarDefs {
		if v.Kind >= ParVar {
			return true
		}
	}
	return false
}

func MustFree() bool {
	for _, v := range VarDefs {
		if v.Value.Typ.Pt == TYP_STRING || v.Value.Typ.Pt == TYP_STRUCT {
			return true
		}
	}
	return false
}

func VarReset(s *State) {
	for _, v := range VarDefs {
		if v.Kind != ErrorVar && v.Kind != GlobalConst {
			delete(VarDefs, v.Name)
		}
	}
	s.LocalVarCount = 0
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

// AddLocalPar is called from ParseFormalArgList
// The name "par" should be used only for formal parameters
func AddLocalPar(s *State, name string, typ *TypeDef) *VarDef {
	v := &VarDef{Name: name, Typ: typ, Kind: ParVar}
	s.ParCount++
	v.Value.Offset = 8 + s.ParCount*8
	v.Value.Typ = typ
	VarDefs[name] = v
	return v
}

func AddGlobalConst(id string, typ *TypeDef) (*VarDef, error) {
	v := VarDefs[id]
	if v != nil {
		return nil, fmt.Errorf("constant '%s' is re-declared", id)
	}
	v = &VarDef{Name: id, Typ: typ, Value: ValueDef{Typ: typ, IsConst: true}, Kind: GlobalConst, BlockLevel: 0}
	VarDefs[id] = v
	return v, nil
}

func AddLocalVar(s *State, id string, typ *TypeDef) *VarDef {
	v := VarDefs[id]
	if v == nil {
		v = &VarDef{Name: id, Typ: typ, Value: ValueDef{Typ: typ, IsConst: false}, Kind: LocalVar, BlockLevel: s.BlockLevel}
		VarDefs[id] = v
		s.LocalVarCount++
		v.Value.Offset = -s.LocalVarCount * 8 // First local variable is at rbp-16, the next at rpb-24
		v.Kind = LocalVar
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
		return ParseStruct(s, id)
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
