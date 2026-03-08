package main

import "fmt"

type PrimaryType int

//goland:noinspection ALL,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage,GoSnakeCaseUsage
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
	TYP_PTR
	TYP_ERROR
	TYP_LEN
)

var PrimaryTypeNames = [...]string{
	"None", "Bool", "U8", "I16", "U16", "I32", "U32", "Rune", "I64", "F32",
	"F64", "String", "Struct", "Func", "Map", "Set", "Ptr", "Error"}
var PrimaryTypeSizes = [...]int{
	0, 1, 1, 2, 2, 4, 4, 4, 8, 4,
	4, 4, 4, 4, 4, 4, 4, 4}

type TypeDef struct {
	pt    PrimaryType
	name  string
	basic bool
}

var TypeDefs map[string]*TypeDef

var BoolType = TypeDef{pt: TYP_BOOL, name: "Bool", basic: true}
var NoneType = TypeDef{pt: TYP_NONE, name: "None", basic: true}
var PtrType = TypeDef{pt: TYP_PTR, name: "Ptr", basic: true}
var I64Type = TypeDef{pt: TYP_I64, name: "I64", basic: true}

func InitTypes() {
	TypeDefs = make(map[string]*TypeDef)
	for t := TYP_NONE; t < TYP_LEN; t++ {
		TypeDefs[PrimaryTypeNames[t]] = &TypeDef{pt: t, name: PrimaryTypeNames[t], basic: true}
	}
}

// CommonType is the smallest type that is greater or equal to each of the two types.
// The common operations like add, mult etc. needs identical types on both operands,
// so we promote each to the CommonType.
// F.ex. to add U16 and I16, both must be promoted to I32 to get correct results.
// Overflow is not handled or detected, so adding 32737+32737 will be -2, which is wrong.
func CommonType(t1 PrimaryType, t2 PrimaryType) (*TypeDef, error) {
	if t1 == t2 {
		return &TypeDef{pt: t1, name: PrimaryTypeNames[t1], basic: true}, nil
	}
	if t1 == TYP_F64 || t2 == TYP_F64 {
		// F64 can take all numeric types
		return &TypeDef{pt: TYP_F64, name: PrimaryTypeNames[TYP_F64], basic: true}, nil
	}
	if t1 == TYP_F32 || t2 == TYP_F32 {
		// F32 can take all numeric types (but with loss of precision).
		return &TypeDef{pt: TYP_F32, name: PrimaryTypeNames[TYP_F32], basic: true}, nil
	}
	if t1 == TYP_I64 && t2 < TYP_I64 {
		// I64 can take all integers
		return &TypeDef{pt: TYP_I64, name: PrimaryTypeNames[TYP_I64], basic: true}, nil
	}
	if t2 == TYP_I64 && t1 < TYP_I64 {
		// I64 can take all integers
		return &TypeDef{pt: TYP_I64, name: PrimaryTypeNames[TYP_I64], basic: true}, nil
	}
	if t1 == TYP_U8 {
		// U8 can be included in all other types
		return &TypeDef{pt: t2, name: PrimaryTypeNames[t2], basic: true}, nil
	}
	if t2 == TYP_U8 {
		// U8 can be included in all other types
		return &TypeDef{pt: t1, name: PrimaryTypeNames[t1], basic: true}, nil
	}
	if t1 == TYP_U16 && t2 == TYP_U32 || t2 == TYP_U16 && t1 == TYP_U32 {
		return &TypeDef{pt: TYP_U32, name: PrimaryTypeNames[TYP_U32], basic: true}, nil
	}

	if t1 == TYP_U16 || t2 == TYP_U16 && t1 != TYP_U32 {
		return &TypeDef{pt: TYP_I32, name: PrimaryTypeNames[TYP_I32], basic: true}, nil
	}
	if t1 == TYP_I16 {
		if t2 == TYP_U16 || t2 == TYP_I32 {
			return &TypeDef{pt: TYP_I32, name: PrimaryTypeNames[TYP_I32], basic: true}, nil
		}
	}
	if t2 == TYP_I16 && t1 == TYP_I32 {
		return &TypeDef{pt: TYP_I32, name: PrimaryTypeNames[TYP_I32], basic: true}, nil
	}
	if t1 == TYP_U32 && (t2 <= TYP_U16 || t2 == TYP_I32) {
		return &TypeDef{pt: TYP_I64, name: PrimaryTypeNames[TYP_I64], basic: true}, nil
	}
	if t2 == TYP_U32 && (t1 <= TYP_U16 || t1 == TYP_I32) {
		return &TypeDef{pt: TYP_I64, name: PrimaryTypeNames[TYP_I64], basic: true}, nil
	}
	return nil, fmt.Errorf("Common type not found for %s and %s", t1.Name(), t2.Name())
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

// CanAssign is true if we can assign type "src" to "dst"
func CanAssign(dst PrimaryType, src PrimaryType) bool {
	if src == dst {
		return true
	}
	return dst == TYP_U8 && src == TYP_U8 ||
		dst == TYP_I16 && (src == TYP_I16 || src == TYP_U8) ||
		dst == TYP_I32 && (src == TYP_I32 || src == TYP_I16 || src == TYP_U8) ||
		dst == TYP_I64 && (src == TYP_I32 || src == TYP_U32 || src == TYP_U16 || src == TYP_I16 || src == TYP_U8) ||
		dst == TYP_U16 && (src == TYP_U16 || src == TYP_U8) ||
		dst == TYP_U32 && (src == TYP_U32 || src == TYP_U16 || src == TYP_U8) ||
		dst == TYP_F64 || dst == TYP_F32
}

// CanAssignConst : Given a constant value, can we assign it to the dst variable?
// A F64 can accept anything. An F32 value can accept everything except F64.
// For integers, it depends on the value.
func CanAssignConst(dst PrimaryType, value ValueDef) bool {
	if dst == value.typ.pt {
		return true
	}
	if dst == TYP_F64 {
		return true
	}
	if dst == TYP_F32 && value.typ.pt != TYP_F64 {
		return true
	}
	if dst < TYP_U8 || dst > TYP_I64 {
		return false
	}
	return dst == TYP_I64 ||
		dst == TYP_U8 && value.intValue >= 0 && value.intValue <= 255 ||
		dst == TYP_I16 && value.intValue >= -32768 && value.intValue <= 32767 ||
		dst == TYP_U16 && value.intValue >= -65535 && value.intValue <= 65535 ||
		dst == TYP_I32 && value.intValue >= -2147483648 && value.intValue <= 2147483647 ||
		dst == TYP_U32 && value.intValue >= 0 && value.intValue <= 4294967296
}

func AddType(name string, typ *TypeDef) {
	TypeDefs[name] = typ
}

func (t PrimaryType) IsInteger() bool {
	return t == TYP_I32 || t == TYP_U32 || t == TYP_U16 || t == TYP_I16 || t == TYP_U8 || t == TYP_I64
}

func (t PrimaryType) IsFloat() bool {
	return t == TYP_F32 || t == TYP_F64
}
