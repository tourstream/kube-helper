package command

import (
	"flag"
	"io/ioutil"

	"github.com/urfave/cli"
)

func RunTestCommand(Action interface{}, arguments []string) {
	testApp := cli.NewApp()
	testApp.Writer = ioutil.Discard
	set := flag.NewFlagSet("test", 0)
	set.Parse(arguments)

	context := cli.NewContext(testApp, set, nil)

	testCommand := cli.Command{
		Name:      arguments[0],
		Usage:     "",
		Action:    Action,
		ArgsUsage: "[branchName]",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config, c",
				Usage: "Load config from `FILE`",
			},
		},
	}

	testCommand.Run(context)
}
