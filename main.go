package main

import "os"

func main() {
	c := &cli{outStream: os.Stdout, errStream: os.Stderr}
	os.Exit(c.run(os.Args))
}
