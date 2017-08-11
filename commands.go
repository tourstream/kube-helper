package main

import (
	"fmt"
	"os"

	"kube-helper/command/app"
	"kube-helper/command/database"
	"kube-helper/command/registry"

	"github.com/urfave/cli"
)

var GlobalFlags = []cli.Flag{}

var Commands = []cli.Command{
	{
		Name:    "application",
		Aliases: []string{"a"},
		Usage:   "options around a application running in kubernetes",
		Subcommands: []cli.Command{
			{
				Name:      "cleanup",
				Usage:     "remove for closed branches the application from kubernetes",
				Action:    app.CmdCleanUp,
				ArgsUsage: "[branchName]",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
				},
			},
			{
				Name:      "has-namespace",
				Usage:     "",
				Action:    app.CmdHasNamespace,
				ArgsUsage: "[branchName]",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
				},
			},
			{
				Name:      "update",
				Usage:     "",
				Action:    app.CmdUpdate,
				ArgsUsage: "[branchName]",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
					cli.BoolFlag{
						Name: "production, p",
						Usage: "update production",
					},
				},
			},
			{
				Name:      "startup",
				Usage:     "",
				Action:    app.CmdStartUp,
				ArgsUsage: "[branchName]",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
					cli.BoolFlag{
						Name: "production, p",
						Usage: "Start production",
					},
				},
			},
			{
				Name:   "startup-all",
				Usage:  "",
				Action: app.CmdStartUpAll,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
				},
			},
			{
				Name:      "shutdown",
				Usage:     "",
				Action:    app.CmdShutdown,
				ArgsUsage: "[branchName]",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
					cli.BoolFlag{
						Name: "production, p",
						Usage: "update production",
					},
				},
			},
			{
				Name:   "shutdown-all",
				Usage:  "",
				Action: app.CmdShutdownAll,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
				},
			},
		},
	},
	{
		Name:    "registry",
		Aliases: []string{"r"},
		Usage:   "options for the registry in the gcp",
		Subcommands: []cli.Command{
			{
				Name:   "cleanup",
				Usage:  "remove all staging images which do not have a latest tag and where the branch is not there anymore",
				Action: registry.CmdCleanup,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
				},
			},
		},
	},
	{
		Name:    "database",
		Aliases: []string{"d"},
		Usage:   "options for the database in the gcp",
		Subcommands: []cli.Command{
			{
				Name:      "copy",
				Usage:     "",
				Action:    database.CmdCopy,
				ArgsUsage: "[branchName]",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
				},
			},
			{
				Name:   "cleanup",
				Usage:  "remove all databases where the branch is not there anymore",
				Action: database.CmdCleanup,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
				},
			},
		},
	},
}

func CommandNotFound(c *cli.Context, command string) {
	fmt.Fprintf(os.Stderr, "%s: '%s' is not a %s command. See '%s --help'.", c.App.Name, command, c.App.Name, c.App.Name)
	os.Exit(2)
}
