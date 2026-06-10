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
var U8Type = TypeDef{Pt: code.TYP_U8, TypeName: "U8", Basic: true}
var I64Type = TypeDef{Pt: code.TYP_I64, TypeName: "I64", Basic: true}
var U64Type = TypeDef{Pt: code.TYP_U64, TypeName: "U64", Basic: true}
var F64Type = TypeDef{Pt: code.TYP_F64, TypeName: "F64", Basic: true}
var StringType = TypeDef{Pt: code.TYP_STRING, TypeName: "String", Basic: true, DataOffset: 8}

func InitTypes() {
	TypeDefs = make(map[string]*TypeDef)
	for t := code.TYP_NONE; t < code.TYP_COUNT; t++ {
		TypeDefs[code.PrimaryTypeNames[t]] = &TypeDef{Pt: t, TypeName: code.PrimaryTypeNames[t], Basic: true}
	}
}

func (t *TypeDef) Size() int {
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

func CanAssignToVar(dstVar *VarDef, src code.PrimaryType) bool {
	dst := dstVar.Typ.Pt
	if dst == src {
		// if dstVar.FieldType != nil {
		//	dst = dstVar.FieldType.Pt
		// }
		return true
	}
	return CanAssign(dst, src)
}

// CanAssign is true if we can assign type "src" to "dst"
func CanAssign(dst code.PrimaryType, src code.PrimaryType) bool {
	if src == dst {
		return true
	}
	if src == code.TYP_PTR && dst == code.TYP_I64 {
		return true
	}
	return dst == code.TYP_U8 && src == code.TYP_U8 ||
		dst == code.TYP_I16 && (src == code.TYP_I16 || src == code.TYP_U8) ||
		dst == code.TYP_I32 && (src == code.TYP_I32 || src == code.TYP_I16 || src == code.TYP_U8) ||
		dst == code.TYP_I64 && (src == code.TYP_I32 || src == code.TYP_U32 || src == code.TYP_U16 || src == code.TYP_I16 || src == code.TYP_U8) ||
		dst == code.TYP_U16 && (src == code.TYP_U16 || src == code.TYP_U8) ||
		dst == code.TYP_U32 && (src == code.TYP_U32 || src == code.TYP_U16 || src == code.TYP_U8) ||
		dst == code.TYP_F64 || dst == code.TYP_F32 ||
		src == code.TYP_I64 || dst == code.TYP_U64 ||
		src == code.TYP_U64 || dst == code.TYP_I64
}

// CanAssignConst : Given a constant value, can we assign it to the dst variable?
// A F64 can accept anything. An F32 value can accept everything except F64.
// For integers, it depends on the value.
func CanAssignConst(dst code.PrimaryType, value *ValueDef) bool {
	if dst == value.Typ.Pt {
		return true
	}
	if dst == code.TYP_F64 {
		return true
	}
	if dst == code.TYP_F32 && value.Typ.Pt != code.TYP_F64 {
		return true
	}
	if dst < code.TYP_U8 || dst > code.TYP_I64 {
		return false
	}
	return dst == code.TYP_I64 ||
		dst == code.TYP_U8 && value.IntValue >= 0 && value.IntValue <= 255 ||
		dst == code.TYP_I16 && value.IntValue >= -32768 && value.IntValue <= 32767 ||
		dst == code.TYP_U16 && value.IntValue >= -65535 && value.IntValue <= 65535 ||
		dst == code.TYP_I32 && value.IntValue >= -2147483648 && value.IntValue <= 2147483647 ||
		dst == code.TYP_U32 && value.IntValue >= 0 && value.IntValue <= 4294967296
}

func AddType(name string, typ *TypeDef) {
	TypeDefs[name] = typ
}
