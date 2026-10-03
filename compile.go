package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jkvatne/jkv/code"
)

var MissingPackages []string

var ImportedPackages map[string]string

func ImportInit() {
	ImportedPackages = make(map[string]string)
}

func DeMangleFun(mangledName string) (path string, fun string) {
	w := strings.Split(mangledName, "@")
	path = w[0]
	fun = w[1]
	return strings.Replace(path, "$", "/", -1), fun
}

func MangleFun(path string, fun string) string {
	return strings.Replace(path, "$", "/", -1) + "@" + fun
}

func Mangle(path string) string {
	return strings.Replace(path, "/", "$", -1)
}

func DeMangle(path string) string {
	return strings.Replace(path, "$", "/", -1)
}

func GetPkgPath(pkgShortName string) (string, error) {
	longName, ok := ImportedPackages[pkgShortName]
	if ok {
		return longName, nil
	}
	return "", errors.New("package '" + pkgShortName + "' not found")
}

// LookupFun will convert a pkg shortname and a function name
// into a mangled function refrernce. F.ex "iter.next" will convert to "github.com$jkvatne$lib$iter@next"
func LookupFun(pkgShortName string, funcName string) (string, error) {
	longName, ok := ImportedPackages[pkgShortName]
	if !ok {
		return funcName, fmt.Errorf("Package not found")
	}
	return longName + "@" + funcName, nil
}

func ParseImport(s *State) error {
	if s.token != TOK_ID && s.token != TOK_STRING {
		return fmt.Errorf("expected id but got %s", s.tokenString)
	}
	path := s.tokenString
	// shortName defaults to the last part of the path.
	shortName := path
	s.next()
	for s.token == TOK_DIV || s.token == TOK_DOT {
		if s.token == TOK_DIV {
			s.next()
			path = path + "/" + s.tokenString
		} else {
			s.next()
			path = path + "." + s.tokenString
		}
		// shortName defaults to the last part of the path.
		shortName = s.tokenString
		s.next()
	}
	// path is now the imported path, like f.ex. "github.com/jkvatne/lib"
	fmt.Printf("Import: %s\n", path)
	if s.token == TOK_AS {
		// Get alternative shortname, if given afte AS
		s.next()
		if s.token != TOK_ID {
			return fmt.Errorf("expected package short name, but got %s", s.tokenString)
		}
		shortName = s.tokenString
	}
	ImportedPackages[shortName] = path
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
	// Check that we have pkg and obj files for the imported packages.
	// If not, we have to compile the missing packages.
	MissingPackages = []string{}
	for shortName, fullName := range ImportedPackages {
		info, err2 := os.Stat("./imports/" + Mangle(fullName))
		if err2 == nil && info.IsDir() {
			fmt.Printf("Existing %s as %s\n", fullName, shortName)
		} else {
			MissingPackages = append(MissingPackages, fullName)
			fmt.Printf("Missing  %s as %s\n", fullName, shortName)
		}
	}
	if len(MissingPackages) > 0 {
		return fmt.Errorf(">> Missing one or more packages. Clone them in the imports directory")
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
		if err != nil {
			return err
		}
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
	ImportInit()
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
	s, err := ResetState(name)
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

// CompileDir will compile a package in the given directory.
// The package can consst of several source code files.
// The output is a single assembly file in the buildDir
func CompileDir(buildDir string, libPath string, inputPath string) error {
	var s *State
	var f *os.File
	inputFiles, err := os.ReadDir(inputPath)
	if err != nil {
		return fmt.Errorf("CompileDir() could not open source directory, %v", err.Error())
	}
	err = InitCompile(buildDir, libPath, "main")
	if err != nil {
		return err
	}
	for _, inputFile := range inputFiles {
		if !inputFile.IsDir() {
			name := filepath.Join(inputPath, inputFile.Name())
			f, err = os.Open(name)
			if err != nil {
				return fmt.Errorf("could not open directory, %v", err.Error())
			}
			s, err = ResetState(name)
			if err != nil {
				_ = f.Close()
				return err
			}
			err = ScanFile(s, name)
			_ = f.Close()
			if err != nil {
				break
			}
		}
	}
	if err == nil {
		return OutputEpilogue()
	} else if err.Error() == "Missing packages" {
		for _, name := range MissingPackages {
			err = CompileDir(buildDir, libPath, "imports/"+Mangle(name))
		}
	}
	_ = code.CloseAsmFile()
	return err
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
