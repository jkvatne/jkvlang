package main

type PrimaryType int

const (
	TYP_NONE PrimaryType = iota
	TYP_BOOL
	TYP_U8
	TYP_I16
	TYP_U16
	TYP_I32
	TYP_U32
	TYP_RUNE
	TYP_I64
	TYP_F32
	TYP_F64
	TYP_STRING
	TYP_STRUCT
	TYP_FUNC
	TYP_MAP
	TYP_SET
	TYP_ERROR
)

type TypeDef struct {
	pt          PrimaryType
	name        string
	elementType PrimaryType
	arraySize   int
}

func (t PrimaryType) Name() string {
	return PrimaryTypeNames[t]
}
func (t PrimaryType) Size() int {
	return PrimaryTypeSizes[t]
}

func (t *TypeDef) Name() string {
	return t.name
}

var PrimaryTypeNames = [...]string{"None", "Bool", "U8", "I16", "U16", "I32", "U32", "I64", "F32", "F64", "Rune", "String", "Struct", "Func", "Error"}
var PrimaryTypeSizes = [...]int{1, 2, 4, 8, 1, 2, 4, 4, 8, 2, 0, 0, 4}

var TypeDefs map[string]*TypeDef

var BoolType = TypeDef{pt: TYP_BOOL, name: "Bool"}
var NoneType = TypeDef{pt: TYP_NONE, name: "None"}

func InitTypes() {
	TypeDefs = make(map[string]*TypeDef)
	TypeDefs["None"] = &NoneType
	TypeDefs["Bool"] = &BoolType
	TypeDefs["I16"] = &TypeDef{pt: TYP_I16, name: "I16"}
	TypeDefs["I32"] = &TypeDef{pt: TYP_I32, name: "I32"}
	TypeDefs["I64"] = &TypeDef{pt: TYP_I64, name: "I64"}
	TypeDefs["U8"] = &TypeDef{pt: TYP_U8, name: "U8"}
	TypeDefs["U16"] = &TypeDef{pt: TYP_U16, name: "U16"}
	TypeDefs["U32"] = &TypeDef{pt: TYP_U32, name: "U32"}
	TypeDefs["F32"] = &TypeDef{pt: TYP_F32, name: "F32"}
	TypeDefs["F64"] = &TypeDef{pt: TYP_F64, name: "F64"}
	TypeDefs["Rune"] = &TypeDef{pt: TYP_RUNE, name: "Rune"}
	TypeDefs["String"] = &TypeDef{pt: TYP_STRING, name: "String"}
	TypeDefs["Struct"] = &TypeDef{pt: TYP_STRUCT, name: "Struct"}
	TypeDefs["Func"] = &TypeDef{pt: TYP_FUNC, name: "Func"}
	TypeDefs["Map"] = &TypeDef{pt: TYP_MAP, name: "Map"}
	TypeDefs["Set"] = &TypeDef{pt: TYP_SET, name: "Set"}
	TypeDefs["Error"] = &TypeDef{pt: TYP_ERROR, name: "Error"}
}

func AddType(s *State, name string, typ *TypeDef) {
	EmitType(s, name, int(typ.pt))
	TypeDefs[name] = typ
}
