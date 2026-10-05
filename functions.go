package main

import (
	"fmt"
	"strconv"
)

type FuncDef struct {
	name          string
	label         string
	returnTypes   []*TypeDef
	parameters    []*ParDef
	parName       []*string
	floatParCount int
	stackSize     int
	builtin       bool
	VarArg        bool
}

type ParDef struct {
	name string
	typ  *TypeDef
}

var Externals []string

var funcDefList []*FuncDef

func AddFunc(id string, parList []*ParDef, returnList []*TypeDef, builtin bool, vararg bool) (*FuncDef, error) {
	n := FuncCount(id)
	if n > 0 {
		f := FindFuncDef(id, parList)
		if f != nil {
			return nil, fmt.Errorf("function %s already exists", id)
		}
	}
	name := id
	if !builtin {
		name = id + "_" + strconv.Itoa(n+1)
	}
	f := &FuncDef{name: id, label: name, returnTypes: returnList, parameters: parList, builtin: builtin, VarArg: vararg}
	f.stackSize = len(parList) + len(returnList)
	funcDefList = append(funcDefList, f)
	return f, nil
}

func FuncInit() {
	funcDefList = make([]*FuncDef, 0, 16)
	_, _ = AddFunc("println", []*ParDef{{name: "arg", typ: &StringType}}, nil, true, true)
	_, _ = AddFunc("printf", []*ParDef{{name: "arg", typ: &StringType}}, nil, true, true)
	_, _ = AddFunc("print", []*ParDef{{name: "arg", typ: &StringType}}, nil, true, true)
	_, _ = AddFunc("fflush", []*ParDef{}, nil, true, false)
	_, _ = AddFunc("flush", []*ParDef{}, nil, true, false)
	_, _ = AddFunc("assert", []*ParDef{{name: "arg", typ: &BoolType}}, nil, true, true)
	_, _ = AddFunc("exit", []*ParDef{{name: "arg", typ: &StringType}}, nil, true, false)
	_, _ = AddFunc("invert_err", []*ParDef{}, nil, true, false)
	_, _ = AddFunc("create_file", []*ParDef{{name: "arg", typ: &PtrType}, {name: "arg", typ: &I32Type},
		{name: "arg", typ: &I32Type}, {name: "arg", typ: &I32Type}, {name: "arg", typ: &I32Type},
		{name: "arg", typ: &I32Type},
		{name: "arg", typ: &I32Type}}, []*TypeDef{&PtrType}, true, false)
	_, _ = AddFunc("cptr", []*ParDef{{name: "arg", typ: &StringType}}, []*TypeDef{&PtrType}, true, false)
	_, _ = AddFunc("lptr", []*ParDef{{name: "arg", typ: &StringType}}, []*TypeDef{&PtrType}, true, false)
	_, _ = AddFunc("bitlen", []*ParDef{{name: "arg", typ: &I32Type}}, []*TypeDef{&I32Type}, true, false)
	_, _ = AddFunc("len", []*ParDef{{name: "arg", typ: &StringType}}, []*TypeDef{&I32Type}, true, false)
	_, _ = AddFunc("cstrlen", []*ParDef{{name: "arg", typ: &StringType}}, []*TypeDef{&I32Type}, true, false)
	_, _ = AddFunc("get_ticks", nil, nil, true, false)
}

// &VarDef{Name: "err", Typ: &I64Type

func AddExternal(name string) {
	Externals = append(Externals, name)
}

func FuncCount(name string) int {
	cnt := 0
	if len(funcDefList) == 0 {
		return 0
	}
	for _, f := range funcDefList {
		if f.name == name {
			cnt++
		}
	}
	return cnt
}

func FindFuncDef(id string, parameters []*ParDef) *FuncDef {
	for _, f := range funcDefList {
		if f.name == id {
			if f.VarArg && len(f.parameters) <= len(parameters) || len(f.parameters) == len(parameters) {
				for i, p := range f.parameters {
					if p.typ != f.parameters[i].typ {
						continue
					}
				}
				return f
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

func TypeListVar(valuelist []*VarDef) []*ParDef {
	l := make([]*ParDef, 0, 8)
	for _, v := range valuelist {
		p := &ParDef{name: v.Name, typ: v.Typ}
		l = append(l, p)
	}
	return l
}
