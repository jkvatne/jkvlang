package main

import (
	"strconv"

	"github.com/jkvatne/jkv/code"
)

type ConstValue struct {
	Bits        uint64
	Pt          code.PrimaryType
	StringValue string
}

type ValueDef struct {
	Typ         *TypeDef
	IntValue    int64
	UintValue   uint64
	FloatValue  float64
	Unsigned    bool
	BoolValue   bool
	StringValue string
	StringLitNo int
	F64LitNo    int
	F32LitNo    int
	IsReturned  bool
	IsTempObj   bool
	IsConst     bool
	IsIndirect  bool
	Offset      int
	localVar    *VarDef
}

var (
	False          = ValueDef{Typ: &BoolType, IsConst: true, BoolValue: false}
	True           = ValueDef{Typ: &BoolType, IsConst: true, IntValue: 1, BoolValue: true}
	NoValue        = ValueDef{Typ: &NoneType, IsConst: false, BoolValue: false}
	PtrValue       = ValueDef{Typ: &PtrType}
	LiteralDefs    []string
	F64LiteralDefs []float64
	F32LiteralDefs []float32
)

func (v *ValueDef) HasValue() bool {
	return v.IsConst
}

func (v *ValueDef) IsTrue() bool {
	return v.IsConst && v.BoolValue
}

func (v *ValueDef) IsFalse() bool {
	return v.IsConst && !v.BoolValue
}

func LiteralInit() {
	LiteralDefs = make([]string, 0, 20)
	F64LiteralDefs = make([]float64, 0, 20)
}

func AddF64Lit(value float64) int {
	for i, s := range F64LiteralDefs {
		if s == value {
			return i + 1
		}
	}
	F64LiteralDefs = append(F64LiteralDefs, value)
	return len(F64LiteralDefs)
}

func AddF32Lit(value float32) int {
	for i, s := range F32LiteralDefs {
		if s == value {
			return i + 1
		}
	}
	F32LiteralDefs = append(F32LiteralDefs, value)
	return len(F32LiteralDefs)
}

func AddLiteral(value string) int {
	for i, s := range LiteralDefs {
		if s == value {
			return i
		}
	}
	LiteralDefs = append(LiteralDefs, value)
	return len(LiteralDefs) - 1
}

func widest(v1 *ValueDef, v2 *ValueDef) *ValueDef {
	if v1.Typ.Pt > v2.Typ.Pt {
		return v1
	}
	return v2
}

func ValueAsString(v ValueDef) string {
	if v.Typ.Pt == code.TYP_U8 || v.Typ.Pt == code.TYP_U16 || v.Typ.Pt == code.TYP_U32 || v.Typ.Pt == code.TYP_I16 || v.Typ.Pt == code.TYP_I32 || v.Typ.Pt == code.TYP_I64 {
		return strconv.FormatInt(v.IntValue, 10)
	} else if v.Typ.Pt == code.TYP_BOOL {
		if v.BoolValue {
			return "true"
		}
		return "false"
	} else if v.Typ.Pt == code.TYP_F64 {
		return strconv.FormatFloat(v.FloatValue, 'g', -1, 64)
	} else if v.Typ.Pt == code.TYP_F32 {
		return strconv.FormatFloat(v.FloatValue, 'g', -1, 32)
	}
	return v.StringValue
}
