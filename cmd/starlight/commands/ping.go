package commands

import (
	"fmt"
	"github.com/boreq/guinea"
	"github.com/boreq/starlight/local/backend"
	"time"
)

var pingCmd = guinea.Command{
	Arguments: []guinea.Argument{
		{"id", false, "node to ping"},
	},
	Run:              runPing,
	ShortDescription: "pings a node",
	Description: `
Finds an address the node and sends ping messages to measure the network
latency.`,
}

func runPing(c guinea.Context) error {
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
