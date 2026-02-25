package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const Version string = "v0.0.1"

var (
	workdir    = flag.String("build", "./build", "Path to intermediate files during build")
	run        = flag.Bool("run", false, "Set false to not run after compile")
	help       = flag.Bool("help", false, "Show help")
	version    = flag.Bool("version", false, "Show version and exit")
	outputName = flag.String("o", "", "Output filename")
	inputPath  = flag.String("src", "./", "Source directory")
	oneFile    = flag.String("file", "", "Compile a single file")
)

func CompileDir(inputPath string, outputPath string) error {
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return fmt.Errorf("fatal error %s", err.Error())
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			// CheckFile(s, workdir)
			name := filepath.Join(inputPath, entry.Name())
			fmt.Printf("=== Compiling %s ===\n", name)
			err = CompileFile(name, outputPath)
			if err != nil {
				return err
			}
			fmt.Printf("File %s compiled ok\n", name)
		}
	}
	return err
}

/*
func readerDemo() {
	f, err1 := os.Create("./temp.txt")
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	if err1 != nil {
		return
	}
	r := strings.NewReader("Hello and goodby dear friend")
	n, err := io.Copy(f, r)
	if err != nil {
		return
	}
	fmt.Printf("%d bytes copied\n", n)
}

func stringDemo() {
	s := "HHHæåø"
	b := []byte(s) // 48 48 48 c3 a6 c3 a5 c3 b8
	fmt.Printf("len(s)=%2x  len(b)=%2x\n", len(s), len(b))
	fmt.Printf("%2x %2x %2x %2x %2x %2x %2x %2x %2x\n", b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7], b[8])
	t := strings.TrimRight(s, "øåæ")
	fmt.Println(t)
	os.Exit(0)
}
*/

func main() {
	t := time.Now()
	// demo()
	fmt.Printf("%v\n", t)
	if len(TokenNames) != int(TOK_SIZE)+1 {
		panic("Token names length must be equal to TOK_SIZE")
	}
	flag.Parse()
	log.SetFlags(0)
	slog.SetLogLoggerLevel(4)
	if *version {
		fmt.Println(Version)
	}
	if *help {
		fmt.Printf("Usage:\n")
		fmt.Printf("    -h           show this text\n")
		fmt.Printf("    -v           show version\n")
		fmt.Printf("    -r           will run the program after compilation\n")
		fmt.Printf("    -src <path>  will compile the files in path\n")
		fmt.Printf("Without any parameters it will compile files in the current directory\n")
		return
	}
	var err error
	if *oneFile != "" {
		fmt.Printf("=== Compiling %s ===\n", *oneFile)
		err = CompileFile(*oneFile, *workdir)
	} else {
		err = CompileDir(*inputPath, *workdir)
	}
	if err != nil {
		fmt.Printf("%v\n", err)
	} else if *run {
		fmt.Printf("Compiled ok\n")
		cmd := exec.Command(*outputName)
		err := cmd.Start()
		if err != nil {
			slog.Error("Starting program failed", "error", err.Error())
		}
	}
}
