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
	workDir    = flag.String("build", "./build", "Path to intermediate files during build")
	run        = flag.Bool("run", true, "Set true to run after compile")
	link       = flag.Bool("link", true, "Set true to just do linking")
	outputName = flag.String("o", "program.exe", "Output filename of exectutable")
	inputPath  = flag.String("src", "./", "Source directory")
	oneFile    = flag.String("file", "", "Compile a single file")
	debug      = flag.Bool("debug", false, "Enable debug mode")
	UseGcc     = flag.Bool("gcc", true, "Use gcc")
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
			fmt.Println(string(out))
			if err != nil {
				return fmt.Errorf("assembly %s error: %s", name, err.Error())
			}
		}
	}
	// var args = []string{"-f", "win64", "-o syscall.obj", "../tools/syscall.asm"}
	// out, err := exec.Command("../tools/nasm.exe", args...).CombinedOutput()
	// fmt.Println(string(out))
	// if err != nil {
	//	return fmt.Errorf("assembly of syscall.asm, error: %s", err.Error())
	// }

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
		args = append(args, "-o")
		args = append(args, "program.exe")
		args = append(args, "-m64")
		args = append(args, "-lkernel32")
		args = append(args, "-lmsvcrt")
		// outp, err := exec.Command("C:/Program Files (x86)/SASM/MinGW64/bin/gcc.exe", args...).CombinedOutput()
		outp, err := exec.Command("C:/w64devkit/bin/gcc.exe", args...).CombinedOutput()

		fmt.Println(string(outp))
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
	fmt.Printf("Running \"%s\" in \"%s\"\n", outputName, cwd)
	out, err := exec.Command(path.Join(cwd, outputName), "").CombinedOutput()
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
	} else {
		err = CompileDir(*inputPath, *workDir)
		if err != nil {
			fmt.Printf("Error compiling %s %s\n", *oneFile, err.Error())
			os.Exit(1)
		}
	}

	// Assemble/link the files
	if *link {
		err = Assemble(*workDir)
		if err != nil {
			fmt.Printf("Assembler error " + err.Error())
			os.Exit(2)
		}
		err = Link(*workDir, *outputName)
		if err != nil {
			fmt.Printf("LInker error " + err.Error())
			os.Exit(3)
		}
	}

	// Run the exe file if -run is present and linking is ok
	if *run {
		err = Run(*outputName)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	}
}
