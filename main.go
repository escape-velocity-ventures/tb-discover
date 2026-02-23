package main

import "github.com/tinkerbelle-io/tb-manage/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
