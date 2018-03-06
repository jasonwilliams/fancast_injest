package main

import (
	"os"

	"bitbucket.org/jayflux/mypodcasts_injest/injest"
)

func main() {
	argsWithoutProg := os.Args[1:]
	injest.Injest(argsWithoutProg[0])

}
