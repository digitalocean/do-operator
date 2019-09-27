package main

import (
	"fmt"

	"github.com/digitalocean/do-operator/version"
)

func main() {
	ver := version.Version
	if ver[0] != 'v' {
		ver = "v" + ver
	}
	fmt.Println(ver)
}
