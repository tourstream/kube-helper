package app

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/urfave/cli"
	"kube-helper/command/database"
	"kube-helper/loader"
)

type Digest struct {
	Digest string
}

var fileSystem = afero.NewOsFs()
var databaseCopy = database.CopyDatabaseByBranchName

func CmdStartUpAll(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	clientSet, err := serviceBuilder.GetClientSet(configContainer)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = cp(".env_dist", ".env")
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = fileSystem.Remove(".env")

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	branches, err := branchLoader.LoadBranches(configContainer.Bitbucket)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	for _, branch := range branches {
		tag := "staging-" + branch + "-latest"
		if branch == "master" {
			tag = "staging-latest"
		}

		hasTag, err := imagesService.HasTag(configContainer.Cleanup, tag)

		if err != nil {
			fmt.Fprintln(writer, err)
		}

		if hasTag == false {
			continue
		}
		dat, err := afero.ReadFile(fileSystem, ".env_dist")
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		stringDat := string(dat)

		if (loader.Database{}) != configContainer.Database {
			databaseName := database.GetDatabaseName(configContainer.Database, branch)
			stringDat += "\nDATABASE_NAME=" + databaseName + "\n"
		}

		err = afero.WriteFile(fileSystem, ".env", []byte(stringDat), 0644)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}

		if (loader.Database{}) != configContainer.Database {
			err = databaseCopy(branch, configContainer)

			if err != nil {
				fmt.Fprintln(writer, err)
			}
		}

		appService, err := serviceBuilder.GetApplicationService(clientSet, getNamespace(branch, false), configContainer)

		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}

		err = appService.Apply()

		if err != nil {
			fmt.Fprintln(writer, err)
		}
	}

	return nil
}

func cp(dst, src string) error {
	s, err := fileSystem.Open(src)
	if err != nil {
		return err
	}
	// no need to check errors on read only file, we already got everything
	// we need from the filesystem, so nothing can go wrong now.
	defer s.Close()
	d, err := fileSystem.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(d, s); err != nil {
		defer d.Close()
		return err
	}
	return d.Close()
}
