package main

import (
	"fmt"

	"github.com/jkvatne/jkv/code"
)

type VarDef struct {
	Typ         *TypeDef
	Name        string
	IsInputType bool // The variable is a formal parameter with the "in" specifier, meaning the function takes ownership.
	BlockLevel  int
	IsGlobal    bool
	Offset      int
	IsIndirect  bool
	Destroyed   bool
	constValue  string
}

var VarDefs map[string]*VarDef

func InitVardefs() {
	VarDefs = make(map[string]*VarDef)
	VarDefs["err"] = &VarDef{Name: "err", Typ: &I64Type, IsGlobal: true}
}

func init() {
	InitVardefs()
}

func MustFree() bool {
	for _, v := range VarDefs {
		if v.Typ.Pt == code.TYP_STRING || v.Typ.Pt == code.TYP_STRUCT {
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
	v = &VarDef{Name: id, Typ: typ, IsGlobal: true, BlockLevel: 0}
	VarDefs[id] = v
	return v, nil
}

func AddLocalVar(s *State, id string, typ *TypeDef) *VarDef {
	v := VarDefs[id]
	if v == nil {
		v = &VarDef{Name: id, Typ: typ, BlockLevel: s.BlockLevel}
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

func ParseStructType(s *State) (*TypeDef, error) {
	if !s.found(TOK_LBRACE) {
		return nil, fmt.Errorf("expected {, found " + s.tokenString)
	}
	t := &TypeDef{Pt: code.TYP_STRUCT}
	t.Fields = make(map[string]*TypeDef)
	t.Offsets = make(map[string]int)
	count := 0
	for {
		fieldName := s.tokenString
		_, ok := t.Fields[fieldName]
		if ok {
			return nil, fmt.Errorf("field \"%s\" already defined", fieldName)
		}
		s.next()
		fieldTypeName := s.tokenString
		ft, ok := TypeDefs[fieldTypeName]
		if !ok {
			return nil, fmt.Errorf("unknown type \"%s\"", fieldTypeName)
		}
		count++
		t.Fields[fieldName] = ft
		// fmt.Printf("name %s, type %s\n", fieldName, fieldTypeName)
		s.next()
		if s.token == TOK_RBRACE {
			break
		}
	}
	ofs := 0
	for fn, f := range t.Fields {
		if f.Pt.Size() == 8 {
			t.Offsets[fn] = ofs
			ofs += 8
		}
	}
	for fn, f := range t.Fields {
		if f.Pt.Size() == 4 {
			t.Offsets[fn] = ofs
			ofs += 4
		}
	}
	for fn, f := range t.Fields {
		if f.Pt.Size() == 2 {
			t.Offsets[fn] = ofs
			ofs += 2
		}
	}
	for fn, f := range t.Fields {
		if f.Pt.Size() == 1 {
			t.Offsets[fn] = ofs
			ofs += 1
		}
	}
	t.StructSize = (ofs + 7) & 0xFFFFFFF8
	s.next()
	return t, nil
}

func ParseSlice(s *State) (*TypeDef, error) {
	if !s.found(TOK_RBRACK) {
		return nil, fmt.Errorf("Fixed size arrays not implemented yet")
	}
	t := &TypeDef{Pt: code.TYP_SLICE}
	var ok bool
	t.Element, ok = TypeDefs[s.tokenString]
	if !ok {
		return nil, fmt.Errorf("unknown type \"%s\"", s.tokenString)
	}
	s.next()
	return t, nil
}

// ParseType expects that the current token is an id for a new or existing type
func ParseType(s *State) (*TypeDef, error) {
	var err error
	if s.found(TOK_STRUCT) {
		return ParseStructType(s)
	}
	if s.found(TOK_LBRACK) {
		return ParseSlice(s)
	}
	id := s.tokenString
	nextToken(s)
	if id[0] > 'Z' {
		return nil, fmt.Errorf("types must start with a capital letter A..Z: '%s'", id)
	}
	typ, ok := TypeDefs[id]
	if !ok {
		return nil, fmt.Errorf("unknown type: %s", s.tokenString)
	}
	return typ, err
}
