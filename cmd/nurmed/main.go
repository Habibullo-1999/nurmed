package main

import (
	"go.uber.org/fx"

	"nurmed/cmd/nurmed/router"
	"nurmed/pkg"
)

func main() {
	fx.New(
		//handlers.Module,
		router.Module,
		//internal.Module,
		pkg.Module,
	).Run()
}
