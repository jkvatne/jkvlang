package main

import "fmt"

type FuncDef struct {
	name          string
	returnTypes   []*TypeDef
	parameters    []*VarDef
	floatParCount int
	stackSize     int
	builtin       bool
}

var FuncDefs map[string]*FuncDef

func FuncInit() {
	FuncDefs = make(map[string]*FuncDef)
	strPar := VarDef{Typ: &StringType, Name: "strarg"}
	_, _ = AddFunc("println", []*VarDef{&strPar}, nil, true)
	_, _ = AddFunc("printf", []*VarDef{&strPar}, nil, true)
	_, _ = AddFunc("fflush", []*VarDef{&strPar}, nil, true)
	_, _ = AddFunc("assert", []*VarDef{&strPar}, nil, true)
	_, _ = AddFunc("exit", []*VarDef{&strPar}, nil, true)
	_, _ = AddFunc("invert_err", []*VarDef{}, nil, true)
}

func AddFunc(id string, parList []*VarDef, returnList []*TypeDef, builtin bool) (*FuncDef, error) {
	f := FuncDefs[id]
	if f != nil {
		return nil, fmt.Errorf("function %s already defined", id)
	}
	// New function
	f = &FuncDef{name: id, returnTypes: returnList, parameters: parList, builtin: builtin}
	FuncDefs[id] = f
	// Calculate size
	f.stackSize = len(parList) + len(returnList)
	return f, nil
}
