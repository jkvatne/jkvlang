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
	return DeMangle(path), fun
}

func MangleFun(path string, fun string) string {
	return Mangle(path) + "@" + fun
}

func Mangle(path string) string {
	path = strings.Replace(path, "/", "$", -1)
	path = strings.Replace(path, "\\", "$", -1)
	return strings.Replace(path, ".", "~", -1)
}

func DeMangle(path string) string {
	path = strings.Replace(path, "~", ".", -1)
	return strings.Replace(path, "$", "/", -1)
}

func GetPkgName(name string) string {
	w := strings.Split(name, "^")
	if len(w) == 1 {
		return name
	}
	return w[len(w)-1]
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
	return Mangle(longName) + "@" + funcName, nil
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

	// Check that we have cloned all the needed packages into the imports directory
	MissingPackages = []string{}
	for _, fullName := range ImportedPackages {
		info, err2 := os.Stat("./imports/" + Mangle(fullName))
		if err2 != nil || !info.IsDir() {
			MissingPackages = append(MissingPackages, fullName)
		}
	}
	if len(MissingPackages) > 0 {
		return fmt.Errorf(">> Missing one or more packages. Clone them in the imports directory")
	}

	// Check that we have pkg and obj files for the imported packages.
	// If not, we have to compile the missing packages.
	MissingPackages = []string{}
	for _, fullName := range ImportedPackages {
		var info os.FileInfo
		info, err = os.Stat(*cacheDir + "/" + Mangle(fullName))
		if err == nil && info.IsDir() {
			var entries []os.DirEntry
			entries, err = os.ReadDir(*cacheDir + "/" + Mangle(fullName))
			if len(entries) == 0 {
				MissingPackages = append(MissingPackages, Mangle(fullName))
			}
			// for _, entry := range entries {
			// TODO: Check that the source code is older than the obj files
			// }
		} else {
			MissingPackages = append(MissingPackages, Mangle(fullName))
		}
	}
	if len(MissingPackages) > 0 {
		return fmt.Errorf("Missing packages")
	}

	// Now scan the pkg files for each import
	for _, fullName := range ImportedPackages {
		var info os.FileInfo
		info, err = os.Stat(*cacheDir + "/" + Mangle(fullName))
		if err == nil && info.IsDir() {
			var entries []os.DirEntry
			entries, err = os.ReadDir(*cacheDir + "/" + Mangle(fullName))
			for _, entry := range entries {
				if !entry.IsDir() && strings.Contains(entry.Name(), ".pkg") {
					fileName := *cacheDir + "/" + Mangle(fullName) + "/" + entry.Name()
					w := strings.Split(fullName, "/")
					packageName := w[len(w)-1]
					s, err2 := NewState(fileName, packageName)
					if err2 != nil {
						return err2
					}
					return ScanFile(s, fullName, true)
				}
			}
		}
	}

	return nil
}

func ScanFile(s *State, name string, pkg bool) (err error) {
	s.nextChar()
	s.next()
	if s.token == TOK_PACKAGE {
		s.next()
		if s.token != TOK_ID {
			return fmt.Errorf("%s:%d %v", name, code.LineNum, "package name should be on top line")
		}
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
	EmitPrologue(libPath, false)
	InitTypes()
	FuncInit()
	ImportInit()
	return nil
}

func OutputPkgFile(path string) error {
	OutputFile, err := os.Create(path + "/import.pkg")
	if err != nil {
		return err
	}
	defer OutputFile.Close()
	for _, t := range TypeDefs {
		if !t.Basic {
			if t.Pt == code.TYP_STRUCT {
				_, _ = OutputFile.WriteString("type " + t.TypeName + " = struct { ")
				i := 0
				for _, field := range t.Fields {
					i++
					_, _ = OutputFile.WriteString(field.Name() + " " + t.Name())
					if i < len(t.Fields) {
						_, _ = OutputFile.WriteString(", ")
					}
				}
				_, _ = OutputFile.WriteString("}\n")
			} else {
				_, _ = OutputFile.WriteString("type " + t.TypeName + " = " + t.Pt.Name() + "\n")
			}
		}
	}
	for _, f := range FuncDefs {
		if !f.builtin {
			_, _ = OutputFile.WriteString("func " + f.name + "(")
			for i, a := range f.parameters {
				_, _ = OutputFile.WriteString(a.name + " " + a.typ.Name())
				if i < len(f.parameters)-1 {
					_, _ = OutputFile.WriteString(", ")
				}
			}
			_, _ = OutputFile.WriteString(") ")
			for i, r := range f.returnTypes {
				_, _ = OutputFile.WriteString(r.TypeName)
				if i < len(f.returnTypes)-1 {
					_, _ = OutputFile.WriteString(", ")
				}
			}
			_, _ = OutputFile.WriteString(" {}\n")
		}
	}
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
func CompileFile(buildDir string, libPath string, fileName string) error {
	fmt.Printf(">>> Compiling %s\n", fileName)
	err := InitCompile(buildDir, libPath, fileName)
	if err != nil {
		return err
	}
	s, err := NewState(fileName, "main")
	if err != nil {
		return err
	}
	defer func(s *State) {
		_ = code.CloseAsmFile()
	}(s)
	err = ScanFile(s, fileName, false)
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
func CompileDir(packageName string, buildDir string, libPath string, inputPath string) error {
	var s *State
	var f *os.File
	inputFiles, err := os.ReadDir(inputPath)
	if err != nil {
		return fmt.Errorf("CompileDir() could not open source directory, %v", err.Error())
	}
	count := 0
	for count == 0 && err == nil {
		fmt.Printf(">>> Compiling package %s\n", inputPath)
		err = InitCompile(buildDir, libPath, "main")
		if packageName == "main" {
			emit("global", "main", "", "")
		}

		if err != nil {
			return err
		}
		for _, inputFile := range inputFiles {
			if !inputFile.IsDir() {
				fileName := filepath.Join(inputPath, inputFile.Name())
				f, err = os.Open(fileName)
				if err != nil {
					return fmt.Errorf("could not open directory, %v", err.Error())
				}
				s, err = NewState(fileName, "main")
				if err != nil {
					_ = f.Close()
					return err
				}
				err = ScanFile(s, fileName, false)
				_ = f.Close()
				if err != nil {
					break
				}
			}
		}
		if err == nil {
			err = OutputEpilogue()
			if err != nil {
				return err
			}
			err = code.CloseAsmFile()
			if err != nil {
				return err
			}
			if packageName != "main" {
				err = OutputPkgFile(buildDir)
				if err != nil {
					return err
				}
			}
			count++
		} else if err.Error() == "Missing packages" {
			_ = code.CloseAsmFile()
			for _, name := range MissingPackages {
				err = CompileDir(GetPkgName(name), "cache/"+Mangle(name), libPath, "imports/"+Mangle(name))
			}
		} else {
			return err
		}
	}
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
