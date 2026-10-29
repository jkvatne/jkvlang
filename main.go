package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
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
	buildDir     = flag.String("build", "c:/doc/compiler/build", "Path to intermediate files during build")
	cacheDir     = flag.String("cache", "", "Cache directory")
	sourceDir    = flag.String("dir", ".", "Source directory where code is found. Defaults to current directory.")
	importDir    = flag.String("import", "", "Path to imported non-system code")
	oneFile      = flag.String("file", "", "Compile a single file")
	run          = flag.Bool("run", true, "Set true to run after compile")
	test         = flag.Bool("test", false, "Set true to run after compile")
	clean        = flag.Bool("clean", false, "Set true to recompile all imports")
	debug        = flag.Bool("debug", false, "Enable debug mode")
	linker       = flag.String("linker", "gcc", "Name of linker. Default gcc, alternatives golink, ucrt, msvc")
	arg          = flag.String("arg", "dummyarg", "Arguments to the compiled program when it is run")
	printLinkCmd = flag.Bool("printlinkcmd", true, "Print link command")
)

type Config struct {
	ExePath    string // Where the compiler is found
	LibPath    string // The sys.asm and system packages, typically a subdirectory to ExePath
	CachePath  string // Cache directory, a temporary location. Where all obj, pkg and asm files are found
	BuildPath  string // Build directory, cleared for each run
	ImportPath string // Where imported non-system code is found
	Run        bool   // Set true to run after build
}

func CopyFile(src string, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

func Build(cfg *Config, inputDir string) error {
	// Assemble cached packages (if needed)
	entries, err := os.ReadDir(cfg.CachePath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			dir := path.Join(cfg.CachePath, entry.Name())
			err = AssembleFiles(cfg, dir, cfg.BuildPath)
			if err != nil {
				return err
			}
		}
	}
	// Assemble program
	err = AssembleFiles(cfg, cfg.BuildPath, cfg.BuildPath)
	return err
}

// AssembleFiles wil run the assembler on all *.asm files in the input directory
// And also the syscall.asm from /tools
func AssembleFiles(cfg *Config, inputDir string, outputDir string) error {
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("collecting asm files error, could not read  %s", inputDir)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.Contains(entry.Name(), ".asm") {
			var args = []string{"-f", "win64"}
			asmName := filepath.Join(inputDir, entry.Name())
			// outName := filepath.Join(outputDir,	strings.TrimSuffix(entry.Name(), ".asm")+".obj")
			outName := outputDir + "/" + strings.TrimSuffix(entry.Name(), ".asm") + ".obj"
			args = append(args, asmName, "-o", outName)
			out, err := exec.Command(cfg.ExePath+"/tools/nasm.exe", args...).CombinedOutput()
			if len(out) > 0 {
				fmt.Println(string(out))
			}
			if err != nil {
				return fmt.Errorf("%s: %s", asmName, err.Error())
			}
		}
	}
	return nil
}

// Link will link all obj files in the build directory and generate an exe file
func Link(cfg *Config, outputName string) error {
	// Make sure the output name includes .exe
	if !strings.HasSuffix(outputName, ".exe") {
		outputName += ".exe"
	}

	// Add all object files to argument list
	var args []string
	entries, err := os.ReadDir(cfg.BuildPath)
	if err != nil {
		return fmt.Errorf("collecting obj files error %s", err.Error())
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.Contains(strings.ToUpper(entry.Name()), ".OBJ") {
			args = append(args, filepath.Join(cfg.BuildPath, entry.Name()))
		}
	}
	outputPath := path.Join(cfg.BuildPath, outputName)
	LinkerName := "c:/doc/compiler/tools/"
	if *linker == "gcc" {
		LinkerName += "MinGW64/bin/gcc.exe"
		args = append(args, "-mconsole", "-m64", "-lkernel32", "-lmsvcrt", "-o", outputPath)
	} else if *linker == "ucrt" {
		LinkerName = "MinGW64/bin/gcc.exe"
		args = append(args, "-lkernel32", "-llegacy_stdio_definitions", "-lmsvcrt")
		args = append(args, "-DUCRT", "-m64", "-o", outputPath)
	} else if *linker == "golink" {
		LinkerName = "./tools/golink.exe"
		args = append(args, "/fo", outputPath, "/entry=main", "/console")
		if *debug {
			args = append(args, "/debug=dbg")
		}
		args = append(args, "kernel32.dll", "msvcrt.dll") //  "legacy_stdio_definitions.lib",

	} else if *linker == "msvc" {
		LinkerName = "C:/Program Files (x86)/Microsoft Visual Studio/18/BuildTools/VC/Tools/MSVC/14.51.36231/bin/Hostx86/x86/link.exe"
		entries, err = os.ReadDir(cfg.LibPath)
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".obj") {
				args = append(args, path.Join(cfg.LibPath, entry.Name()))
			}
		}
		args = append(args, "/out:"+outputPath, "/entry:main")
		args = append(args, "/SUBSYSTEM:CONSOLE")
		args = append(args, "C:/Program Files (x86)/Windows Kits/10/Lib/10.0.26100.0/um/x64/kernel32.Lib")
		args = append(args, "C:/Program Files (x86)/Windows Kits/10/Lib/10.0.26100.0/um/x64/ucrt.Lib")
		args = append(args, "C:/Program Files (x86)/Windows Kits/10/Lib/10.0.26100.0/um/x64/legacy_stdio_definitions.lib")
		if *debug {
			args = append(args, "/debug")
		}
		// args = append(args, "legacy_stdio_definitions.lib")
	} else {
		fmt.Printf("Must specify either gcc, golink or ucrt")
	}

	// Print link command line to console
	if *printLinkCmd {
		fmt.Printf("Linker command line:\n" + LinkerName + " ")
		for _, s := range args {
			fmt.Printf(" %s", s)
		}
		fmt.Printf("\n")
	}
	// Now start the linker
	output, err := exec.Command(LinkerName, args...).CombinedOutput()
	if err != nil {
		println("\n" + string(output))
		return fmt.Errorf("linking %s error: %s", outputName, err.Error())
	}

	// Print linker output to console
	if len(output) > 0 {
		fmt.Println(string(output))
	}

	return nil
}

var cmd *exec.Cmd
var stdin io.WriteCloser
var stdout io.ReadCloser

func monitor(reader *bufio.Reader) {
	for cmd != nil {
		line, err := reader.ReadString('\n')
		fmt.Printf(line)
		if err != nil {
			return
		}
	}
}

// Run will start execution of the exe file made by the link step
// It must be placed in the build directory
func Run(outputName string) error {
	if *run {
		var err error
		cmd = exec.Command(outputName, *arg)
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("StdinPipe() failed : %v", err)
		}
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("StdoutPipe() failed : %v", err)
		}
		reader := bufio.NewReader(stdout)
		err = cmd.Start()
		if err != nil {
			return fmt.Errorf("Start() failed : %s", err.Error())
		}
		go monitor(reader)
		_, err = cmd.Process.Wait()
		cmd = nil
	}
	return nil
}

func main() {

	// TestDecode()
	// TestUnrounded()
	// TestFormat()

	flag.Parse()
	// Set logger to not prepend any time/date
	log.SetFlags(0)

	wd, err := os.Getwd()
	fmt.Printf("Starting jkv compiler %s, in '%s'\n", Version, wd)
	if *sourceDir == "" {
		*sourceDir = wd
	}

	cfg := &Config{}
	cfg.ExePath, _ = os.Executable()
	cfg.ExePath = filepath.ToSlash(cfg.ExePath)
	cfg.ExePath = path.Dir(cfg.ExePath)
	cfg.LibPath = path.Join(cfg.ExePath, "lib")
	cfg.CachePath = *cacheDir
	if cfg.CachePath == "" {
		cfg.CachePath = cfg.ExePath + "/cache"
	}
	cfg.BuildPath = *buildDir
	cfg.ImportPath = cfg.ExePath + "/imports"

	if *clean {
		// Remove all cached obj files and re-build them from source
		err = os.RemoveAll(cfg.CachePath)
		if err != nil {
			panic("Unable to remove cache directory")
		}
		err = os.Mkdir(cfg.CachePath, os.ModePerm)
		if err != nil {
			panic("Unable to create cache directory")
		}
		err = os.Mkdir(cfg.CachePath+"/sys", os.ModePerm)
		// Now we have a clean cache directory. Remake sys.obj and other sys files
		// The obj files are put into cache/sys/...
		if err == nil {
			err = AssembleFiles(cfg, cfg.LibPath, cfg.CachePath+"/sys")
		}
	}
	if err == nil {
		err = CreateBuildDir(cfg.BuildPath)
	}
	if err == nil {
		err = CopyFile(cfg.CachePath+"/sys/sys.obj", cfg.BuildPath+"/sys.obj")
	}
	if err != nil {
		// Do nothing upon errors in previous steps
	} else if *oneFile != "" {
		// Compile a single source file
		if !strings.Contains(*oneFile, ".") {
			*oneFile += ".jkv"
		}
		err = CompileFile(cfg, *oneFile)
		if err == nil {
			outputName := strings.TrimSuffix(filepath.Base(*oneFile), ".jkv") + ".exe"
			Build(cfg, filepath.Dir(*oneFile))
			err = Link(cfg, outputName)
			if err == nil {
				err = Run(cfg.BuildPath + "/" + outputName)
			}
		}

	} else if *test {
		// Compile and run all files int the test directory
		n := 0
		n, err = CompileTests(cfg, *sourceDir)
		if err == nil {
			fmt.Printf("------------------------------------------\n")
			fmt.Printf("Run %d files. All tests passed\n", n)
		}

	} else if *sourceDir != "" {
		// Compile and run all files in the source directory
		err = CompileDir(cfg, *sourceDir, cfg.BuildPath, "main")
		if err == nil {
			err = Build(cfg, *sourceDir)
		}
		if err == nil {
			err = Link(cfg, "main")
		}
		if err == nil {
			err = Run(cfg.BuildPath + "/" + "main.exe")
		}
	} else {
		// Must have file, test or directory
		err = fmt.Errorf("Must have file, test or directory")
	}
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

}
