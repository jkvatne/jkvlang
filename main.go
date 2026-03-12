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

// Version must be manualy updated.
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
	// Make sure output directory is empty
	err := os.RemoveAll(workDir)
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
	args = append(args, "kernel32.dll")
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
	wd, err := os.Getwd()
	fmt.Printf("Currrent directory is %s\n", wd)
	fmt.Printf("Running the executable %s:\n", outputName)
	out, err := exec.Command("./test.exe", "x").CombinedOutput()
	if err == nil {
		println(string(out))
	}
	return err
}

func main() {
	var err error
	flag.Parse()
	fmt.Printf("jkv compiler version %s\n", Version)

	// Set logger to not prepend any time/date
	log.SetFlags(0)

	// Now do the actual compile/link/run as requested by flags
	if *oneFile != "" {
		fmt.Printf("=== Compiling %s ===\n", oneFile)
		err = CompileFile(*oneFile, *workDir)
	} else if !*link {
		err = CompileDir(*inputPath, *workDir)
		if err == nil {
			*link = true
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
