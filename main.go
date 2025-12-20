package main

import (
	_ "github.com/lib/pq"
	"redifu-example/cmd"
)

func main() {
	cmd.StartAPI()
}
