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
	"time"
)

// Version must be manually updated.
const Version string = "v0.0.2"

var (
	buildDir  = flag.String("build", "c:/doc/compiler/build", "Path to intermediate files during build")
	run       = flag.Bool("run", true, "Set true to run after compile")
	test      = flag.Bool("test", false, "Set true to run after compile")
	link      = flag.Bool("link", true, "Set true to just do linking")
	linklib   = flag.Bool("linklib", true, "Set true to just do linking")
	sourceDir = flag.String("dir", ".", "Source directory where code is found. Defaults to current directory.")
	oneFile   = flag.String("file", "", "Compile a single file")
	clean     = flag.Bool("clean", false, "Set true to recompile all imports")
	debug     = flag.Bool("debug", false, "Enable debug mode")
	linker    = flag.String("linker", "gcc", "Name of linker. Default gcc, alternatives golink, ucrt, msvc")
	arg       = flag.String("arg", "", "Arguments to the compiled program when it is run")
	cacheDir  = flag.String("cache", "./cache", "Cache directory")
)

func LinkRun(workDir string, libPath string, outputName string) error {
	var err error
	// Assemble/link the files
	outputPath := path.Join(workDir, outputName)
	if *link {
		// Assemble library if the linklib argument is given
		if *linklib {
			err = Assemble(libPath)
			if err != nil {
				return err
			}
		}
		// Assemble cached packages (if needed)
		entries, err2 := os.ReadDir(*cacheDir)
		if err2 != nil {
			return err2
		}
		for _, entry := range entries {
			if entry.IsDir() {
				dir := path.Join(*cacheDir, entry.Name())
				err = Assemble(dir)
				if err != nil {
					return err
				}
			}
		}
		err = Assemble(workDir)
		if err == nil {
			err = Link(workDir, libPath, outputName)
		}
	}
	if err == nil && *run {
		// Run the exe file if -run is present and linking is ok
		err = Run(outputPath)
	}
	return err
}

// Assemble wil run the assembler on all *.asm files in the working directory
// And also the syscall.asm from /tools
func Assemble(buildDir string) error {
	entries, err := os.ReadDir(buildDir)
	if err != nil {
		return fmt.Errorf("collecting asm files error,  %s", err.Error())
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.Contains(entry.Name(), ".asm") {
			var args = []string{"-f", "win64"}
			name := filepath.Join(buildDir, strings.TrimSuffix(entry.Name(), ".asm"))
			args = append(args, name+".asm", "-o", name+".obj")
			out, err := exec.Command("c:/doc/compiler/tools/nasm.exe", args...).CombinedOutput()
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
func Link(workDir string, libPath string, outputName string) error {
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
	entries, err = os.ReadDir(libPath)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".obj") {
			args = append(args, path.Join(libPath, entry.Name()))
		}
	}
	outputPath := path.Join(workDir, outputName)
	LinkerName := "c:/doc/compiler/tools/"
	if *linker == "gcc" {
		LinkerName += "MinGW64/bin/gcc.exe"
		if *linklib {
			args = append(args, "-m64", "-lkernel32", "-lmsvcrt", "-o", outputPath)
		} else {
			args = append(args, "-m64", "-lkernel32", "-lmsvcrt", "-o", outputPath)
		}
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
		entries, err = os.ReadDir(libPath)
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".obj") {
				args = append(args, path.Join(libPath, entry.Name()))
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
		// args = append(args, "kernel32.dll", "msvcrt.dll") //  "legacy_stdio_definitions.lib",
	} else {
		fmt.Printf("Must specify either gcc, golink or ucrt")
	}

	// Print link command line to console
	fmt.Printf(LinkerName + " ")
	for _, s := range args {
		fmt.Printf(" %s", s)
	}
	fmt.Printf("\n")

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

// Run will start execution of the exe file made by the link step
func Run(outputName string) error {
	out, err := exec.Command(outputName, *arg).CombinedOutput()
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

func main() {
	// utf8.TestDecode()
	t := time.Now()
	fmt.Printf("%v\n", t)

	flag.Parse()
	// Set logger to not prepend any time/date
	log.SetFlags(0)

	wd, err := os.Getwd()
	fmt.Printf("Starting jkv compiler version %s, in \"%s\"\n", Version, wd)
	if *sourceDir == "" {
		*sourceDir = wd
	}

	exePath, _ := os.Executable()
	exePath = filepath.ToSlash(exePath)
	fmt.Printf("Executable path : %s\n", exePath)

	libPath := path.Dir(exePath)
	libPath = path.Join(libPath, "lib")

	// CompileImports(*buildDir, libPath)

	// Now compile the source files into asm files
	if *oneFile != "" {
		if !strings.Contains(*oneFile, ".") {
			*oneFile += ".jkv"
		}
		err = CompileFile(*buildDir, libPath, *oneFile)
		if err == nil {
			outputName := strings.TrimSuffix(filepath.Base(*oneFile), ".jkv") + ".exe"
			err = LinkRun(*buildDir, libPath, outputName)
		}
	} else if *test {
		n := 0
		n, err = CompileTests(*buildDir, libPath, *sourceDir)
		if err == nil {
			fmt.Printf("------------------------------------------\n")
			fmt.Printf("Run %d files. All tests passed\n", n)
		}
	} else {
		if *clean {
			// Remove all cached files
			err := os.RemoveAll(*cacheDir)
			if err != nil {
				panic("Unable to remove cache directory")
			}
			err = os.Mkdir(*cacheDir, os.ModePerm)
			if err != nil {
				panic("Unable to create cache directory")
			}
		}
		err = CompileDir("main", *buildDir, libPath, *sourceDir)
		if err == nil {
			err = LinkRun(*buildDir, libPath, "main")
		}
	}
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

}
