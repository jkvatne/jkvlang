package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jkvatne/jkv/code"
)

type pkg struct {
	key  string
	path string
}

type ImportedPackages map[string]*pkg

func ParseImport(s *State) error {
	if s.token != TOK_ID && s.token != TOK_STRING {
		return fmt.Errorf("expected id but got %s", s.tokenString)
	}
	id := s.tokenString
	fmt.Printf("Import: %s\n", id)
	s.next()
	return nil
}

// ParseImport parses import statements
func ParseImports(s *State) error {
	var err error
	if s.token == TOK_LPAR {
		s.next()
		for s.token != TOK_RPAR {
			err = ParseImport(s)
			if err != nil {
				return err
			}
		}
		s.next()
	} else {
		err = ParseImport(s)
	}
	return err
}

func ScanFile(s *State, name string) (err error) {
	s.nextChar()
	s.next()
	if s.token == TOK_PACKAGE {
		s.next()
		if s.token != TOK_ID {
			return fmt.Errorf("%s:%d %v", name, code.LineNum, "package name should be on top line")
		}
		fmt.Printf("Package %s\n", s.tokenString)
		s.next()
	}

	// Imports must be at top of file
	if s.token == TOK_IMPORT {
		s.next()
		err = ParseImports(s)
	}

	for s.token != TOK_EOF {
		if s.token == TOK_FUNC {
			err = ParseFuncDef(s)
		} else if s.token == TOK_CONST {
			s.next()
			err = ParseConsts(s)
		} else if s.token == TOK_TYPE {
			s.next()
			err = ParseTypeDefs(s)
		} else {
			err = fmt.Errorf("unexpected token \"%s\"", s.tokenString)
		}
		if err != nil {
			return fmt.Errorf("%s:%d %v", name, code.LineNum, err)
		}
	}
	return err
}

func CreateBuildDir(buildDir string) {
	// Make sure output directory is empty
	err := os.RemoveAll(buildDir)
	if err != nil {
		fmt.Printf("could not remove old working directory " + err.Error())
		os.Exit(1)
	}
	err = os.Mkdir(buildDir, os.ModePerm)
	if err != nil {
		fmt.Printf("could not create working directory " + err.Error())
		os.Exit(1)
	}
}

func InitCompile(buildDir string, libPath string, AsmFileName string) error {
	CreateBuildDir(buildDir)
	err := code.NewAsmFile(AsmFileName, buildDir)
	if err != nil {
		return err
	}

	InitVardefs()
	InitTypes()
	LiteralInit()
	EmitPrologue(libPath, true)
	InitTypes()
	FuncInit()
	return nil
}

func OutputEpilogue() error {
	EmitSection("rodata")
	for i, l := range StringLiteralDefs {
		// ALl strings must be aligned to qword
		EmitStringLitteral("str"+strconv.Itoa(i), l)
	}
	for i, l := range F64LiteralDefs {
		EmitF64Litteral("f64_"+strconv.Itoa(i+1), l)
	}
	for i, l := range F32LiteralDefs {
		EmitF32Litteral("f32_"+strconv.Itoa(i+1), l)
	}
	for _, l := range SliceLiteralDefs {
		EmitSliceLit(*l)
	}
	return nil
}

// CompileFile will compile and run a single file. It must have a main() function.
func CompileFile(buildDir string, libPath string, name string) error {
	fmt.Printf(">>> Compiling %s\n", name)
	err := InitCompile(buildDir, libPath, name)
	if err != nil {
		return err
	}
	s, err := NewState(name)
	if err != nil {
		return err
	}

	defer func(s *State) {
		_ = code.CloseAsmFile()
	}(s)
	err = ScanFile(s, name)
	if err != nil {
		return err
	}
	err = OutputEpilogue()
	if err != nil {
		return err
	}
	return nil
}

// CompileDir will compile all source files in the given directory
// and put the object files in the outputPath
func CompileDir(buildDir string, libPath string, inputPath string) error {
	err := InitCompile(buildDir, libPath, "main")
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return fmt.Errorf("could not open source directory, %v", err.Error())
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			name := filepath.Join(inputPath, entry.Name())
			s, err := NewState(name)
			if err != nil {
				return err
			}
			err = ScanFile(s, name)
			if err != nil {
				return err
			}
			if err != nil {
				return err
			}
			fmt.Printf("File %s compiled ok\n", name)
		}
	}
	return OutputEpilogue()
}

// CompileTests will compile all files in the test directory
// Files starting with err_ should intentionally fail
// Uses the build directory for outputs
func CompileTests(buildDir string, libPath string, inputPath string) (int, error) {
	n := 0
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return n, fmt.Errorf("could not open source directory '%s'", err.Error())
	}
	for _, entry := range entries {
		// For each jkv file in the test directory
		if !entry.IsDir() {
			n++
			name := filepath.Join(inputPath, entry.Name())
			if strings.HasSuffix(name, ".jkv") {
				outputName := strings.TrimSuffix(filepath.Base(name), ".jkv") + ".exe"
				err = CompileFile(buildDir, libPath, name)
				if strings.Contains(name, "err_") {
					if err == nil {
						return n, fmt.Errorf("expected %s to return error when compiled, but it did not", name)
					}
					fmt.Printf("File %s failed with error %v\n", name, err)
				} else {
					if err == nil {
						err = LinkRun(buildDir, libPath, outputName)
					}
					if err != nil {
						return n, fmt.Errorf("error in  %s : %s", name, err.Error())
					}
				}
			}
		}
	}
	return n, err
}

func CompileImports(buildDir string, libPath string) error {
	entries, err := os.ReadDir("./imports")
	if err != nil {
		return fmt.Errorf("could not open source directory, %v", err.Error())
	}
	for _, entry := range entries {
		if entry.IsDir() {
			name := filepath.Join("./imports", entry.Name())
			err = CompileDir(buildDir, libPath, name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
