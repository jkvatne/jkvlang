package main

import (
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
	if len(w) != 2 {
		return mangledName, mangledName
	}
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

// ParseImports parses import statements
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
		if err2 != nil {
			info, err2 = os.Stat(s.Cfg.LibPath + "/imports/" + Mangle(fullName))
		}
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
		info, err = os.Stat(s.Cfg.CachePath + "/" + Mangle(fullName))
		if err != nil {
			info, err = os.Stat(s.Cfg.LibPath + "/imports/" + Mangle(fullName))
		}
		if err == nil && info.IsDir() {
			var entries []os.DirEntry
			entries, err = os.ReadDir(s.Cfg.CachePath + "/" + Mangle(fullName))
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
		return fmt.Errorf("missing packages")
	}

	// Now scan the pkg files for each import
	for _, fullName := range ImportedPackages {
		var info os.FileInfo
		info, err = os.Stat(s.Cfg.CachePath + "/" + Mangle(fullName))
		if err == nil && info.IsDir() {
			var entries []os.DirEntry
			entries, err = os.ReadDir(s.Cfg.CachePath + "/" + Mangle(fullName))
			for _, entry := range entries {
				if !entry.IsDir() && strings.Contains(entry.Name(), ".pkg") {
					fileName := s.Cfg.CachePath + "/" + Mangle(fullName) + "/" + entry.Name()
					w := strings.Split(fullName, "/")
					packageName := w[len(w)-1]
					//                   cfg  SourceFileName PackageName isPkgFile
					ss, err2 := NewState(s.Cfg, fileName, packageName, true)
					if err2 != nil {
						return err2
					}
					err2 = ScanFile(ss, fullName)
					if err2 != nil {
						return err2
					}
				}
			}
		}
	}

	return nil
}

func ScanFile(s *State, name string) (err error) {
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
			s.CollectNextLine()
			code.LineNum = code.NextLineNum
			err = ParseFuncDef(s)
		} else if s.token == TOK_CONST {
			s.CollectNextLine()
			code.LineNum = code.NextLineNum
			s.next()
			err = ParseConsts(s)
		} else if s.token == TOK_TYPE {
			s.CollectNextLine()
			code.LineNum = code.NextLineNum
			s.next()
			err = ParseTypeDefs(s)
		} else {
			err = fmt.Errorf("unexpected top level token \"%s\"", s.tokenString)
		}
		if err != nil {
			return fmt.Errorf("%s:%d %v", name, code.LineNum, err)
		}
	}
	return err
}

func CreateBuildDir(buildDir string) error {
	// Make sure output directory is empty
	err := os.RemoveAll(buildDir)
	if err != nil {
		return fmt.Errorf("could not remove old working directory " + err.Error())
	}
	err = os.Mkdir(buildDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not create working directory " + err.Error())
	}
	return nil
}

func InitCompile(buildDir string, AsmFileName string) error {
	err := code.NewAsmFile(AsmFileName, buildDir)
	if err != nil {
		return err
	}
	InitVardefs()
	InitTypes()
	LiteralInit()
	EmitPrologue()
	InitTypes()
	FuncInit()
	ImportInit()
	return nil
}

func OutputPkgFile(path string, packageName string) error {
	OutputFile, err := os.Create(path + "/import.pkg")
	if err != nil {
		return err
	}
	defer func(OutputFile *os.File) {
		_ = OutputFile.Close()
	}(OutputFile)
	for _, t := range TypeDefs {
		if !t.Basic {
			if t.Pt == code.TYP_STRUCT {
				_, _ = OutputFile.WriteString("type " + t.TypeName + " = struct { ")
				i := 0
				for name, field := range t.Fields {
					i++
					_, _ = OutputFile.WriteString(name + " " + field.TypeName + " ")
				}
				_, _ = OutputFile.WriteString("}\n")
			} else {
				_, _ = OutputFile.WriteString("type " + t.TypeName + " = " + t.Pt.Name() + "\n")
			}
		}
	}
	for _, f := range FuncDefs {
		if !f.builtin {
			fn := MangleFun(packageName, f.name)
			_, _ = OutputFile.WriteString("func " + fn + "(")
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

func OutputEpilogue(s *State) error {
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
	if s.IsPackage {
		for _, f := range FuncDefs {
			if !f.builtin {
				EmitGlobal(s.PackageName + "@" + f.label)
			}
		}
	}
	return nil
}

// CompileDir will compile a package in the given directory.
// The package can consist of several source code files.
// The output is a single assembly file in the buildDir
func CompileDir(cfg *Config, ip string, op string, pn string) error {
	var s *State
	inputPath := ip
	outputPath := op
	packageName := pn
	// Repeat if some imports were missing
	for {
		fmt.Printf(">>> Compiling package %s\n", inputPath)
		inputFiles, err := os.ReadDir(inputPath)
		if err != nil {
			return fmt.Errorf("CompileDir() could not open source directory, %v", err.Error())
		}
		err = InitCompile(outputPath, packageName)
		if err != nil {
			return err
		}
		for _, inputFile := range inputFiles {
			if !inputFile.IsDir() {
				fileName := filepath.Join(inputPath, inputFile.Name())
				if err != nil {
					return fmt.Errorf("could not open directory, %v", err.Error())
				}
				s, err = NewState(cfg, fileName, packageName, false)
				if err != nil {
					return err
				}
				err = ScanFile(s, fileName)
				if err != nil {
					break
				}
			}
		}
		if err == nil {
			if packageName != "main" {
				err = OutputPkgFile(outputPath, packageName)
				if err != nil {
					return err
				}
			}
			err = OutputEpilogue(s)
			if err != nil {
				return err
			}
			err = code.CloseAsmFile()
			break
		} else if err.Error() == "missing packages" {
			_ = code.CloseAsmFile()
			// First compile system packages
			for _, name := range MissingPackages {
				inp := cfg.LibPath + "/imports/" + Mangle(name)
				inputFiles, err = os.ReadDir(inp)
				if err == nil && len(inputFiles) != 0 {
					outp := cfg.CachePath + "/" + Mangle(name)
					err = os.MkdirAll(outp, os.ModePerm)
					if err != nil {
						return err
					}
					err = CompileDir(cfg, inp, outp, GetPkgName(name))
					if err != nil {
						return err
					}
				}
			}
			for _, name := range MissingPackages {
				inp := cfg.ImportPath + "/" + Mangle(name)
				inputFiles, err = os.ReadDir(inp)
				if err == nil && len(inputFiles) != 0 {
					outp := cfg.CachePath + "/" + Mangle(name)
					err = os.MkdirAll(outp, os.ModePerm)
					if err != nil {
						return err
					}
					err = CompileDir(cfg, inp, outp, GetPkgName(name))
					if err != nil {
						return err
					}
				}
			}
			err = nil
		} else {
			return err
		}
	}
	return nil
}

// CompileTests will compile all files in the test directory
// Files starting with err_ should intentionally fail
// Uses the build directory for outputs
func CompileTests(cfg *Config, sourcePath string) (int, error) {
	n := 0
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return n, fmt.Errorf("could not open source directory '%s'", err.Error())
	}
	for _, entry := range entries {
		// For each jkv file in the test directory
		if !entry.IsDir() {
			n++
			name := filepath.Join(sourcePath, entry.Name())
			if strings.HasSuffix(name, ".jkv") {
				outputName := strings.TrimSuffix(filepath.Base(name), ".jkv") + ".exe"
				err = CompileFile(cfg, name)
				if strings.Contains(name, "err_") {
					if err == nil {
						return n, fmt.Errorf("expected %s to return error when compiled, but it did not", name)
					}
					fmt.Printf("File %s failed with error %v\n", name, err)
					err = nil
				} else {
					if err == nil {
						err = Link(cfg, outputName)
					}
					if err != nil {
						return n, fmt.Errorf("error in  %s : %s", name, err.Error())
					}
					Run(outputName)
				}
			}
		} else {
			// Enter subdirectory
			m := 0
			m, err = CompileTests(cfg, sourcePath+"/"+entry.Name())
			n += m
			if err != nil {
				return n, err
			}
		}
	}
	return n, err
}

// CompileFile will compile and run a single file. It must have a main() function.
func CompileFile(cfg *Config, fileName string) error {
	fmt.Printf(">>> Compiling %s\n", fileName)
	err := InitCompile(cfg.BuildPath, strings.TrimSuffix(fileName, ".jkv"))
	if err != nil {
		return err
	}
	s, err := NewState(cfg, fileName, "main", false)
	if err != nil {
		return err
	}
	defer func(s *State) {
		_ = code.CloseAsmFile()
	}(s)
	err = ScanFile(s, fileName)
	if err.Error() == "missing packages" {
		_ = code.CloseAsmFile()
		for _, name := range MissingPackages {
			err = CompileDir(cfg, cfg.ImportPath+"/"+Mangle(name), cfg.CachePath+"/"+Mangle(name), GetPkgName(name))
			if err != nil {
				return err
			}
		}
	}
	if err != nil {
		return err
	}
	err = OutputEpilogue(s)
	if err != nil {
		return err
	}
	return nil
}

/*
func CompileLib(cfg *Config) error {
	entries, err := os.ReadDir(cfg.LibPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			err = CompileDir(cfg, cfg.LibPath+"/"+name, name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
*/
