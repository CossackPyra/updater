package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/CossackPyra/updater"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("pyra-poster KEY FILE URL")
		return
	}
	key1 := os.Args[1]
	key2, err := hex.DecodeString(key1)
	if err != nil {
		log.Panicf("Failed decode KET %s %#v", key1, err)
	}
	filename1 := os.Args[2]
	url1 := os.Args[3]

	err = updater.PostFile(url1, filename1, key2)
	if err != nil {
		fmt.Printf("Error: %#v\n", err)
	}

}
