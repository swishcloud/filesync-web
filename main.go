package main

import (
	"log"

	"github.com/swishcloud/filesync-web/cmd"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile | log.LUTC)
	cmd.Execute()
}
