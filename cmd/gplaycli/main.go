package main

import (
	"github.com/Elbandi/go-googleplay/cmd/gplaycli/cmd"
	log "github.com/sirupsen/logrus"
)

func main()  {

	log.SetFormatter(&log.TextFormatter{ForceColors: true})

	cmd.Execute()
}


