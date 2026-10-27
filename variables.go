package main

import (
	"fmt"
	"strings"

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
	IsReturned  bool
	IsArg       bool
}

var VarDefs map[string]*VarDef
var LastGlobalConstValue string
var LastGlobalConstType *TypeDef

func InitVardefs() {
	VarDefs = make(map[string]*VarDef)
	VarDefs["err"] = &VarDef{Name: "err", Typ: &I64Type, IsGlobal: true}
	VarDefs["argc"] = &VarDef{Name: "argc", Typ: &I64Type, IsGlobal: true}
	VarDefs["argv"] = &VarDef{Name: "argv", Typ: &I64Type, IsGlobal: true}
	VarDefs["env"] = &VarDef{Name: "env", Typ: &I64Type, IsGlobal: true}
	VarDefs["args"] = &VarDef{Name: "args", Typ: &SliceType, IsGlobal: true}
	VarDefs["envs"] = &VarDef{Name: "envs", Typ: &SliceType, IsGlobal: true}
	VarDefs["envc"] = &VarDef{Name: "envc", Typ: &I64Type, IsGlobal: true}
	VarDefs["start_ticks"] = &VarDef{Name: "start_ticks", Typ: &I64Type, IsGlobal: true}
	VarDefs["tick_count"] = &VarDef{Name: "tick_count", Typ: &I64Type, IsGlobal: true}
	VarDefs["tick_frequency"] = &VarDef{Name: "tick_frequency", Typ: &I64Type, IsGlobal: true}
}

func init() {
	InitVardefs()
}

func VarReset(s *State) {
	for _, v := range VarDefs {
		if v.Typ.Pt != code.TYP_ERROR && !v.IsGlobal {
			delete(VarDefs, v.Name)
		}
	}
}

func (v *VarDef) Size() int {
	return code.PrimaryTypeSizes[v.Typ.Pt]
}

// AddLocalPar is called from ParseFormalArgList
// The name "par" should be used only for formal parameters
func AddLocalPar(s *State, name string, typ *TypeDef) *VarDef {
	v := &VarDef{Name: name, Typ: typ, IsArg: true}
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
		v.Offset = -code.LocalSp*8 - 8
	}
	return v
}

func DeleteLocalVar(s *State, id string) (err error) {
	v, ok := VarDefs[id]
	if !ok {
		return fmt.Errorf("local variable '%s' is not defined", id)
	}
	if v != nil && v.Typ.Pt >= code.TYP_STRING && !v.Destroyed {
		// Free variable
		if v.Typ.Pt == code.TYP_STRING {
			EmitLoad(8, v.Offset, "DeleteLocalVar string '"+v.Name+"'")
			EmitFreeString("")
		} else if v.Typ.Pt == code.TYP_STRUCT {
			// Load local var pointer into rax
			EmitLoad(8, v.Offset, "DeleteLocalVar struct "+v.Name)
			err = FreeStruct(v.Typ)
		} else if v.Typ.Pt == code.TYP_SLICE && !v.IsGlobal {
			// Load local var pointer into rax
			EmitLoad(8, v.Offset, "DeleteLocalVar slice "+v.Name)
			EmitFreeSlice(v.Typ)
		}
	}
	delete(VarDefs, id)
	return err
}

func FreeBlockVars(s *State, keep bool) (err error) {
	for _, v := range VarDefs {
		if v.BlockLevel == s.BlockLevel {
			if !keep {
				err = DeleteLocalVar(s, v.Name)
				if err != nil {
					return err
				}
			}
			EmitPopAx("FreeBlockVars:  " + v.Name)
			if keep {
				// This is a special case, used in break.
				// We pop the stack but continue using the same LocalSp
				code.LocalSp++
			}
		}
	}
	return err
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
			t.Fields = nil
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
	// Sort fields with largest fields first, to avoid mis-alignment when fields are small.
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
		return nil, fmt.Errorf("fixed size arrays not implemented yet")
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
	s.next()
	if s.token == TOK_DOT {
		s.next()
		n := s.tokenString
		if n[0] > 'Z' {
			return nil, fmt.Errorf("imported types must start with a capital letter: '%s'", id)
		}
		id = id + "@" + n
		s.next()
	} else if id[0] > 'Z' && !strings.Contains(id, "@") {
		return nil, fmt.Errorf("types must start with a capital letter A..Z: '%s'", id)
	}
	typ, ok := TypeDefs[id]
	if !ok {
		return nil, fmt.Errorf("unknown type: \"%s\"", id)
	}
	return typ, err
}
