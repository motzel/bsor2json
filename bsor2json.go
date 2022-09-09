package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mitchellh/colorstring"
	"github.com/schollz/progressbar/v3"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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
	Desc: "bsor2json v0.7.0",
	Argv: func() interface{} { return new(rootT) },
	Fn: func(ctx *cli.Context) error {
		return nil
	},
}

type OutputType byte

const (
	RawReplay OutputType = iota
	ReplayEvents
	ReplayEventsWithStats
	ReplayStats
)

func (s OutputType) String() string {
	switch s {
	case RawReplay:
		return "raw"
	case ReplayEvents:
		return "events"
	case ReplayEventsWithStats:
		return "events_and_stats"
	case ReplayStats:
		return "stats"
	default:
		return "unknown"
	}
}

func loadAndDecodeReplay(fileName string, buffered bool) (*bsor.Replay, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("can not open replay: %v", err)
	}

	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("can not get replay size: %v", err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("%v is a directory", fileName)
	}

	var reader io.Reader
	if buffered {
		buf := bytes.NewBuffer(make([]byte, 0, fileInfo.Size()))
		_, err = buf.ReadFrom(file)
		if err != nil {
			return nil, fmt.Errorf("can not read replay: %v", err)
		}
		reader = io.Reader(buf)
	} else {
		reader = file
	}

	var replay *bsor.Replay
	if replay, err = bsor.Read(reader); err != nil {
		return nil, fmt.Errorf("replay decode error: %v", err)
	}

	return replay, nil
}

func convertReplay(fileName string, outputType OutputType, output string, buffered bool, pretty bool, force bool) error {
	replay, err := loadAndDecodeReplay(fileName, buffered)
	if err != nil {
		return err
	}

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
	case RawReplay:
		out, err = toJson(replay, pretty)
	case ReplayEvents:
		replayEvents := bsor.NewReplayEvents(replay)
		out, err = toJson(replayEvents, pretty)
	case ReplayEventsWithStats:
		replayEvents := bsor.NewReplayEvents(replay)
		replayEventsWithStats := bsor.NewReplayEventsWithStats(replayEvents)
		out, err = toJson(replayEventsWithStats, pretty)
	case ReplayStats:
		replayEvents := bsor.NewReplayEvents(replay)
		replayStats := bsor.NewReplayStats(replayEvents)
		out, err = toJson(replayStats, pretty)
	default:
		err = fmt.Errorf("unknown output type")
	}

	if err != nil {
		return fmt.Errorf("JSON marshaling error: %v", err)
	}

	fmt.Fprintln(writer, string(out))

	writer.Flush()

	return nil
}

type Job struct {
	Dir      string
	Filename string
	Error    *error
}

func convert(argv *ReplayT, outputType OutputType) error {
	if len(argv.Dir) > 0 {
		files, err := ioutil.ReadDir(argv.Dir)
		if err != nil {
			return err
		}

		parallel := argv.Parallel
		if parallel <= 0 || parallel > runtime.NumCPU() {
			parallel = runtime.NumCPU()
		}

		jobs := make(chan *Job, parallel)
		results := make(chan *Job, parallel)
		done := make(chan bool)

		bsorFiles := make([]Job, 0, len(files))

		outputDirectory := argv.Output
		if len(outputDirectory) == 0 {
			outputDirectory = argv.Dir
		}

		for _, file := range files {
			inputFilename := filepath.Join(argv.Dir, file.Name())

			if file.IsDir() || strings.ToLower(filepath.Ext(inputFilename)) != ".bsor" {
				continue
			}

			bsorFiles = append(bsorFiles, Job{Filename: file.Name(), Dir: argv.Dir})
		}

		barDescription := fmt.Sprintf("[green]Processing replays [yellow](parallel: %v)[reset]...", parallel)
		bar := progressbar.NewOptions(len(bsorFiles),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionSetDescription(barDescription),
			progressbar.OptionShowCount(),
			progressbar.OptionSetElapsedTime(true),
			progressbar.OptionSetPredictTime(false),
			progressbar.OptionShowIts(),
			progressbar.OptionSetItsString("replays"),
		)

		// jobs producer
		go func() {
			for i := range bsorFiles {
				jobs <- &bsorFiles[i]
			}
			close(jobs)
		}()

		// results receiver
		go func(done chan bool) {
			for range results {
				bar.Add(1)
			}
			done <- true
			//bar.Finish()
		}(done)

		// create worker pool
		var wg sync.WaitGroup
		for i := 0; i < parallel; i++ {
			wg.Add(1)

			// worker
			go func(wg *sync.WaitGroup) {
				defer wg.Done()

				for job := range jobs {
					inputFilename := filepath.Join(job.Dir, job.Filename)
					outputFilename := filepath.Join(outputDirectory, fileNameWithoutExt(filepath.Base(job.Filename))+"."+outputType.String()+".json")

					if err = convertReplay(inputFilename, outputType, outputFilename, argv.Buffered, argv.Pretty, argv.Force); err != nil {
						jobErr := err
						job.Error = &jobErr
					}

					results <- job
				}
			}(&wg)
		}
		wg.Wait()
		close(results)

		<-done

		total := 0
		ok := 0
		failed := 0
		failedJobs := make([]error, 0, len(bsorFiles))
		for _, job := range bsorFiles {
			if job.Error != nil {
				failed++

				if argv.DisplayFailed {
					failedJobs = append(failedJobs, *job.Error)
				}
			} else {
				ok++
			}

			total++
		}

		log.Printf(colorstring.Color("\nReplays processed. [blue]Total:[reset] %v, [green]OK:[reset] %v, [red]Failed:[reset] %v"), total, ok, failed)

		if argv.DisplayFailed && len(failedJobs) > 0 {
			for _, err := range failedJobs {
				log.Printf(colorstring.Color("[red]%s[reset]"), err)
			}
		}

		return nil
	} else {
		return convertReplay(argv.File, outputType, argv.Output, argv.Buffered, argv.Pretty, argv.Force)
	}
}

type ReplayT struct {
	cli.Helper
	Dir           string `cli:"d,dir" usage:"directory containing bsor files to convert" dft:""`
	File          string `cli:"f,file" usage:"bsor file to convert"`
	Output        string `cli:"o,output" usage:"output filename (with -f option) or directory (with -d option); defaults to stdout or bsor directory" dft:""`
	Force         bool   `cli:"force" usage:"force overwrite" dft:"false"`
	Pretty        bool   `cli:"p,pretty" usage:"whether the output JSON should be pretty formatted; conversion time will be much longer and the file will be larger" dft:"false"`
	Buffered      bool   `cli:"b,buffered" usage:"whether file read should be buffered; it's faster but increases memory usage" dft:"true"`
	Parallel      int    `cli:"parallel" usage:"parallel processing of multiple replays at once; equal to the number of cpu cores if zero or not specified " dft:"0"`
	DisplayFailed bool   `cli:"display-failed" usage:"display failed replays when using the -d option" dft:"true"`
}

func (argv *ReplayT) Validate(ctx *cli.Context) error {
	if len(argv.File) == 0 && len(argv.Dir) == 0 {
		return fmt.Errorf("%s\nfile or directory is required; add -h flag for help", ctx.Command().Desc)
	}
	return nil
}

var replay = &cli.Command{
	Name: "raw",
	Desc: "Convert raw replay data to JSON",
	Argv: func() interface{} { return new(ReplayT) },
	Fn: func(ctx *cli.Context) error {
		argv := ctx.Argv().(*ReplayT)

		return convert(argv, RawReplay)
	},
}

type ReplayEventsT struct {
	ReplayT
	WithStats bool `cli:"s,with-stats" usage:"whether to add stats" dft:"true"`
}

func (argv *ReplayEventsT) Validate(ctx *cli.Context) error {
	if len(argv.File) == 0 && len(argv.Dir) == 0 {
		return fmt.Errorf("%s\nfile or directory is required; add -h flag for help", ctx.Command().Desc)
	}
	return nil
}

var events = &cli.Command{
	Name: "events",
	Desc: "Simplify replay (notes/walls/pauses events only) and export to JSON",
	Argv: func() interface{} { return new(ReplayEventsT) },
	Fn: func(ctx *cli.Context) error {
		argv := ctx.Argv().(*ReplayEventsT)

		outputType := ReplayEvents
		if argv.WithStats {
			outputType = ReplayEventsWithStats
		}

		return convert((*ReplayT)(unsafe.Pointer(argv)), outputType)
	},
}

type ReplayStatsT struct {
	ReplayT
}

var stats = &cli.Command{
	Name: "stats",
	Desc: "Calculate stats and export to JSON",
	Argv: func() interface{} { return new(ReplayStatsT) },
	Fn: func(ctx *cli.Context) error {
		argv := ctx.Argv().(*ReplayStatsT)

		return convert((*ReplayT)(unsafe.Pointer(argv)), ReplayStats)
	},
}

func main() {
	log.SetFlags(0)

	start := time.Now()

	if err := cli.Root(root, cli.Tree(help), cli.Tree(replay), cli.Tree(events), cli.Tree(stats)).Run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	elapsed := time.Since(start)

	log.SetOutput(os.Stderr)
	log.Printf(colorstring.Color("\nOperation took [green]%s[reset]"), elapsed)
}
