package main

import (
	"fmt"
	"os"

	"kube-helper/command/app"
	"kube-helper/command/database"
	"kube-helper/command/registry"

	"github.com/urfave/cli"
	"kube-helper/command/services"
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
				Name:      "get-domain",
				Usage:     "",
				Action:    app.CmdGetDomain,
				ArgsUsage: "[branchName]",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
				},
			},
			{
				Name:      "apply",
				Usage:     "",
				Action:    app.CmdApply,
				ArgsUsage: "[branchName]",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
					cli.BoolFlag{
						Name:  "production, p",
						Usage: "update production",
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
						Name:  "production, p",
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
		Name:    "services",
		Aliases: []string{"s"},
		Usage:   "options for the services in the cluster",
		Subcommands: []cli.Command{
			{
				Name:   "get-ip",
				Usage:  "get the cluster ip of a service in a namespace",
				Action: services.CmdGetIp,
				ArgsUsage: "[namespace, name]",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
				},
			},
			{
				Name:   "add-ip",
				Usage:  "add a named ip to a load balancer",
				Action: services.CmdAddRuleToLoadBalancer,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
					cli.StringFlag{
						Name:  "address, a",
						Usage: "named address to add to context loadbalancer",
					},
				},
			},
			{
				Name:   "remove-ip",
				Usage:  "remove a named ip from a load balancer",
				Action: services.CmdRemoveRuleFromLoadBalancer,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
					cli.StringFlag{
						Name:  "address, a",
						Usage: "named address to remove from context loadbalancer",
					},
				},
			},
			{
				Name:   "add-cert",
				Usage:  "add a ssl-certificate to a load balancer",
				Action: services.CmdAddCertificate,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
					cli.StringFlag{
						Name:  "ssl-certificate, s",
						Usage: "ssl-certificate to add to context loadbalancer",
					},
				},
			},
			{
				Name:   "remove-cert",
				Usage:  "remove a ssl-certificate from a load balancer",
				Action: services.CmdRemoveCertificate,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config, c",
						Usage: "Load config from `FILE`",
					},
					cli.StringFlag{
						Name:  "ssl-certificate, s",
						Usage: "ssl-certificate to remove from context loadbalancer",
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
