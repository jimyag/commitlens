package main

import (
	"embed"

	"github.com/jimyag/commitlens/cmd"
)

//go:embed frontend/dist
var frontendFS embed.FS

func main() {
	cmd.ExecuteWithAssets(frontendFS)
}
