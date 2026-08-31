package main

import (
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
	IsReturned  bool
	IsTempObj   bool
	IsConst     bool
	IsIndirect  bool
	Offset      int
	localVar    *VarDef
}

var (
	False             = ValueDef{Typ: &BoolType, IsConst: true, BoolValue: false}
	True              = ValueDef{Typ: &BoolType, IsConst: true, IntValue: 1, BoolValue: true}
	NoValue           = ValueDef{Typ: &NoneType, IsConst: false, BoolValue: false}
	PtrValue          = ValueDef{Typ: &PtrType}
	StringLiteralDefs []string
	F64LiteralDefs    []float64
	F32LiteralDefs    []float32
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
	StringLiteralDefs = make([]string, 0, 20)
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
	for i, s := range StringLiteralDefs {
		if s == value {
			return i
		}
	}
	StringLiteralDefs = append(StringLiteralDefs, value)
	return len(StringLiteralDefs) - 1
}

func widest(v1 *ValueDef, v2 *ValueDef) *ValueDef {
	if v1.Typ.Pt > v2.Typ.Pt {
		return v1
	}
	return v2
}
