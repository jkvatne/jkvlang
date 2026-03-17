package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Version must be manually updated.
const Version string = "v0.0.2"

var (
	workDir    = flag.String("build", "./build", "Path to intermediate files during build")
	run        = flag.Bool("run", false, "Set true to run after compile")
	link       = flag.Bool("link", false, "Set true to just do linking")
	outputName = flag.String("o", "my_program", "Output filename of exectutable")
	inputPath  = flag.String("src", "./", "Source directory")
	oneFile    = flag.String("file", "", "Compile a single file")
)

// CompileDir will compile all source files in the given directory
// and put the object files in the outputPath
func CompileDir(inputPath string, workDir string) error {
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return fmt.Errorf("fatal error %s", err.Error())
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			name := filepath.Join(inputPath, entry.Name())
			fmt.Printf("=== Compiling %s ===\n", name)
			err = CompileFile(name, workDir)
			if err != nil {
				return err
			}
			fmt.Printf("File %s compiled ok\n", name)
		}
	}
	return err
}

// Assemble wil run the assembler on all *.asm files in the working directory
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
			fmt.Println(string(out))
			if err != nil {
				return fmt.Errorf("assembly %s error: %s", name, err.Error())
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
	var args = []string{"/fo", outputName}
	args = append(args, "/entry=_start")
	args = append(args, "/console")
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
	args = append(args, "C:/doc/compiler/tools/build/syscall.obj")
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
	out, err := exec.Command("../tools/golink.exe", args...).CombinedOutput()
	// Print linker output
	fmt.Println(string(out))
	return err
}

// Run will start execution of the exe file made by the link step
func Run(outputName string) error {
	fmt.Printf("Running %s:\n", outputName)
	out, err := exec.Command(outputName, "").CombinedOutput()
	println(string(out))
	return err
}

func main() {
	var err error
	flag.Parse()
	fmt.Printf("jkv compiler version %s\n", Version)
	wd, err := os.Getwd()
	fmt.Printf("Currrent directory is %s\n", wd)

	// Expand working directory path
	*workDir, err = filepath.Abs(*workDir)
	if err != nil {
		fmt.Printf("could expand working directory " + err.Error())
		os.Exit(1)
	}

	// Set logger to not prepend any time/date
	log.SetFlags(0)

	// Make sure output directory is empty, except when only linking
	if !*link {
		err = os.RemoveAll(*workDir)
		if err != nil {
			fmt.Printf("could not remove old working directory " + err.Error())
			os.Exit(1)
		}
	}
	err = os.Mkdir(*workDir, os.ModePerm)
	if err != nil {
		fmt.Printf("could not create working directory " + err.Error())
		os.Exit(1)
	}

	// Now do the actual compile/link/run as requested by flags
	if *oneFile != "" {
		fmt.Printf("=== Compiling %s ===\n", oneFile)
		err = CompileFile(*oneFile, *workDir)
	} else if !*link {
		err = CompileDir(*inputPath, *workDir)
		if err == nil {
			*link = true
		} else {
			fmt.Printf("Error compiling %s %s\n", *oneFile, err.Error())
		}
	}

	// Assemble the files
	if *link {
		err = Assemble(*workDir)
		if err != nil {
			fmt.Printf("Assembler error " + err.Error())
			os.Exit(1)
		}
	}

	// Link object files
	if *link {
		err = Link(*workDir, *outputName)
	}

	// Run the exe file if -run is present and linking is ok
	if err == nil && *run {
		err = Run(*outputName)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	}
}
