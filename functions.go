package main

import "fmt"

type FuncDef = struct {
	name        string
	returnTypes []*TypeDef
	argTypes    []*TypeDef
}

var FuncDefs map[string]*FuncDef

func FuncInit() {
	FuncDefs = make(map[string]*FuncDef)
}

func AddFunc(id string, argList []*TypeDef, returnList []*TypeDef) error {
	f := FuncDefs[id]
	if f != nil {
		return fmt.Errorf("function %s already defined", id)
	}
	// New function
	FuncDefs[id] = &FuncDef{name: id, returnTypes: returnList, argTypes: argList}
	return nil
}
