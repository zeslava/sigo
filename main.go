package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/slavablind91/sigo/detector"
)

func main() {
	flag.Parse()

	filename := flag.Arg(0)
	file, err := os.Open(filename)
	check(err)
	defer file.Close()

	t, err := detector.Detect(file)
	if err != nil {
		fmt.Printf("undefined type: %v\n", err)
	} else {
		fmt.Printf("type: %s\n", t)
	}
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
