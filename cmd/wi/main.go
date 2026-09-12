package main

import (
	"context"
	"os"

	pidriver "github.com/regb/workitem/internal/agentdriver/pi"
	"github.com/regb/workitem/internal/cli"
)

func main() {
	ctx := context.Background()
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "internal-driver" {
		if len(args) != 2 || args[1] != "pi" {
			os.Stderr.WriteString("usage: wi internal-driver pi\n")
			os.Exit(2)
		}
		if err := pidriver.Run(ctx); err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}
		return
	}
	os.Exit(cli.Main(ctx, args))
}
