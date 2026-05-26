package main

import "fmt"

type FuncDef struct {
	name          string
	returnTypes   []*TypeDef
	parameters    []*VarDef
	floatParCount int
	stackSize     int
	builtin       bool
	VarArg        bool
}

var FuncDefs map[string]*FuncDef

func FuncInit() {
	FuncDefs = make(map[string]*FuncDef)
	strPar := VarDef{Typ: &StringType, Name: "strarg"}
	intPar := VarDef{Typ: &I64Type, Name: "intarg"}
	_, _ = AddFunc("println", []*VarDef{&strPar}, nil, true, true)
	_, _ = AddFunc("printf", []*VarDef{&strPar}, nil, true, true)
	_, _ = AddFunc("fflush", []*VarDef{}, nil, true, false)
	_, _ = AddFunc("assert", []*VarDef{&strPar}, nil, true, true)
	_, _ = AddFunc("exit", []*VarDef{&strPar}, nil, true, false)
	_, _ = AddFunc("invert_err", []*VarDef{}, nil, true, false)
	_, _ = AddFunc("create_file", []*VarDef{&strPar, &intPar, &intPar, &intPar, &intPar, &intPar, &intPar}, []*TypeDef{&PtrType}, true, false)
	_, _ = AddFunc("cptr", []*VarDef{&strPar}, []*TypeDef{&PtrType}, true, false)
	_, _ = AddFunc("lptr", []*VarDef{&strPar}, []*TypeDef{&PtrType}, true, false)
	_, _ = AddFunc("write_file", []*VarDef{&intPar, &intPar, &intPar, &intPar, &intPar}, []*TypeDef{&I64Type}, true, false)
	_, _ = AddFunc("read_file", []*VarDef{&intPar, &intPar, &intPar, &intPar, &intPar}, []*TypeDef{&I64Type}, true, false)
	_, _ = AddFunc("close_file", []*VarDef{&intPar}, nil, true, false)
}

func AddFunc(id string, parList []*VarDef, returnList []*TypeDef, builtin bool, vararg bool) (*FuncDef, error) {
	f := FuncDefs[id]
	if f != nil {
		return nil, fmt.Errorf("function %s already defined", id)
	}
	// New function
	f = &FuncDef{name: id, returnTypes: returnList, parameters: parList, builtin: builtin, VarArg: vararg}
	FuncDefs[id] = f
	// Calculate size
	f.stackSize = len(parList) + len(returnList)
	return f, nil
}
