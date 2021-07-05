package main

import (
	"os"

	"tls-secret-injector/cmd"
)

func main() {
	os.Exit(cmd.NewTLSSecretInjector().Run())
}
