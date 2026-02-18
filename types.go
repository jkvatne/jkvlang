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
	pt        PrimaryType
	name      string
	arraySize int
}

var TypeDefs map[string]*TypeDef
var BoolType = TypeDef{pt: TYP_BOOL, name: "Bool"}
var NoneType = TypeDef{pt: TYP_NONE, name: "None"}
var ErrType = TypeDef{pt: TYP_ERROR, name: "Err"}
var FuncType = TypeDef{pt: TYP_FUNC, name: "func"}

func InitTypes() {
	TypeDefs = make(map[string]*TypeDef)
	for t := TYP_NONE; t < TYP_LEN; t++ {
		TypeDefs[PrimaryTypeNames[t]] = &TypeDef{pt: t, name: PrimaryTypeNames[t]}
	}
	/*
		// Testing the CommonType() function
		for t1 := TYP_U8; t1 <= TYP_F64; t1++ {
			for t2 := TYP_U8; t2 <= TYP_F64; t2++ {
				if t1 != TYP_RUNE && t2 != TYP_RUNE {
					tc := CommonType(t1, t2)
					fmt.Printf("%10s %10s %10s\n", PrimaryTypeNames[t1], PrimaryTypeNames[t2], PrimaryTypeNames[tc])
					if tc.Name() == "None" {
						fmt.Printf("No common type for %s and %s\n", PrimaryTypeNames[t1], PrimaryTypeNames[t2])
					}
				}
			}
		}	*/
}

// CommonType is the smallest type that is greater or equal to each of the two types.
// The common operations like add, mult etc needs identical types on both operands,
// so we promote each to the CommonType.
// F.ex. to add U16 and I16, both must be promoted to I32 to get correct results.
// Overflow is not handled or detected, so adding 32737+32737 will be -2, which is wrong.
func CommonType(t1 PrimaryType, t2 PrimaryType) PrimaryType {
	if t1 == t2 {
		return t1
	}
	if t1 == TYP_F64 || t2 == TYP_F64 {
		// F64 can take all numeric types
		return TYP_F64
	}
	if t1 == TYP_F32 || t2 == TYP_F32 {
		// F32 can take all numeric types (but with loss of precission).
		return TYP_F32
	}
	if t1 == TYP_I64 && t2 < TYP_I64 {
		// I64 can take all integers
		return TYP_I64
	}
	if t2 == TYP_I64 && t1 < TYP_I64 {
		// I64 can take all integers
		return TYP_I64
	}
	if t1 == TYP_U8 {
		// U8 can be included in all other types
		return t2
	}
	if t2 == TYP_U8 {
		// U8 can be included in all other types
		return t1
	}
	if t1 == TYP_U16 && t2 == TYP_U32 || t2 == TYP_U16 && t1 == TYP_U32 {
		return TYP_U32
	}

	if t1 == TYP_U16 && t2 != TYP_U32 || t2 == TYP_U16 && t1 != TYP_U32 {
		return TYP_I32
	}
	if t1 == TYP_I16 {
		if t2 == TYP_U16 || t2 == TYP_I32 {
			return TYP_I32
		}
	}
	if t2 == TYP_I16 {
		if t1 == TYP_U16 || t1 == TYP_I32 {
			return TYP_I32
		}
	}
	if t1 == TYP_U32 && (t2 <= TYP_U16 || t2 == TYP_I32) {
		return TYP_I64
	}
	if t2 == TYP_U32 && (t1 <= TYP_U16 || t1 == TYP_I32) {
		return TYP_I64
	}
	return TYP_NONE
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
	return dst == TYP_U8 && src == TYP_U8 ||
		dst == TYP_I16 && (src == TYP_I16 || src == TYP_U8) ||
		dst == TYP_I32 && (src == TYP_I32 || src == TYP_I16 || src == TYP_U8) ||
		dst == TYP_I64 && (src == TYP_I32 || src == TYP_U32 || src == TYP_U16 || src == TYP_I16 || src == TYP_U8) ||
		dst == TYP_U8 && src == TYP_U8 ||
		dst == TYP_U16 && (src == TYP_U16 || src == TYP_U8) ||
		dst == TYP_U32 && (src == TYP_U32 || src == TYP_U16 || src == TYP_U8) ||
		dst == TYP_F64 || dst == TYP_F32
}

func CanAssingConst(dst PrimaryType, value ValueDef) bool {
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

func AddType(s *State, name string, typ *TypeDef) {
	EmitType(s, name, int(typ.pt))
	TypeDefs[name] = typ
}
