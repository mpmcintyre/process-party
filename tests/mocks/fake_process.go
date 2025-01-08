package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"strconv"
	"time"
)

func main() {
	args := os.Args[1:]

	switch args[0] {
	case "sleep":
		i, err := strconv.Atoi(args[1])
		fmt.Printf("Sleeping for %d second(s)\n", i)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Duration(i) * time.Second)
	case "mkdir":
		err := os.MkdirAll(args[1], 0755)
		if err != nil {
			log.Fatal(err)
		}
	case "touch":
		err := os.WriteFile(args[1], []byte{}, fs.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	case "fail":
		fmt.Printf("failing task on purpouse\n")
		os.Exit(1)
	}

	fmt.Printf("%s executed successfully\n", args[0])
}
