package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/vieitesss/k8s-d2/cmd"
)

var version = "dev"

func main() {
	if err := cmd.Execute(version); err != nil {
		log.SetReportTimestamp(false)
		log.Error(err.Error())
		os.Exit(1)
	}
}
