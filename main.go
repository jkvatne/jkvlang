package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os/exec"
)

const Version string = "v0.0.1"

var (
	workdir    = flag.String("build", "./build", "Path to intermediate files during build")
	run        = flag.Bool("run", true, "Set false to not run after compile")
	help       = flag.Bool("help", false, "Show help")
	version    = flag.Bool("version", false, "Show version and exit")
	outputName = flag.String("o", "", "Output filename")
	inputPath  = flag.String("src", "./", "Source directory")
)

func main() {
	flag.Parse()
	log.SetFlags(0)
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
	err := Compile(*workdir, *inputPath, *outputName)
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
