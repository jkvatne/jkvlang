package main

import "fmt"

type FuncDef struct {
	name        string
	returnTypes []*TypeDef
	parameters  []*VarDef
	stackSize   int
}

var FuncDefs map[string]*FuncDef

func FuncInit() {
	FuncDefs = make(map[string]*FuncDef)
	intPar := VarDef{Typ: &I64Type, Name: "intarg"}
	strPar := VarDef{Typ: &StringType, Name: "strarg"}
	_, _ = AddFunc("print_int", []*VarDef{&intPar}, nil)
	_, _ = AddFunc("println", []*VarDef{&strPar}, nil)
	_, _ = AddFunc("print", []*VarDef{&strPar}, nil)
	_, _ = AddFunc("assert", []*VarDef{&strPar}, nil)
}

func AddFunc(id string, parList []*VarDef, returnList []*TypeDef) (*FuncDef, error) {
	f := FuncDefs[id]
	if f != nil {
		return nil, fmt.Errorf("function %s already defined", id)
	}
	// New function
	f = &FuncDef{name: id, returnTypes: returnList, parameters: parList}
	FuncDefs[id] = f
	// Calculate siz
	f.stackSize = len(parList) + len(returnList)
	return f, nil
}
