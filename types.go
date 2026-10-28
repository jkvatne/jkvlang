package main

import (
	"fmt"

	"github.com/jkvatne/jkv/code"
)

type TypeDef struct {
	Pt       code.PrimaryType
	TypeName string
	Basic    bool
	// StructSize is the total size of a struct
	StructSize int
	// Data offset is the offset from the start of a string/slice/buffer to the actual data contents (i.e. first element)
	DataOffset int
	// Element
	Element *TypeDef
	Fields  map[string]*TypeDef
	Offsets map[string]int
}

var TypeDefs map[string]*TypeDef

var BoolType = TypeDef{Pt: code.TYP_BOOL, TypeName: "Bool", Basic: true}
var NoneType = TypeDef{Pt: code.TYP_NONE, TypeName: "None", Basic: true}
var PtrType = TypeDef{Pt: code.TYP_PTR, TypeName: "Ptr", Basic: true}
var I32Type = TypeDef{Pt: code.TYP_I32, TypeName: "I32", Basic: true}
var U32Type = TypeDef{Pt: code.TYP_U32, TypeName: "U32", Basic: true}
var I16Type = TypeDef{Pt: code.TYP_I16, TypeName: "I16", Basic: true}
var U16Type = TypeDef{Pt: code.TYP_U16, TypeName: "U16", Basic: true}
var U8Type = TypeDef{Pt: code.TYP_U8, TypeName: "U8", Basic: true}
var I64Type = TypeDef{Pt: code.TYP_I64, TypeName: "I64", Basic: true}
var U64Type = TypeDef{Pt: code.TYP_U64, TypeName: "U64", Basic: true}
var F64Type = TypeDef{Pt: code.TYP_F64, TypeName: "F64", Basic: true}
var F32Type = TypeDef{Pt: code.TYP_F32, TypeName: "F32", Basic: true}
var AnyType = TypeDef{Pt: code.TYP_NONE, TypeName: "Any", Basic: true}
var StringType = TypeDef{Pt: code.TYP_STRING, TypeName: "String", Basic: true, DataOffset: 8, Element: &U8Type}
var SliceType = TypeDef{Pt: code.TYP_SLICE, TypeName: "Slice", Basic: true, DataOffset: 8, Element: &StringType}
var StructType = TypeDef{Pt: code.TYP_STRUCT, TypeName: "Struct", Basic: true, DataOffset: 0, Element: &AnyType}

func InitTypes() {
	TypeDefs = make(map[string]*TypeDef)
	for t := code.TYP_NONE; t < code.TYP_COUNT; t++ {
		TypeDefs[code.PrimaryTypeNames[t]] = &TypeDef{Pt: t, TypeName: code.PrimaryTypeNames[t], Basic: true}
	}
	TypeDefs["String"].DataOffset = 8
}

func (t *TypeDef) Size() int {
	if t == nil {
		panic("Internal error")
	}
	if t.Pt == code.TYP_STRUCT {
		return t.StructSize
	}
	return t.Pt.Size()
}

// CommonType is the smallest type that is greater or equal to each of the two types.
// The common operations like add, mult etc. needs identical types on both operands,
// so we promote each to the CommonType.
// F.ex. to add U16 and I16, both must be promoted to I32 to get correct results.
// Overflow is not handled or detected, so adding 32737+32737 will be -2, which is wrong.
func CommonType(t1 code.PrimaryType, t2 code.PrimaryType) (*TypeDef, error) {
	if t1 == t2 {
		return &TypeDef{Pt: t1, TypeName: code.PrimaryTypeNames[t1], Basic: true}, nil
	}
	if t1 == code.TYP_F64 || t2 == code.TYP_F64 {
		// F64 can take all numeric types
		return &TypeDef{Pt: code.TYP_F64, TypeName: code.PrimaryTypeNames[code.TYP_F64], Basic: true}, nil
	}
	if t1 == code.TYP_F32 || t2 == code.TYP_F32 {
		// F32 can take all numeric types (but with loss of precision).
		return &TypeDef{Pt: code.TYP_F32, TypeName: code.PrimaryTypeNames[code.TYP_F32], Basic: true}, nil
	}
	if t1 == code.TYP_I64 && t2 < code.TYP_I64 {
		// I64 can take all integers
		return &TypeDef{Pt: code.TYP_I64, TypeName: code.PrimaryTypeNames[code.TYP_I64], Basic: true}, nil
	}
	if t2 == code.TYP_I64 && t1 < code.TYP_I64 {
		// I64 can take all integers
		return &TypeDef{Pt: code.TYP_I64, TypeName: code.PrimaryTypeNames[code.TYP_I64], Basic: true}, nil
	}
	if t1 == code.TYP_U8 {
		// U8 can be included in all other types
		return &TypeDef{Pt: t2, TypeName: code.PrimaryTypeNames[t2], Basic: true}, nil
	}
	if t2 == code.TYP_U8 {
		// U8 can be included in all other types
		return &TypeDef{Pt: t1, TypeName: code.PrimaryTypeNames[t1], Basic: true}, nil
	}
	if t1 == code.TYP_U16 && t2 == code.TYP_U32 || t2 == code.TYP_U16 && t1 == code.TYP_U32 {
		return &TypeDef{Pt: code.TYP_U32, TypeName: code.PrimaryTypeNames[code.TYP_U32], Basic: true}, nil
	}

	if t1 == code.TYP_U16 || t2 == code.TYP_U16 && t1 != code.TYP_U32 {
		return &TypeDef{Pt: code.TYP_I32, TypeName: code.PrimaryTypeNames[code.TYP_I32], Basic: true}, nil
	}
	if t1 == code.TYP_I16 {
		if t2 == code.TYP_U16 || t2 == code.TYP_I32 {
			return &TypeDef{Pt: code.TYP_I32, TypeName: code.PrimaryTypeNames[code.TYP_I32], Basic: true}, nil
		}
	}
	if t2 == code.TYP_I16 && t1 == code.TYP_I32 {
		return &TypeDef{Pt: code.TYP_I32, TypeName: code.PrimaryTypeNames[code.TYP_I32], Basic: true}, nil
	}
	if t1 == code.TYP_U32 && (t2 <= code.TYP_U16 || t2 == code.TYP_I32) {
		return &TypeDef{Pt: code.TYP_I64, TypeName: code.PrimaryTypeNames[code.TYP_I64], Basic: true}, nil
	}
	if t2 == code.TYP_U32 && (t1 <= code.TYP_U16 || t1 == code.TYP_I32) {
		return &TypeDef{Pt: code.TYP_I64, TypeName: code.PrimaryTypeNames[code.TYP_I64], Basic: true}, nil
	}
	return nil, fmt.Errorf("common type not found for %s and %s", t1.Name(), t2.Name())
}

func (t *TypeDef) Name() string {
	return t.TypeName
}

func AddType(name string, typ *TypeDef) {
	TypeDefs[name] = typ
}

// CanAssignConst : Given a constant value, can we assign it to the dst variable?
func CanAssign(dst code.PrimaryType, value *ValueDef) bool {
	src := value.Typ.Pt
	if src == dst {
		return true
	}
	if value.IsConst {
		f, err := StrToConst(value.StringValue)
		if err != nil {
			return false
		}
		return dst == code.TYP_U8 && f >= 0 && f <= 255 ||
			dst == code.TYP_F32 && value.Typ.Pt == code.TYP_F64 ||
			dst == code.TYP_I16 && f >= -32768 && f <= 32767 ||
			dst == code.TYP_U16 && f >= -65535 && f <= 65535 ||
			dst == code.TYP_I32 && f >= -2147483648 && f <= 2147483647 ||
			dst == code.TYP_U32 && f >= 0 && f <= 4294967296 ||
			dst == code.TYP_I64 && f >= -9223372036854775808 && f <= 9223372036854775807 ||
			dst == code.TYP_U64 && f >= 0 && f <= 18446744073709551615 ||
			dst == code.TYP_STRING && f >= 0 && f <= 255
	}

	if value.Typ.Pt.IsInteger() && dst == code.TYP_STRING {
		return true
	}
	if value.Typ.Pt == code.TYP_PTR && dst == code.TYP_I64 {
		return true
	}
	if dst == code.TYP_I16 && (src == code.TYP_I16 || src == code.TYP_U8) {
		return true
	}
	if dst == code.TYP_I32 && (src == code.TYP_I16 || src == code.TYP_U8 || src == code.TYP_U16) {
		return true
	}
	if dst == code.TYP_I64 && (src == code.TYP_I32 || src == code.TYP_U32 || src == code.TYP_U16 || src == code.TYP_I16 || src == code.TYP_U8) {
		return true
	}
	if dst == code.TYP_U16 && src == code.TYP_U8 {
		return true
	}
	if dst == code.TYP_U32 && (src == code.TYP_U16 || src == code.TYP_U8) {
		return true
	}
	if dst == code.TYP_U64 && (src == code.TYP_U32 || src == code.TYP_U16 || src == code.TYP_U8) {
		return true
	}
	if dst == code.TYP_F64 && src == code.TYP_F32 {
		return true
	}

	f, _ := StrToConst(value.StringValue)
	if value.Typ.Pt == code.TYP_U8 && dst == code.TYP_STRING {
		return true
	}
	if dst == value.Typ.Pt {
		return true
	}
	if dst == code.TYP_F64 {
		return true
	}
	// TODO Check range if F64
	if dst == code.TYP_F32 {
		return true
	}
	if dst < code.TYP_U8 || dst > code.TYP_F64 {
		return false
	}
	if src == code.TYP_BOOL {
		return true
	}
	return dst == code.TYP_I64 ||
		dst == code.TYP_U8 && f >= 0 && f <= 255 ||
		dst == code.TYP_I16 && f >= -32768 && f <= 32767 ||
		dst == code.TYP_U16 && f >= -65535 && f <= 65535 ||
		dst == code.TYP_I32 && f >= -2147483648 && f <= 2147483647 ||
		dst == code.TYP_U32 && f >= 0 && f <= 4294967296
}
