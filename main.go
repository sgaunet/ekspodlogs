package main

import "github.com/sgaunet/ekspodlogs/cmd"

//go:generate go tool github.com/sqlc-dev/sqlc/cmd/sqlc generate -f ./sqlc.yaml

func main() {
	cmd.Execute()
}
