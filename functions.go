package main

import (
	"fmt"
	"strconv"
	"unicode"
	"unicode/utf8"
)

type FuncDef struct {
	name          string
	label         string
	returnTypes   []*TypeDef
	parameters    []*ParDef
	parName       []*string
	floatParCount int
	stackSize     int
	Builtin       bool
	VarArg        bool
	Extern        bool
}

type ParDef struct {
	name string
	typ  *TypeDef
}

var (
	Externals []string
	FuncDefs  []*FuncDef
)

func StartsWithUppercase(s string) bool {
	if len(s) == 0 {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

func AddFunc(id string, parList []*ParDef, returnList []*TypeDef, builtin bool, vararg bool, extern bool) (*FuncDef, error) {
	if StartsWithUppercase(id) {
		return nil, fmt.Errorf("function '%s' cannot start with uppercase", id)
	}
	n := FuncCount(id)
	if n > 0 {
		f := FindFuncDefFromParList(id, parList, true)
		if f != nil {
			// return nil, fmt.Errorf("function %s already exists", id)
			panic("function " + id + " already exists")
		}
	}
	name := id
	if !builtin {
		name = id + "_" + strconv.Itoa(n+1)
	}
	// if len(FuncDefs) >= 28 {
	// fmt.Printf("Func %s, %s\n", id, name)
	// }
	f := &FuncDef{name: id, label: name, returnTypes: returnList, parameters: parList, Builtin: builtin, VarArg: vararg}
	f.stackSize = len(parList) + len(returnList)
	f.Extern = extern
	FuncDefs = append(FuncDefs, f)
	return f, nil
}

func FuncInit() {
	FuncDefs = make([]*FuncDef, 0, 30)
	_, _ = AddFunc("println", []*ParDef{{name: "arg", typ: &StringType}}, nil, true, true, true)
	_, _ = AddFunc("printf", []*ParDef{{name: "arg", typ: &StringType}}, nil, true, true, true)
	_, _ = AddFunc("print", []*ParDef{{name: "arg", typ: &StringType}}, nil, true, true, true)
	_, _ = AddFunc("fflush", []*ParDef{}, nil, true, false, true)
	_, _ = AddFunc("flush", []*ParDef{}, nil, true, false, true)
	_, _ = AddFunc("assert", []*ParDef{{name: "arg", typ: &BoolType}}, nil, true, true, true)
	_, _ = AddFunc("exit", []*ParDef{{name: "arg", typ: &StringType}}, nil, true, false, true)
	_, _ = AddFunc("invert_err", []*ParDef{}, nil, true, false, true)
	_, _ = AddFunc("create_file", []*ParDef{{name: "arg", typ: &PtrType}, {name: "arg", typ: &I32Type},
		{name: "arg", typ: &I32Type}, {name: "arg", typ: &I32Type}, {name: "arg", typ: &I32Type},
		{name: "arg", typ: &I32Type},
		{name: "arg", typ: &I32Type}}, []*TypeDef{&PtrType}, true, false, true)
	_, _ = AddFunc("cptr", []*ParDef{{name: "arg", typ: &StringType}}, []*TypeDef{&PtrType}, true, false, true)
	_, _ = AddFunc("lptr", []*ParDef{{name: "arg", typ: &StringType}}, []*TypeDef{&PtrType}, true, false, true)
	_, _ = AddFunc("bitlen", []*ParDef{{name: "arg", typ: &I32Type}}, []*TypeDef{&I32Type}, true, false, true)
	_, _ = AddFunc("bitlen", []*ParDef{{name: "arg", typ: &U32Type}}, []*TypeDef{&I32Type}, true, false, true)
	_, _ = AddFunc("bitlen", []*ParDef{{name: "arg", typ: &I16Type}}, []*TypeDef{&I32Type}, true, false, true)
	_, _ = AddFunc("bitlen", []*ParDef{{name: "arg", typ: &U16Type}}, []*TypeDef{&I32Type}, true, false, true)
	_, _ = AddFunc("bitlen", []*ParDef{{name: "arg", typ: &U8Type}}, []*TypeDef{&I32Type}, true, false, true)
	_, _ = AddFunc("len", []*ParDef{{name: "arg", typ: &StringType}}, []*TypeDef{&I32Type}, true, false, true)
	_, _ = AddFunc("cstrlen", []*ParDef{{name: "arg", typ: &StringType}}, []*TypeDef{&I32Type}, true, false, true)
	_, _ = AddFunc("get_ticks", nil, nil, true, false, true)
	_, _ = AddFunc("get_tick_freq", nil, nil, true, false, true)
	_, _ = AddFunc("low_byte", []*ParDef{{name: "arg", typ: &I32Type}}, []*TypeDef{&U8Type}, true, false, true)
	_, _ = AddFunc("low_byte", []*ParDef{{name: "arg", typ: &U16Type}}, []*TypeDef{&U16Type}, true, false, true)
	_, _ = AddFunc("low_byte", []*ParDef{{name: "arg", typ: &I16Type}}, []*TypeDef{&I16Type}, true, false, true)
	_, _ = AddFunc("floor", []*ParDef{{name: "arg", typ: &F64Type}}, []*TypeDef{&F64Type}, true, false, true)
	_, _ = AddFunc("ceil", []*ParDef{{name: "arg", typ: &F64Type}}, []*TypeDef{&F64Type}, true, false, true)
	_, _ = AddFunc("int64", []*ParDef{{name: "arg", typ: &F64Type}}, []*TypeDef{&I64Type}, true, false, true)
	_, _ = AddFunc("uint64", []*ParDef{{name: "arg", typ: &F64Type}}, []*TypeDef{&U64Type}, true, false, true)
	_, _ = AddFunc("int32", []*ParDef{{name: "arg", typ: &F64Type}}, []*TypeDef{&I32Type}, true, false, true)
	_, _ = AddFunc("mul64", []*ParDef{{name: "arg1", typ: &U64Type}, {name: "arg21", typ: &U64Type}},
		[]*TypeDef{&U64Type, &U64Type}, true, false, true)
}

// &VarDef{Name: "err", Typ: &I64Type

func AddExternal(name string) {
	Externals = append(Externals, name)
}

func FuncCount(name string) int {
	cnt := 0
	if len(FuncDefs) == 0 {
		return 0
	}
	for _, f := range FuncDefs {
		if f.name == name {
			cnt++
		}
	}
	return cnt
}

func FindFuncDefFromParList(id string, parameters []*ParDef, exact bool) *FuncDef {
	for _, f := range FuncDefs {
		if f.name == id {
			if f.VarArg && len(f.parameters) <= len(parameters) || len(f.parameters) == len(parameters) {
				ok := true
				for i, p := range f.parameters {
					v := &ValueDef{Typ: parameters[i].typ}
					if exact && p.typ.Pt != v.Typ.Pt || !CanAssign(p.typ.Pt, v) {
						ok = false
						break
					}
				}
				if ok {
					return f
				}
			}
		}
	}
	return nil
}

func TypeListVal(valueList []*ValueDef) []*ParDef {
	l := make([]*ParDef, 0, 8)
	for i, v := range valueList {
		p := &ParDef{name: "arg" + strconv.Itoa(i), typ: v.Typ}
		l = append(l, p)
	}
	return l
}

func TypeListVar(valueList []*VarDef) []*ParDef {
	l := make([]*ParDef, 0, 8)
	for _, v := range valueList {
		p := &ParDef{name: v.Name, typ: v.Typ}
		l = append(l, p)
	}
	return l
}
