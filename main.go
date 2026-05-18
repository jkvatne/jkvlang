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
	workDir = flag.String("build", "./build", "Path to intermediate files during build")
	run     = flag.Bool("run", true, "Set true to run after compile")
	test    = flag.Bool("test", false, "Set true to run after compile")
	link    = flag.Bool("link", true, "Set true to just do linking")
	// outputName = flag.String("o", "program.exe", "Output filename of exectutable")
	inputPath = flag.String("src", "./", "Source directory")
	oneFile   = flag.String("file", "", "Compile a single file")
	debug     = flag.Bool("debug", false, "Enable debug mode")
	UseGcc    = flag.Bool("gcc", true, "Use gcc")
	PrintSp   = flag.Bool("sp", false, "Print program SP")
)

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
			// fmt.Printf("Compiling %s\n", name)
			err = CompileFile(name, workDir)
			if err != nil {
				return err
			}
			fmt.Printf("File %s compiled ok\n", name)
		}
	}
	// Assemble/link the files
	if *link {
		err = Assemble(workDir)
		if err != nil {
			fmt.Printf(err.Error())
			os.Exit(2)
		}
		err = Link(workDir, outputName)
		if err != nil {
			fmt.Printf(err.Error())
			os.Exit(3)
		}
	}
	if *run {
		// Run the exe file if -run is present and linking is ok
		err = Run(outputName)
	}
	return nil
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

func CompileTests(inputPath string, workDir string) error {
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return fmt.Errorf("fatal error %s", err.Error())
	}
	for _, entry := range entries {
		// For each subdirectory in the test directory
		if entry.IsDir() {
			name := filepath.Join(inputPath, entry.Name())
			if name != "build" {
				fmt.Printf(">>> Compiling directory \"%s\"\n", name)
				if name != "fail" {
					err = CompileDir(name, workDir)
					if err != nil {
						return err
					}
					fmt.Printf("%s compiled ok\n", name)
				} else {
					// All files in the "fail" directory should return an error
					err = CompileFailDir(name, workDir)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return err
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
		args = append(args, "-o", outputName, "-m64", "-lkernel32", "-lmsvcrt")
		// C:/Program Files (x86)/SASM/MinGW64/bin/gcc.exe
		// C:/w64devkit/bin/gcc.exe
		// OK outp, err := exec.Command("C:/w64devkit/bin/gcc.exe", args...).CombinedOutput()
		outp, err := exec.Command("../tools/MinGW64/bin/gcc.exe", args...).CombinedOutput()
		if len(outp) > 0 {
			fmt.Println(string(outp))
		}
		if err != nil {
			return fmt.Errorf("linking %s error: %s", outputName, err.Error())
		}
	} else {
		args = append(args, "/fo")
		args = append(args, outputName)
		args = append(args, "/entry=main")
		args = append(args, "/console")
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
		args = append(args, "-g")
		args = append(args, "kernel32.dll")
		args = append(args, "msvcrt.dll")
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
	fmt.Printf("Running \"%s\" in \"%s\"\n", outputName+".exe", cwd)
	fmt.Printf("--------------------------------------\n")
	out, err := exec.Command(path.Join(cwd, outputName+".exe"), "").CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Printf("The exit code was %d\n", err.(*exec.ExitError).ExitCode())
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
	x := 1.0
	y := 2.2
	z := 3.3
	w := 1.0/y + (x+y/z)*(z*z*(x+y))
	fmt.Printf("Test: %f  (should be 58.534545)\n", w)

	flag.Parse()
	wd, err := os.Getwd()
	fmt.Printf("Starting jkv compiler version %s, in \"%s\"\n", Version, wd)

	// Expand working directory path
	*workDir, err = filepath.Abs(*workDir)
	if err != nil {
		fmt.Printf("could expand working directory " + err.Error())
		os.Exit(1)
	}

	// Set logger to not prepend any time/date
	log.SetFlags(0)

	// Make sure output directory is empty
	err = os.RemoveAll(*workDir)
	if err != nil {
		fmt.Printf("could not remove old working directory " + err.Error())
		os.Exit(1)
	}
	err = os.Mkdir(*workDir, os.ModePerm)
	if err != nil {
		fmt.Printf("could not create working directory " + err.Error())
		os.Exit(1)
	}

	// Now compile the source files into asm files
	if *oneFile != "" {
		fmt.Printf("=== Compiling %s ===\n", oneFile)
		err = CompileFile(*oneFile, *workDir)
	} else if *test {
		err = CompileTests(*inputPath, *workDir)
		if err == nil {
			fmt.Printf("------------------------------------------\n")
			fmt.Printf("All tests passed\n")
		}
	} else {
		err = CompileDir(*inputPath, *workDir)
	}
	if err != nil {
		fmt.Printf("%s%s\n", *oneFile, err.Error())
		os.Exit(1)
	}

}
