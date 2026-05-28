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
	IsInputType bool // The variable is a formal parameter with the "in" specifier, meaning the function takse ownership.
	Kind        VarKind
	FieldOfs    int
	FieldType   *TypeDef
	IsIndirect  bool
}

var VarDefs map[string]*VarDef

func MustFree() bool {
	for _, v := range VarDefs {
		if v.Value.Typ.Pt == TYP_STRING || v.Value.Typ.Pt == TYP_STRUCT {
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

func AddLocalVar(s *State, id string, typ *TypeDef, isConst bool) *VarDef {
	v := VarDefs[id]
	if v == nil {
		// New variable.
		v = &VarDef{Name: id, Typ: typ, Value: ValueDef{Typ: typ, IsConst: isConst}, Kind: LocalVar}
		VarDefs[id] = v
		s.VarCount++
		v.Value.Offset = -s.VarCount * 8 // First local variable is at rbp-16, the next at rpb-24
		// fmt.Printf("AddLocalVar(%s)  offs=%d  s.localSp=%d\n", v.Name, v.Offset, s.localSp)
	}
	return v
}

func ParseType(s *State, id string) (*TypeDef, error) {
	var err error
	if s.token == TOK_LBRACE {
		return nil, nil
	}
	if s.token == TOK_STRUCT {
		return ParseStruct(s, id)
	} else {
		id := s.tokenString
		if id[0] > 'Z' {
			return nil, fmt.Errorf("types must start with a capital letter A..Z: '%s'", id)
		}
		nextToken(s)
		typ, ok := TypeDefs[id]
		if !ok {
			return nil, fmt.Errorf("unknown type: %s", id)
		}
		return typ, err
	}
}
