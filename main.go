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
	noCode     = flag.Bool("no", false, "Do not generate code")
	oneFile    = flag.String("file", "", "Compile a single file")
)

func CompileDir(inputPath string, outputPath string) error {
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return fmt.Errorf("Fatal error " + err.Error())
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			// CheckFile(s, workdir)
			name := filepath.Join(inputPath, entry.Name())
			err = CompileFile(name, outputPath)
		}
	}
	return err
}

func main() {
	t := time.Now()
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
		err = CompileFile(*oneFile, *workdir)
	} else {
		err = CompileDir(*inputPath, *workdir)
	}
	if err != nil {
		fmt.Println(err.Error())
	} else if *run {
		cmd := exec.Command(*outputName)
		err := cmd.Start()
		if err != nil {
			slog.Error("Starting program failed", "error", err.Error())
		}
	}
}
