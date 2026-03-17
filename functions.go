package main

import "fmt"

type FuncDef struct {
	name        string
	returnTypes []*TypeDef
	arguments   []*VarDef
	stackSize   int
}

var FuncDefs map[string]*FuncDef

func FuncInit() {
	FuncDefs = make(map[string]*FuncDef)
	intArg := VarDef{Typ: &I64Type, Name: "intarg"}
	strArg := VarDef{Typ: &StringType, Name: "strarg"}
	_, _ = AddFunc("print_int", []*VarDef{&intArg}, nil)
	_, _ = AddFunc("println", []*VarDef{&strArg}, nil)
	_, _ = AddFunc("print", []*VarDef{&strArg}, nil)
	_, _ = AddFunc("assert", []*VarDef{&strArg}, nil)
}

func AddFunc(id string, argList []*VarDef, returnList []*TypeDef) (*FuncDef, error) {
	f := FuncDefs[id]
	if f != nil {
		return nil, fmt.Errorf("function %s already defined", id)
	}
	// New function
	f = &FuncDef{name: id, returnTypes: returnList, arguments: argList}
	FuncDefs[id] = f
	// Calculate siz
	f.stackSize = len(argList) + len(returnList)
	return f, nil
}
