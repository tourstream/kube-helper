package app

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/urfave/cli"
	"kube-helper/command/database"
	"kube-helper/util"
)

type Digest struct {
	Digest string
}

func CmdStartUpAll(c *cli.Context) error {

	err := cp(".env", ".env_dist")
	if err != nil {
		return err
	}

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	clientSet, err := serviceBuilder.GetClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	if err != nil {
		return err
	}

	err = os.Remove(".env")

	if err != nil {
		return err
	}

	branches, err := branchLoader.LoadBranches(configContainer.Bitbucket)

	if err != nil {
		return err
	}

	for _, branch := range branches {
		tag := "staging-" + branch + "-latest"
		if branch == "master" {
			tag = "staging-latest"
		}

		hasTag, err := imagesService.HasTag(configContainer.Cleanup, tag)

		if err != nil {
			util.Dump(err)
		}

		if hasTag == false {
			continue
		}

		dat, err := ioutil.ReadFile(".env_dist")
		if err != nil {
			return err
		}

		databaseName := database.GetDatabaseName(configContainer.Database, branch)
		stringDat := string(dat)

		stringDat += "\nDATABASE_NAME=" + databaseName + "\n"

		err = ioutil.WriteFile(".env", []byte(stringDat), 0644)
		if err != nil {
			return err
		}

		err = database.CopyDatabaseByBranchName(branch, configContainer)

		if err != nil {
			util.Dump(err)
		}

		err = createApplicationByNamespace(clientSet, getNamespace(branch), configContainer)
		if err != nil {
			util.Dump(err)
		}
	}

	return nil
}

func cp(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	// no need to check errors on read only file, we already got everything
	// we need from the filesystem, so nothing can go wrong now.
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}
