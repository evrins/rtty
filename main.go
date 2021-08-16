package main

import (
	"github.com/skanehira/rtty/cmd"
	"log"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
}

func main() {
	cmd.Execute()
}
