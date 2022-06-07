package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/motzel/go-bsor/bsor"
)

func main() {
	log.SetPrefix("[bsor2json] ")
	log.SetFlags(0)

	argsWithoutProg := os.Args[1:]

	if len(argsWithoutProg) != 1 {
		fmt.Print("Usage: bsor2json filename.bsor > filename.json\n\n")
		return
	}

	path := argsWithoutProg[0]

	file, err := os.Open(path)
	if err != nil {
		log.Fatal("Can not open replay: ", err)
	}

	defer file.Close()

	var replay *bsor.Bsor
	if replay, err = bsor.Read(file); err != nil {
		log.Fatal("Replay decode: ", err)
	}

	json, err := json.Marshal(replay)
	if err != nil {
		log.Fatal("JSON marshalling error:", err)
	}

	fmt.Println(string(json))
}
