package main

import (
	"flag"
	"fmt"
	"log"
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

// CompileDir will compile all source files in the given directory
// and put the object files in the outputPath
func CompileFailDir(inputPath string, workDir string) error {
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
			if err == nil {
				return fmt.Errorf("Expected %s to return error when compiled, but it did not", name)
			}
			fmt.Printf("File %s failed with error %v\n", name, err)
		}
	}
	return nil
}

// Compile all files in the test directory
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
						return n, fmt.Errorf("Expected %s to return error when compiled, but it did not", name)
					}
					fmt.Printf("File %s failed with error %v\n", name, err)
				} else {
					err = Build(workDir, name)
					if err != nil {
						return n, fmt.Errorf("Error in  %s : %s", name, err.Error())
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
	if !strings.Contains(outputName, ".") {
		outputName += ".exe"
	}
	// Calculate all arguments to the linker
	var args []string
	if *UseGcc {
		// Add all object files to argument list
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
		args = append(args, "-m64", "-lmsvcrt", "-o", outputPath)
		outp, err := exec.Command("../tools/MinGW64/bin/gcc.exe", args...).CombinedOutput()
		if len(outp) > 0 {
			fmt.Println(string(outp))
		}
		if err != nil {
			return fmt.Errorf("linking %s error: %s", outputName, err.Error())
		}
	} else {
		args = append(args, "/fo", outputName, "/entry=main", "/console")
		if *debug {
			args = append(args, "/debug=dbg")
		}
		// Add all object files to argument list
		entries, err := os.ReadDir(workDir)
		if err != nil {
			return fmt.Errorf("collecting obj files error %s", err.Error())
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.Contains(strings.ToUpper(entry.Name()), ".OBJ") {
				args = append(args, filepath.Join(workDir, entry.Name()))
			}
		}
		args = append(args, "-g", "kernel32.dll", "msvcrt.dll")
		// Print the arguments and the command
		fmt.Println("Link command:")
		fmt.Printf("../tools/golink.exe ")
		for _, s := range args {
			fmt.Printf(" %s", s)
		}
		fmt.Printf("\n")
		// Now start the linker
		outp, err := exec.Command("../tools/golink.exe", args...).CombinedOutput()
		fmt.Println(string(outp))
	}
	return nil
}

// Run will start execution of the exe file made by the link step
func Run(outputName string) error {
	cwd, _ := os.Getwd()
	fmt.Printf("Running \"%s\" in \"%s\"\n", outputName, cwd)
	fmt.Printf("--------------------------------------\n")
	out, err := exec.Command(path.Join(cwd, outputName), "").CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Printf("The exit code from '%s' was %d\n", outputName, err.(*exec.ExitError).ExitCode())
	}
	return err
}

func f1(s string) string {
	t := "f1 "
	return t + s
}

func f2(s string) string {
	t := " f2"
	return s + t
}

func main() {
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
	CreateBuildDir(*buildDir)

	// Now compile the source files into asm files
	if *oneFile != "" {
		fmt.Printf("=== Compiling %s ===\n", *oneFile)
		err = CompileFile(*oneFile, *buildDir)
		if err == nil {
			err = LinkRun(*buildDir, *oneFile)
		}
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
