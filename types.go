package main

const (
	TYP_NULL PrimaryType = iota
	TYP_BOOL
	TYP_I8
	TYP_I16
	TYP_I32
	TYP_I64
	TYP_U8
	TYP_U16
	TYP_U32
	TYP_U64
	TYP_F32
	TYP_F64
	TYP_STRUCT
	TYP_MAP
	TYP_FUNC
	TYP_ARRAY
	TYP_RUNE
	TYP_STRING
	TYP_ERROR
)

type PrimaryType int

type TypeDef struct {
	pt   PrimaryType
	size int
}

var TypeName = [...]string{"NULL", "BOOL", "I8", "I16", "I32", "I64", "U8", "U16", "U32", "U64", "F32", "F64", "Rune", "String", "string", "func", "array"}

var TypeDefs map[string]*TypeDef
var UserTypes map[string]*TypeDef

func InitTypes() {
	TypeDefs = make(map[string]*TypeDef)
	UserTypes = make(map[string]*TypeDef)
	TypeDefs["Bool"] = &TypeDef{pt: TYP_BOOL, size: 1}
	TypeDefs["bool"] = &TypeDef{pt: TYP_BOOL, size: 1}
	TypeDefs["I8"] = &TypeDef{pt: TYP_I8, size: 1}
	TypeDefs["I16"] = &TypeDef{pt: TYP_I16, size: 2}
	TypeDefs["I32"] = &TypeDef{pt: TYP_I32, size: 4}
	TypeDefs["I64"] = &TypeDef{pt: TYP_I64, size: 8}
	TypeDefs["U8"] = &TypeDef{pt: TYP_U8, size: 1}
	TypeDefs["U16"] = &TypeDef{pt: TYP_U16, size: 2}
	TypeDefs["U32"] = &TypeDef{pt: TYP_U32, size: 4}
	TypeDefs["U64"] = &TypeDef{pt: TYP_U64, size: 8}
	TypeDefs["F32"] = &TypeDef{pt: TYP_F32, size: 4}
	TypeDefs["F64"] = &TypeDef{pt: TYP_F64, size: 8}
	TypeDefs["Rune"] = &TypeDef{pt: TYP_RUNE, size: 4}
	TypeDefs["String"] = &TypeDef{pt: TYP_RUNE, size: 0}
	TypeDefs["string"] = &TypeDef{pt: TYP_RUNE, size: 0}
	TypeDefs["Struct"] = &TypeDef{pt: TYP_STRUCT, size: 0}
	TypeDefs["func"] = &TypeDef{pt: TYP_FUNC, size: 0}
	TypeDefs["array"] = &TypeDef{pt: TYP_ARRAY, size: 0}
}

func AddType(s *state, name string, typ *TypeDef) {
	EmitType(s, name, int(typ.pt))
	UserTypes[name] = typ
}
