package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	tpnode "github.com/TopiaNetwork/topia/node"
)

const (
	nodeFuncName = "node"
	nodeCmdDes   = "Operate a node: start."
)

var rootPath string
var endPoint string
var seed string
var role string

var nodeStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the node.",
	Long:  `Starts a node that interacts with the network.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("trailing args detected")
		}
		// Parsing of the command line is done so silence cmd usage
		cmd.SilenceUsage = true

		n := tpnode.NewNode(rootPath, endPoint, seed, role)
		n.Start()
		return nil
	},
}

func startCmd() *cobra.Command {
	flags := nodeStartCmd.PersistentFlags()
	flags.StringVarP(&rootPath, "rootpath", "", "", "the node data root path")
	flags.StringVarP(&endPoint, "endpoint", "", "/ip4/127.0.0.1/tcp/21000", "the node listening endpoint")
	flags.StringVarP(&seed, "seed", "", "universal", "the network peer's seed for generating key")
	flags.StringVarP(&role, "role", "", "executor", "the node role, you can input one of executor,proposer,validator,archiver")
	return nodeStartCmd
}

var nodeCmd = &cobra.Command{
	Use:   nodeFuncName,
	Short: fmt.Sprint(nodeCmdDes),
	Long:  fmt.Sprint(nodeCmdDes),
}

func NodeCmd() *cobra.Command {
	nodeCmd.AddCommand(startCmd())

	return nodeCmd
}
