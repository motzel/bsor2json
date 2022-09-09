# BS Open Replay to JSON

[Beat Saber Open Replay format](https://github.com/BeatLeader/BS-Open-Replay) to JSON converter

**Disclaimer**: This is my Go learning project, so expect bugs and ugly code

## Usage

```
> bsor2json.exe -h

bsor2json v0.7.2

Options:

  -h, --help   display help information

Commands:

  help     display help information
  raw      Convert raw replay data to JSON
  events   Simplify replay (notes/walls/pauses events only) and export to JSON
  stats    Calculate stats and export to JSON
```

### Convert raw replay data to JSON

```
> bsor2json.exe raw -h
  
Convert raw replay data to JSON

Options:

  -h, --help                    display help information
  -d, --dir                     directory containing bsor files to convert
  -f, --file                    bsor file to convert
  -o, --output                  output filename (with -f option) or directory (with -d option); defaults to stdout or bsor directory
      --force[=false]           force overwrite
  -p, --pretty[=false]          whether the output JSON should be pretty formatted; conversion time will be much longer and the file will be larger
  -b, --buffered[=true]         whether file read should be buffered; it's faster but increases memory usage
      --parallel[=0]            parallel processing of multiple replays at once; equal to the number of cpu cores if zero or not specified 
      --display-failed[=true]   display failed replays when using the -d option
```

For example:

```sh
> bsor2json.exe raw -f filename.bsor -o filename.json --force
```

### Simplify replay and export to JSON 

```
> bsor2json.exe events -h
Simplify replay (notes/walls/pauses events only) and export to JSON

Options:

  -h, --help                    display help information
  -d, --dir                     directory containing bsor files to convert
  -f, --file                    bsor file to convert
  -o, --output                  output filename (with -f option) or directory (with -d option); defaults to stdout or bsor directory
      --force[=false]           force overwrite
  -p, --pretty[=false]          whether the output JSON should be pretty formatted; conversion time will be much longer and the file will be larger
  -b, --buffered[=true]         whether file read should be buffered; it's faster but increases memory usage
      --parallel[=0]            parallel processing of multiple replays at once; equal to the number of cpu cores if zero or not specified
      --display-failed[=true]   display failed replays when using the -d option
  -s, --with-stats[=true]       whether to add stats
```

For example:

```sh
> bsor2json.exe events -d Replays --with-stats
```

### Calculate stats and export to JSON

```
> bsor2json.exe stats -h
Calculate stats and export to JSON

Options:

  -h, --help                    display help information
  -d, --dir                     directory containing bsor files to convert
  -f, --file                    bsor file to convert
  -o, --output                  output filename (with -f option) or directory (with -d option); defaults to stdout or bsor directory
      --force[=false]           force overwrite
  -p, --pretty[=false]          whether the output JSON should be pretty formatted; conversion time will be much longer and the file will be larger
  -b, --buffered[=true]         whether file read should be buffered; it's faster but increases memory usage
      --parallel[=0]            parallel processing of multiple replays at once; equal to the number of cpu cores if zero or not specified 
      --display-failed[=true]   display failed replays when using the -d option
```

For example:

```sh
> bsor2json.exe stats -f filename.bsor --pretty
```

## Build

### Dev

Install [cosmtrek/air](https://github.com/cosmtrek/air), customize the ``args_bin`` in ``.air.toml`` and then:

```sh
go install
air.exe
```

### Release

```sh
go build -ldflags "-s -w"
```

### Dependencies

[go-bsor (BS Open Replay parser)](https://github.com/motzel/go-bsor)
