package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/mkideal/cli"
	"github.com/motzel/go-bsor/bsor"
)

func fileNameWithoutExt(fileName string) string {
	return strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
}

func toJson(data interface{}, pretty bool) ([]byte, error) {
	var out []byte
	var err error

	switch pretty {
	case true:
		out, err = json.MarshalIndent(data, "", "  ")
	case false:
		out, err = json.Marshal(data)
	}

	return out, err
}

var help = cli.HelpCommand("display help information")

type rootT struct {
	cli.Helper
}

func (argv *rootT) Validate(ctx *cli.Context) error {
	return fmt.Errorf("%s\nno command; add -h flag for help", root.Desc)
}

var root = &cli.Command{
	Name: "root",
	Desc: "BSOR utils v0.4.0",
	Argv: func() interface{} { return new(rootT) },
	Fn: func(ctx *cli.Context) error {
		return nil
	},
}

type OutputType byte

const (
	FullReplay OutputType = iota
	SimpleReplay
	Stats
)

func convertReplay(fileName string, outputType OutputType, output string, buffered bool, pretty bool, force bool) error {
	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("can not open replay: %v", err)
	}

	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("can not get replay size: %v", err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("%v is a directory", fileName)
	}

	var reader io.Reader
	if buffered {
		buf := bytes.NewBuffer(make([]byte, 0, fileInfo.Size()))
		_, err = buf.ReadFrom(file)
		if err != nil {
			return fmt.Errorf("can not read replay: %v", err)
		}
		reader = io.Reader(buf)
	} else {
		reader = file
	}

	var replay *bsor.Bsor
	if replay, err = bsor.Read(reader); err != nil {
		return fmt.Errorf("replay decode error: %v", err)
	}

	simplifiedReplay := bsor.NewBsorSimple(replay)

	var writer *bufio.Writer

	if len(output) > 0 {
		var outFileName = output

		if _, err := os.Stat(outFileName); errors.Is(err, os.ErrNotExist) {
			// file does not exist
		} else {
			// file exists

			if !force {
				return fmt.Errorf("file already exists: %v", outFileName)
			}
		}

		var fileW *os.File

		fileW, err = os.Create(outFileName)
		if err != nil {
			return fmt.Errorf("can not create output file: %v", err)
		}

		defer fileW.Close()

		writer = bufio.NewWriter(fileW)
	} else {
		writer = bufio.NewWriter(os.Stdout)
	}

	var out []byte
	switch outputType {
	case FullReplay:
		out, err = toJson(replay, pretty)
	case SimpleReplay:
		out, err = toJson(simplifiedReplay, pretty)
	case Stats:
		err = fmt.Errorf("stats output not implemented yet")
	}
	if err != nil {
		return fmt.Errorf("JSON marshaling error: %v", err)
	}

	fmt.Fprintln(writer, string(out))

	writer.Flush()

	return nil
}

func convert(argv *ReplayT, outputType OutputType) error {
	if len(argv.File) == 0 {
		return fmt.Errorf("Directory option is not implemented yet! Please use the -f option")
	}

	var outputFilename = argv.Output
	if len(argv.Dir) > 0 {
		outputFilename = filepath.Join(argv.Output, fileNameWithoutExt(filepath.Base(argv.File))+".json")
	}

	return convertReplay(argv.File, outputType, outputFilename, argv.Buffered, argv.Pretty, argv.Force)
}

type ReplayT struct {
	cli.Helper
	Dir      string `cli:"d,dir" usage:"directory containing bsor files to convert" dft:""`
	File     string `cli:"f,file" usage:"bsor file to convert"`
	Output   string `cli:"o,output" usage:"output filename (with -f option) or directory (with -d option); defaults to stdout or bsor directory" dft:""`
	Force    bool   `cli:"force" usage:"force overwrite" dft:"false"`
	Pretty   bool   `cli:"p,pretty" usage:"whether the output JSON should be pretty formatted; conversion time will be much longer and the file will be larger" dft:"false"`
	Buffered bool   `cli:"b,buffered" usage:"whether file read should be buffered; it's faster but increases memory usage" dft:"true"`
}

func (argv *ReplayT) Validate(ctx *cli.Context) error {
	if len(argv.File) == 0 && len(argv.Dir) == 0 {
		return fmt.Errorf("%s\nfile or directory is required; add -h flag for help", ctx.Command().Desc)
	}
	return nil
}

var replay = &cli.Command{
	Name: "full",
	Desc: "Convert replay to JSON",
	Argv: func() interface{} { return new(ReplayT) },
	Fn: func(ctx *cli.Context) error {
		argv := ctx.Argv().(*ReplayT)

		return convert(argv, FullReplay)
	},
}

type SimpleT struct {
	ReplayT
}

func (argv *SimpleT) Validate(ctx *cli.Context) error {
	if len(argv.File) == 0 && len(argv.Dir) == 0 {
		return fmt.Errorf("%s\nfile or directory is required; add -h flag for help", ctx.Command().Desc)
	}
	return nil
}

var simple = &cli.Command{
	Name: "simple",
	Desc: "Simplify replay (acc data only) and convert to JSON",
	Argv: func() interface{} { return new(SimpleT) },
	Fn: func(ctx *cli.Context) error {
		argv := ctx.Argv().(*SimpleT)

		return convert((*ReplayT)(unsafe.Pointer(argv)), SimpleReplay)
	},
}

func main() {
	log.SetFlags(0)

	start := time.Now()

	if err := cli.Root(root, cli.Tree(help), cli.Tree(replay), cli.Tree(simple)).Run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	elapsed := time.Since(start)

	log.SetOutput(os.Stderr)
	log.Printf("\nOperation took %s", elapsed)
}
