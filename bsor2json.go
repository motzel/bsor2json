package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/mkideal/cli"
	"github.com/motzel/go-bsor/bsor"
)

var help = cli.HelpCommand("display help information")

// root command
type rootT struct {
	cli.Helper
	// Dir    string `cli:"d,dir" usage:"directory containing bsor files to convert" dft:".\\BeatLeader\\Replays"`
	File     string `cli:"f,file" usage:"bsor file to convert"`
	Pretty   bool   `cli:"p,pretty" usage:"whether the output JSON should be pretty formatted; conversion time will be much longer and the file will be larger" dft:"false"`
	Buffered bool   `cli:"b,buffered" usage:"whether file read should be buffered; it's faster but increases memory usage" dft:"true"`
}

func (argv *rootT) Validate(ctx *cli.Context) error {
	if len(argv.File) == 0 {
		return fmt.Errorf("file is required; add -h flag for help")
	}
	return nil
}

var root = &cli.Command{
	Name: "bsor2json v0.4.0",
	Desc: "Convert bsor file to json",
	Argv: func() interface{} { return new(rootT) },
	Fn: func(ctx *cli.Context) error {
		argv := ctx.Argv().(*rootT)

		file, err := os.Open(argv.File)
		if err != nil {
			log.Fatal("Can not open replay: ", err)
		}

		defer file.Close()

		var reader io.Reader
		if argv.Buffered {
			fi, err := file.Stat()
			if err != nil {
				log.Fatal("Can not get replay size: ", err)
			}

			buf := bytes.NewBuffer(make([]byte, 0, fi.Size()))
			_, err = buf.ReadFrom(file)
			if err != nil {
				log.Fatal("Can not read replay: ", err)
			}
			reader = io.Reader(buf)
		} else {
			reader = file
		}

		var replay *bsor.Bsor
		if replay, err = bsor.Read(reader); err != nil {
			log.Fatal("Replay decode: ", err)
		}

		var out []byte
		switch argv.Pretty {
		case true:
			out, err = json.MarshalIndent(replay, "", "  ")
		case false:
			out, err = json.Marshal(replay)
		}
		if err != nil {
			log.Fatal("JSON marshalling error:", err)
		}

		fmt.Println(string(out))

		return nil
	},
}

func main() {
	log.SetFlags(0)

	if err := cli.Root(root, cli.Tree(help)).Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
