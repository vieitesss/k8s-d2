package main

import "github.com/vieitesss/k8s-d2/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
