package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/bits"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// Version must be manually updated.
const Version string = "v0.0.2"

var (
	buildDir  = flag.String("build", "./build", "Path to intermediate files during build")
	run       = flag.Bool("run", true, "Set true to run after compile")
	test      = flag.Bool("test", false, "Set true to run after compile")
	link      = flag.Bool("link", true, "Set true to just do linking")
	sourceDir = flag.String("src", "", "Source directory where code is found. Defaults to current directory.")
	oneFile   = flag.String("file", "", "Compile a single file")
	debug     = flag.Bool("debug", false, "Enable debug mode")
	UseGcc    = flag.Bool("gcc", true, "Use gcc")
	UseUcrt   = flag.Bool("ucrt", false, "Use gcc")
	UseGoLink = flag.Bool("golink", false, "Use gcc")
	PrintSp   = flag.Bool("sp", false, "Print program SP")
)

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

func LinkRun(workDir string, outputName string) (err error) {
	// Assemble/link the files
	outputPath := path.Join(workDir, outputName)
	if *link {
		err = Assemble(workDir)
		if err == nil {
			err = Link(workDir, outputName)
		}
	}
	if err == nil && *run {
		// Run the exe file if -run is present and linking is ok
		err = Run(outputPath)
	}
	return err
}

func Build(workDir string, fileName string) (err error) {
	fmt.Printf("Build '%s'\n", fileName)
	outputName := strings.TrimSuffix(filepath.Base(fileName), ".jkv") + ".exe"
	// Make sure output directory is empty
	CreateBuildDir(workDir)
	err = CompileFile(fileName, workDir)
	if err == nil {
		err = LinkRun(workDir, outputName)
	}
	return err
}

// CompileDir will compile all source files in the given directory
// and put the object files in the outputPath
func CompileDir(inputPath string, workDir string) error {
	outputName := path.Base(inputPath)
	// Make sure output directory is empty
	_ = os.RemoveAll(workDir)
	err := os.Mkdir(workDir, os.ModePerm)
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return fmt.Errorf("fatal error %s", err.Error())
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			name := filepath.Join(inputPath, entry.Name())
			err = CompileFile(name, workDir)
			if err != nil {
				return err
			}
			fmt.Printf("File %s compiled ok\n", name)
		}
	}
	return LinkRun(workDir, outputName)
}

// CompileTests will compile all files in the test directory
// Files starting with err_ should intentionally fail
// Uses the build directory for outputs
func CompileTests(inputPath string, workDir string) (int, error) {
	n := 0
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return n, fmt.Errorf("fatal error %s", err.Error())
	}
	for _, entry := range entries {
		// For each jkv file in the test directory
		if !entry.IsDir() {
			n++
			name := filepath.Join(inputPath, entry.Name())
			if strings.HasSuffix(name, ".jkv") {
				if strings.Contains(name, "err_") {
					err = Build(workDir, name)
					if err == nil {
						return n, fmt.Errorf("expected %s to return error when compiled, but it did not", name)
					}
					fmt.Printf("File %s failed with error %v\n", name, err)
				} else {
					err = Build(workDir, name)
					if err != nil {
						return n, fmt.Errorf("error in  %s : %s", name, err.Error())
					}
				}
			}
		}
	}
	return n, err
}

// Assemble wil run the assembler on all *.asm files in the working directory
// And also the syscall.asm from /tools
func Assemble(workDir string) error {
	entries, err := os.ReadDir(workDir)
	if err != nil {
		return fmt.Errorf("collecting asm files error %s", err.Error())
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.Contains(entry.Name(), ".asm") {
			var args = []string{"-f", "win64"}
			name := filepath.Join(workDir, strings.TrimSuffix(entry.Name(), ".asm"))
			args = append(args, name+".asm", "-o", name+".obj")
			out, err := exec.Command("../tools/nasm.exe", args...).CombinedOutput()
			if len(out) > 0 {
				fmt.Println(string(out))
			}
			if err != nil {
				return fmt.Errorf("%s: %s", name, err.Error())
			}
		}
	}
	return nil
}

// Link will link all obj files and generate an exe file
func Link(workDir string, outputName string) error {
	// Make sure the output name includes .exe
	if !strings.HasSuffix(outputName, ".exe") {
		outputName += ".exe"
	}

	// Add all object files to argument list
	var args []string
	entries, err := os.ReadDir(workDir)
	if err != nil {
		return fmt.Errorf("collecting obj files error %s", err.Error())
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.Contains(strings.ToUpper(entry.Name()), ".OBJ") {
			args = append(args, filepath.Join(workDir, entry.Name()))
		}
	}

	outputPath := path.Join(workDir, outputName)
	LinkerName := "../tools/"
	if *UseGcc {
		LinkerName += "MinGW64/bin/gcc.exe"
		args = append(args, "-m64", "-lkernel32", "-lmsvcrt", "-o", outputPath)
	} else if *UseUcrt {
		LinkerName = "MinGW64/bin/gcc.exe"
		args = append(args, "-lkernel32", "-llegacy_stdio_definitions", "-lmsvcrt")
		args = append(args, "-DUCRT", "-m64", "-o", outputPath)
	} else if *UseGoLink {
		LinkerName = "golink.exe"
		args = append(args, "/fo", outputPath, "/entry=main", "/console")
		if *debug {
			args = append(args, "/debug=dbg")
		}
		args = append(args, "-g", "kernel32.dll", "msvcrt.dll") //  "legacy_stdio_definitions.lib",
		// Print the arguments and the command
	} else {
		fmt.Printf("Must specify either gcc, golink or ucrt")
	}

	// Print link command line to console
	/*
		fmt.Printf(LinkerName + " ")
		for _, s := range args {
			fmt.Printf(" %s", s)
		}
		fmt.Printf("\n")
	*/
	// Now start the linker
	output, err := exec.Command(LinkerName, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("linking %s error: %s", outputName, err.Error())
	}

	// Print linker output to console
	if len(output) > 0 {
		fmt.Println(string(output))
	}

	return nil
}

// Run will start execution of the exe file made by the link step
func Run(outputName string) error {
	cwd, _ := os.Getwd()
	// fmt.Printf("Running \"%s\" in \"%s\"\n", outputName, cwd)
	out, err := exec.Command(path.Join(cwd, outputName), "").CombinedOutput()
	fmt.Printf("%s", string(out))
	if err != nil {
		fmt.Printf("The exit code from '%s' was %d\n", outputName, err.(*exec.ExitError).ExitCode())
	}
	return err
}

// unpack64 returns m, e such that f = m * 2**e.
// The caller is expected to have handled 0, NaN, and ±Inf already.
// To unpack a float32 f, use unpack64(float64(f)).
func unpack64(f float64) (uint64, int) {
	const shift = 64 - 53
	const minExp = -(1074 + shift)
	b := math.Float64bits(f)
	m := 1<<63 | (b&(1<<52-1))<<shift
	e := int((b >> 52) & (1<<shift - 1))
	if e == 0 {
		m &^= 1 << 63
		e = minExp
		s := 64 - bits.Len64(m)
		return m << s, e - s
	}
	return m, (e - 1) + minExp
}

func GoTests() {
	m := uint64(0x7ff)
	e := bits.Len64(m)
	fmt.Printf("%d bits\n", e)

	m, e = unpack64(1.75)
	fmt.Printf("m=0x%x, e=%d\n", m, e)
}

func main() {
	/*s := "'`´\""
	for i, ch := range s {
		fmt.Printf("%d: %s  0x%x \n", i, string(ch), int(ch))
	}*/

	flag.Parse()
	// Set logger to not prepend any time/date
	log.SetFlags(0)

	wd, err := os.Getwd()
	fmt.Printf("Starting jkv compiler version %s, in \"%s\"\n", Version, wd)
	if *sourceDir == "" {
		*sourceDir = wd
	}

	// Expand temporary build directory path
	//	*buildDir, err = filepath.Abs(*buildDir)
	if err != nil {
		fmt.Printf("could expand working directory " + err.Error())
		os.Exit(1)
	}

	// Now compile the source files into asm files
	if *oneFile != "" {
		if !strings.Contains(*oneFile, ".") {
			*oneFile += ".jkv"
		}
		err = Build(*buildDir, *oneFile)
	} else if *test {
		n := 0
		n, err = CompileTests(*sourceDir, *buildDir)
		if err == nil {
			fmt.Printf("------------------------------------------\n")
			fmt.Printf("Run %d files. All tests passed\n", n)
		}
	} else {
		err = CompileDir(*sourceDir, *buildDir)
	}
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

}
