package main

import (
	"fmt"
	"strconv"
)

type ConstValue struct {
	Bits        uint64
	Unsigned    bool
	Float       bool
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
	Offset      int
	LocalVar    *VarDef
	IsTempObj   bool
	IsConst     bool
}

var (
	False            = ValueDef{Typ: &BoolType, IsConst: true, BoolValue: false}
	True             = ValueDef{Typ: &BoolType, IsConst: true, IntValue: 1, BoolValue: true}
	NoValue          = ValueDef{Typ: &NoneType, IsConst: false, BoolValue: false}
	PtrValue         = ValueDef{Typ: &PtrType}
	LiteralDefs      []string
	FloatLiteralDefs []float64
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

func StringToValue(s *State) (value *ValueDef, err error) {
	value = &ValueDef{}
	value.IntValue = int64(s.ConstValue.Bits)
	value.UintValue = s.ConstValue.Bits
	if value.IntValue >= 0 && value.IntValue <= 255 {
		value.Typ = TypeDefs["U8"]
	} else if value.IntValue >= -32768 && value.IntValue <= 32767 {
		value.Typ = TypeDefs["I16"]
	} else if value.IntValue >= 32768 && value.IntValue <= 65536 {
		value.Typ = TypeDefs["U16"]
	} else if value.IntValue >= -2147483648 && value.IntValue <= 2147483647 {
		value.Typ = TypeDefs["I32"]
	} else if value.IntValue >= 2147483648 && value.IntValue <= 4294967296 {
		value.Typ = TypeDefs["U32"]
	} else if value.UintValue != 0 {
		value.Typ = TypeDefs["U64"]
	} else {
		value.Typ = TypeDefs["I64"]
	}
	value.IsConst = true
	return value, nil
	return &NoValue, fmt.Errorf("not a value: %s", s)
}

func widest(v1 *ValueDef, v2 *ValueDef) *ValueDef {
	if v1.Typ.Pt > v2.Typ.Pt {
		return v1
	}
	return v2
}

func ValueAsString(v ValueDef) string {
	if v.Typ.Pt == TYP_U8 || v.Typ.Pt == TYP_U16 || v.Typ.Pt == TYP_U32 || v.Typ.Pt == TYP_I16 || v.Typ.Pt == TYP_I32 || v.Typ.Pt == TYP_I64 {
		return strconv.FormatInt(v.IntValue, 10)
	} else if v.Typ.Pt == TYP_BOOL {
		if v.BoolValue {
			return "true"
		}
		return "false"
	} else if v.Typ.Pt == TYP_F64 {
		return strconv.FormatFloat(v.FloatValue, 'g', -1, 64)
	} else if v.Typ.Pt == TYP_F32 {
		return strconv.FormatFloat(v.FloatValue, 'g', -1, 32)
	}
	return v.StringValue
}
