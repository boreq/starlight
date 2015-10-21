package commands

import (
	"fmt"
	"github.com/boreq/lainnet/cli"
	"github.com/boreq/lainnet/local/backend"
	"time"
)

var pingCmd = cli.Command{
	Arguments: []cli.Argument{
		{"id", false, "node to ping"},
	},
	Run:              runPing,
	ShortDescription: "pings a node",
	Description: `
Finds an address the node and sends ping messages to measure the network
latency.`,
}

func runPing(c cli.Context) error {
	client, err := GetClient()
	if err != nil {
		return err
	}

	for {
		args := &backend.PingArgs{c.Arguments[0]}
		latency := new(float64)
		err := client.Call("Backend.Ping", args, &latency)
		if err != nil {
			return err
		}
		fmt.Printf("%fms\n", *latency)
		<-time.After(1 * time.Second)
	}

	return nil
}
