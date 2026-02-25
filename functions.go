package main

import "fmt"

type FuncDef struct {
	name        string
	returnTypes []*TypeDef
	argTypes    []*TypeDef
	stackSize   int
}

var FuncDefs map[string]*FuncDef

func FuncInit() {
	FuncDefs = make(map[string]*FuncDef)
	args := []*TypeDef{&I64Type}
	_, _ = AddFunc("PrintInt", args, nil)
}

func AddFunc(id string, argList []*TypeDef, returnList []*TypeDef) (*FuncDef, error) {
	f := FuncDefs[id]
	if f != nil {
		return nil, fmt.Errorf("function %s already defined", id)
	}
	// New function
	f = &FuncDef{name: id, returnTypes: returnList, argTypes: argList}
	FuncDefs[id] = f
	// Calculate siz
	f.stackSize = len(argList) + len(returnList)
	return f, nil
}
