package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/TopiaNetwork/topia/cmd"
)

var mainCmd = &cobra.Command{Use: "universal"}

func main() {
	mainCmd.AddCommand(cmd.NodeCmd())

	if mainCmd.Execute() != nil {
		os.Exit(1)
	}

}
