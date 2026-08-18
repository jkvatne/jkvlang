package main

import "strconv"

type FuncDef struct {
	name          string
	label         string
	returnTypes   []*TypeDef
	parameters    []*TypeDef
	floatParCount int
	stackSize     int
	builtin       bool
	VarArg        bool
}

var funcDefList []*FuncDef

func FuncInit() {
	_, _ = AddFunc("println", []*TypeDef{&StringType}, nil, true, true)
	_, _ = AddFunc("printf", []*TypeDef{&StringType}, nil, true, true)
	_, _ = AddFunc("print", []*TypeDef{&StringType}, nil, true, true)
	_, _ = AddFunc("fflush", []*TypeDef{}, nil, true, false)
	_, _ = AddFunc("flush", []*TypeDef{}, nil, true, false)
	_, _ = AddFunc("assert", []*TypeDef{&BoolType, &AnyType}, nil, true, true)
	_, _ = AddFunc("exit", []*TypeDef{&StringType}, nil, true, false)
	_, _ = AddFunc("invert_err", []*TypeDef{}, nil, true, false)
	_, _ = AddFunc("create_file", []*TypeDef{&PtrType, &I32Type, &I32Type, &I32Type, &I32Type, &I32Type, &I32Type}, []*TypeDef{&PtrType}, true, false)
	_, _ = AddFunc("cptr", []*TypeDef{&StringType}, []*TypeDef{&PtrType}, true, false)
	_, _ = AddFunc("lptr", []*TypeDef{&StringType}, []*TypeDef{&PtrType}, true, false)
	_, _ = AddFunc("write_file", []*TypeDef{&I32Type, &I32Type, &I32Type, &I32Type, &I32Type}, []*TypeDef{&I64Type}, true, false)
	_, _ = AddFunc("read_file", []*TypeDef{&I32Type, &I32Type, &I32Type, &I32Type, &I32Type}, []*TypeDef{&I64Type}, true, false)
	_, _ = AddFunc("close_file", []*TypeDef{&I32Type}, nil, true, false)
	_, _ = AddFunc("bitlen", []*TypeDef{&I32Type}, []*TypeDef{&I32Type}, true, false)
	_, _ = AddFunc("len", []*TypeDef{&StringType}, []*TypeDef{&I32Type}, true, false)
	_, _ = AddFunc("cstrlen", []*TypeDef{&StringType}, []*TypeDef{&I32Type}, true, false)
}

func FuncCount(name string) int {
	cnt := 0
	for _, f := range funcDefList {
		if f.name == name {
			cnt++
		}
	}
	return cnt
}

func FindFuncDef(id string, parameters []*TypeDef) *FuncDef {
	for _, f := range funcDefList {
		if f.name == id {
			if f.VarArg && len(f.parameters) <= len(parameters) || len(f.parameters) == len(parameters) {
				return f
			}
		}
	}
	return nil
}

func AddFunc(id string, parList []*TypeDef, returnList []*TypeDef, builtin bool, vararg bool) (*FuncDef, error) {
	n := FuncCount(id)
	name := id
	if !builtin {
		name = id + strconv.Itoa(n+1)
	}
	f := &FuncDef{name: id, label: name, returnTypes: returnList, parameters: parList, builtin: builtin, VarArg: vararg}
	f.stackSize = len(parList) + len(returnList)
	funcDefList = append(funcDefList, f)
	return f, nil
}

func TypeListVal(valuelist []*ValueDef) []*TypeDef {
	l := make([]*TypeDef, 0, 8)
	for _, v := range valuelist {
		l = append(l, v.Typ)
	}
	return l
}

func TypeListVar(valuelist []*VarDef) []*TypeDef {
	l := make([]*TypeDef, 0, 8)
	for _, v := range valuelist {
		l = append(l, v.Typ)
	}
	return l
}
