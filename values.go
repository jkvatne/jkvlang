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
	FloatLitNo  int
	IsReturned  bool
	LocalVar    *VarDef
	IsTempObj   bool
	IsConst     bool
	IsIndirect  bool
}

var (
	False            = ValueDef{Typ: &BoolType, IsConst: true, BoolValue: false}
	True             = ValueDef{Typ: &BoolType, IsConst: true, IntValue: 1, BoolValue: true}
	NoValue          = ValueDef{Typ: &NoneType, IsConst: false, BoolValue: false}
	PtrValue         = ValueDef{Typ: &PtrType}
	LiteralDefs      []string
	FloatLiteralDefs []float64
)

func (v *ValueDef) Offset() int {
	if v.LocalVar != nil {
		return v.LocalVar.Offset
	}
	return 0
}

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
	FloatLiteralDefs = make([]float64, 0, 20)
}

func AddFloatLiteral(value float64) int {
	for i, s := range FloatLiteralDefs {
		if s == value {
			return i + 1
		}
	}
	FloatLiteralDefs = append(FloatLiteralDefs, value)
	return len(FloatLiteralDefs)
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
