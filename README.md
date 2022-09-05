# BS Open Replay to JSON

[Beat Saber Open Replay format](https://github.com/BeatLeader/BS-Open-Replay) to JSON converter

**Disclaimer**: This is my Go learning project, so expect bugs and ugly code

## Usage

```sh
> bsorutils.exe -h

BSOR utils v0.4.0

Options:

  -h, --help   display help information

Commands:

  help     display help information
  full     Convert replay to JSON
  simple   Simplify replay (acc data only) and convert to JSON
```

### Convert full replay data to JSON

```sh
> bsor2json.exe full -h
  
Convert replay to JSON

Options:

  -h, --help              display help information
  -d, --dir               directory containing bsor files to convert
  -f, --file              bsor file to convert
  -o, --output            output filename (with -f option) or directory (with -d option); defaults to stdout or bsor directory
      --force[=false]     force overwrite
  -p, --pretty[=false]    whether the output JSON should be pretty formatted; conversion time will be much longer and the file will be larger
  -b, --buffered[=true]   whether file read should be buffered; it's faster but increases memory usage
```

For example:

```sh
> bsorutils.exe full -f filename.bsor -o filename.json --force
```

### Dependencies

[go-bsor (BS Open Relay parser)](https://github.com/motzel/go-bsor)