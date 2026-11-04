package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jkvatne/jkv/code"
)

type PackageInfo struct {
	level         int
	mangledName   string // Mangled path mangledName, unique for each package
	sourcePath    string
	cachePath     string
	isCompiled    bool
	isStdLib      bool
	currentlyUsed bool
	alias         string
}

var Packages map[string]*PackageInfo

func init() {
	// Packages is a global map of all packages in the project
	// It is initialized only once, at start of compilation
	// The key is the mangled path name, unique for each package
	Packages = make(map[string]*PackageInfo)
}

func SetPackagesCurrentlyNotUsed() {
	for _, p := range Packages {
		p.currentlyUsed = false
	}
}

// AddPackage adds a package to the global Packages map
// It updates the Packages map and checks if it is already present in imports and cache
func AddPackage(cfg *Config, mangledName string, level int, alias string) error {
	p, ok := Packages[mangledName]
	if ok && p.level < level {
		return fmt.Errorf("circular package import: %s", DeMangle(mangledName))
	}
	if ok {
		p.currentlyUsed = true
		return nil
	}
	// Ok, create info.
	p = &PackageInfo{
		level:       level,
		alias:       alias,
		mangledName: mangledName,
	}

	// Check if the package exists either in the library or in the import directory
	p.sourcePath = path.Join(cfg.LibPath, mangledName)
	if strings.Contains(mangledName, "$") {
		p.sourcePath = path.Join(cfg.ImportPath, mangledName)
	}
	_, err := os.Stat(p.sourcePath)
	if err != nil {
		return fmt.Errorf("package %s not found. Clone it into the imports directory", p.sourcePath)
	}

	// Check if it is compiled and exists in the cache
	// TODO Check file dates
	p.cachePath = path.Join(cfg.CachePath, mangledName)
	_, err = os.Stat(p.cachePath)
	p.isCompiled = err == nil
	p.currentlyUsed = true
	// Save info
	Packages[mangledName] = p
	return nil
}

func ParseImport(s *State) error {
	if s.token != TOK_ID && s.token != TOK_STRING {
		return fmt.Errorf("expected id but got %s", s.tokenString)
	}
	packageName := s.tokenString
	// shortName defaults to the last part of the path.
	shortName := packageName
	s.next()
	for s.token == TOK_DIV || s.token == TOK_DOT {
		if s.token == TOK_DIV {
			s.next()
			packageName = packageName + "/" + s.tokenString
		} else {
			s.next()
			packageName = packageName + "." + s.tokenString
		}
		// shortName defaults to the last part of the path.
		shortName = s.tokenString
		s.next()
	}
	// path is now the imported path, like f.ex. "github.com/jkvatne/lib"
	if s.token == TOK_AS {
		// Get alternative shortname, if given after "as"
		s.next()
		if s.token != TOK_ID {
			return fmt.Errorf("expected package short name, but got %s", s.tokenString)
		}
		shortName = s.tokenString
	}
	packageName = Mangle(packageName)
	s.packages[shortName] = packageName
	err := AddPackage(s.Cfg, packageName, s.Level, shortName)
	return err
}

// ParseImports parses import statements and add package to the global Packages map
func ParseImports(s *State) error {
	var err error
	for s.token == TOK_IMPORT {
		s.next()
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
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CollectFileImports parses one file's import statements and add package to the global Packages map
// It does not generate any code
func CollectFileImports(cfg *Config, sourceFileName string, mangledName string, level int) error {
	s, err := NewState(cfg, sourceFileName, mangledName, level)
	if err != nil {
		return err
	}
	s.nextChar()
	s.next()
	if s.token == TOK_PACKAGE {
		s.next()
		if s.token != TOK_ID {
			return fmt.Errorf("%s:%d %v", sourceFileName, code.LineNum, "package name should be on top line")
		}
		if s.tokenString != mangledName {
			return fmt.Errorf("%s:%d %v", sourceFileName, code.LineNum,
				"got package name %s, should be %s on top line", s.tokenString, s.MangledName)
		}
		s.next()
	}
	err = ParseImports(s)
	return err
}

func CollectPackageImports(cfg *Config, packagePath string, mangledName string, level int) error {
	inputFiles, err := os.ReadDir(packagePath)
	if err != nil {
		return err
	}
	for _, file := range inputFiles {
		// For each jkv file in the package directory
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".jkv") {
			fileName := filepath.Join(packagePath, file.Name())
			err = CollectFileImports(cfg, fileName, mangledName, level+1)
			if err != nil {
				return err
			}
		}
	}
	// Recursively collect imported packages
	newPackages := make([]*PackageInfo, 0, 12)

	for _, p := range Packages {
		if p.level > level {
			newPackages = append(newPackages, p)
		}
	}

	for _, p := range newPackages {
		err = CollectPackageImports(cfg, p.sourcePath, p.mangledName, level+1)
		if err != nil {
			return err
		}
	}
	return nil
}

// ProcessPackage will scan each file in the package directory and collect imports
// If any package is not compiled, it will be compiled now
// If all packages are ok and the obj file exists, we can compile this package
func ProcessPackage(cfg *Config, mangledPackageName string, packagePath string, outputPath string) error {
	var s *State
	inputFiles, err := os.ReadDir(packagePath)
	if err != nil {
		return err
	}
	level := 0
	SetPackagesCurrentlyNotUsed()
	for _, file := range inputFiles {
		err = CollectFileImports(cfg, packagePath+"/"+file.Name(), mangledPackageName, level)
		if err != nil {
			return err
		}
	}
	// Now we have collected all packages used by all source files.
	// Compile/build those not ok
	packages := make([]*PackageInfo, 0, 12)
	for _, p := range Packages {
		if !p.isCompiled && p.currentlyUsed {
			packages = append(packages, p)
		}
	}
	for _, p := range packages {
		err = ProcessPackage(cfg, p.mangledName, p.sourcePath, p.cachePath)
		if err != nil {
			return err
		}
	}

	// Now we should have all packages compiled and can compile this one
	if mangledPackageName == "main" {
		err = InitCompile(cfg.BuildPath + "/" + "main.asm")
	} else {
		err = os.MkdirAll(cfg.CachePath+"/"+mangledPackageName, os.ModePerm)
		if err != nil {
			return fmt.Errorf("could not create package directory " + cfg.CachePath + "/" + mangledPackageName)
		}
		err = InitCompile(outputPath + "/" + ShortName(mangledPackageName) + ".asm")
	}
	if err != nil {
		return err
	}
	for _, file := range inputFiles {
		// For each jkv file in the package directory
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".jkv") {
			sourceFilePath := path.Join(packagePath, file.Name())
			s, err = NewState(cfg, sourceFilePath, mangledPackageName, level)
			if err != nil {
				return err
			}
			err = ScanFile(s, packagePath+"/"+file.Name())
			if err != nil {
				return err
			}
			// After scanning file, we can output the epilogue
			err = OutputEpilogue(s.MangledName)
			if err != nil {
				return err
			}
			// And close the asm file
			err = code.CloseAsmFile()
		}
	}

	err = OutputPkgFile(outputPath, mangledPackageName)
	// Make obj files
	err = AssembleFiles(cfg, outputPath, outputPath)
	return err
}

func ShortName(mangledName string) string {
	w := strings.Split(mangledName, "$")
	if len(w) < 2 {
		return mangledName
	}
	return w[len(w)-1]
}

// CompileDir will compile a package in the given directory.
// The package can consist of several source code files.
// The output is a single assembly file in the buildDir
func CompileDir(cfg *Config, inputPath string) (err error) {
	err = CollectPackageImports(cfg, inputPath, "main", 0)
	err = ProcessPackage(cfg, "main", inputPath, cfg.BuildPath)
	return err
}

func DeMangleFun(mangledName string) (path string, fun string) {
	w := strings.Split(mangledName, SepChar)
	if len(w) != 2 {
		return mangledName, mangledName
	}
	path = w[0]
	fun = w[1]
	return DeMangle(path), fun
}

func MangleFun(path string, fun string) string {
	return Mangle(path) + SepChar + fun
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

func ScanPackages(cfg *Config, level int) error {
	// Search for pkg files for each imported package
	// Should never fail since we already have built all packages
	for _, p := range Packages {
		if p.currentlyUsed {
			// Get the pkg file
			s, err := NewState(cfg, p.cachePath+"/import.pkg", p.mangledName, level)
			s.ScanPkg = true
			if err != nil {
				return err
			}
			code.NoCode++
			err = ScanFile(s, p.sourcePath)
			if err != nil {
				return err
			}
			code.NoCode--
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
	SetPackagesCurrentlyNotUsed()
	// Imports must be at top of file
	err = ParseImports(s)
	if err != nil {
		return err
	}
	err = ScanPackages(s.Cfg, s.Level)
	if err != nil {
		return err
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

func InitCompile(asmFilePath string) error {
	err := code.NewAsmFile(asmFilePath)
	if err != nil {
		return err
	}
	InitVardefs()
	InitTypes()
	LiteralInit()
	EmitPrologue()
	InitTypes()
	FuncInit()
	SetPackagesCurrentlyNotUsed()
	return nil
}

func OutputPkgFile(path string, packageName string) error {
	OutputFile, err := os.Create(path + "/import.pkg")
	if err != nil {
		return fmt.Errorf("could not create package file " + path + "/import.pkg")
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
			fn := f.name // MangleFun(packageName, f.name)
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

func OutputEpilogue(mangledName string) error {
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
	if mangledName != "main" && mangledName != "" {
		for _, f := range FuncDefs {
			if !f.builtin {
				EmitGlobal(f.label)
			}
		}
	}
	EmitExternals()
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
			err = CreateBuildDir(cfg.BuildPath)
			if err != nil {
				return n, fmt.Errorf("could not create build directory '%s'", err.Error())
			}
			err = CopyFile(cfg.CachePath+"/sys/sys.obj", cfg.BuildPath+"/sys.obj")
			if err != nil {
				return n, fmt.Errorf("could not copy sys.obj to build directory '%s'", err.Error())
			}
			name := filepath.Join(sourcePath, entry.Name())
			if strings.HasSuffix(name, ".jkv") {
				outputName := strings.TrimSuffix(filepath.Base(name), ".jkv")
				err = CompileFile(cfg, name)
				if strings.Contains(name, "err_") {
					if err == nil {
						return n, fmt.Errorf("expected %s to return error when compiled, but it did not", name)
					}
					fmt.Printf("File %s failed with error %v\n", name, err)
					err = nil
				} else {
					err = BuildLinkRun(cfg, outputName)
				}
			}
		} else {
			// Enter subdirectory and recursively compile the contents
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

func DropExt(filePath string) string {
	name, _, _ := strings.CutLast(filePath, ".")
	return name
}

// CompileFile will compile and run a single file. It must have a main() function.
func CompileFile(cfg *Config, filePath string) error {
	fmt.Printf(">>> Compiling %s\n", filePath)
	fileName := filepath.Base(filePath)
	err := InitCompile(cfg.BuildPath + "/" + DropExt(fileName) + ".asm")
	if err != nil {
		return err
	}
	s, err := NewState(cfg, filePath, "main", 0)
	if err != nil {
		return err
	}
	defer func(s *State) {
		_ = code.CloseAsmFile()
	}(s)
	err = ScanFile(s, fileName)
	if err != nil {
		return err
	}
	err = OutputEpilogue(s.MangledName)
	if err != nil {
		return err
	}
	return nil
}
