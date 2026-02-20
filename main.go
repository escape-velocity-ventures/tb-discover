package main

import "github.com/tinkerbelle-io/tb-discover/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
