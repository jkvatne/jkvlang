package main

import (
	"fmt"
	"strconv"
	"strings"
)

type ValueDef struct {
	Typ         *TypeDef
	HasValue    bool
	IntValue    int64
	FloatValue  float64
	BoolValue   bool
	StringValue string
	StringLitNo int
}

var (
	False       = ValueDef{Typ: &BoolType, HasValue: true, BoolValue: false}
	True        = ValueDef{Typ: &BoolType, HasValue: true, IntValue: 1, BoolValue: true}
	NoValue     = ValueDef{Typ: &NoneType, HasValue: false, BoolValue: false}
	LiteralDefs []string
)

func LiteralInit() {
	LiteralDefs = make([]string, 0, 20)
}

func AddLiteral(value string) int {
	if value == "" {
		return -1
	}
	for i, s := range LiteralDefs {
		if s == value {
			return i
		}
	}
	LiteralDefs = append(LiteralDefs, value)
	return len(LiteralDefs) - 1
}

func StringToValue(s string) (value ValueDef, err error) {
	if strings.ContainsRune(s, '.') {
		var num float64
		num, err = strconv.ParseFloat(s, 64)
		if err != nil {
			return NoValue, err
		}
		value.Typ.Pt = TYP_F64
		value.FloatValue = num
	} else {
		var num int64
		num, err = strconv.ParseInt(s, 10, 64)
		if err == nil {
			if num >= 0 && num <= 255 {
				value.Typ = TypeDefs["U8"]
			} else if num >= -32768 && num <= 32767 {
				value.Typ = TypeDefs["I16"]
			} else if num >= 32768 && num <= 65536 {
				value.Typ = TypeDefs["U16"]
			} else if num >= -2147483648 && num <= 2147483647 {
				value.Typ = TypeDefs["I32"]
			} else if num >= 2147483648 && num <= 4294967296 {
				value.Typ = TypeDefs["U32"]
			} else {
				value.Typ = TypeDefs["I64"]
			}
			value.IntValue = num
			value.HasValue = true
			return value, nil
		}
	}
	return NoValue, fmt.Errorf("not a value: %s", s)
}

func widest(v1 ValueDef, v2 ValueDef) ValueDef {
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
