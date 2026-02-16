package main

import (
	"go.uber.org/fx"

	"nurmed/cmd/nurmed/handlers"
	"nurmed/cmd/nurmed/router"
	"nurmed/internal"
	"nurmed/pkg"
)

func main() {
	fx.New(
		handlers.Module,
		router.Module,
		internal.Module,
		pkg.Module,
	).Run()
}
