package app

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/urfave/cli"

	"gopkg.in/yaml.v2"

	"kube-helper/command/database"
	"kube-helper/config"
	"kube-helper/util"
)

type Digest struct {
	Digest string
}

func CmdStartUpAll(c *cli.Context) error {

	err := cp(".env", ".env_dist")
	util.CheckError(err)

	configContainer := config.LoadConfigFromPath(c.String("config"))
	err = os.Remove(".env")
	util.CheckError(err)

	createUniveralDecoder()
	createContainerService()
	createClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)
	branches := util.GetBranches(configContainer.Cleanup.RepoUrl)
	for _, branch := range branches {
		tag := "staging-" + branch + "-latest"
		if branch == "master" {
			tag = "staging-latest"
		}

		otherCmd := exec.Command("gcloud", "beta", "container", "images", "list-tags", configContainer.Cleanup.ImagePath, "--filter=tags="+tag, "--format=yaml")
		stdoutStderr, err := otherCmd.CombinedOutput()
		if err != nil {
			log.Print(err)
			continue
		}

		imageDigest := Digest{}
		err = yaml.Unmarshal(stdoutStderr, &imageDigest)
		if err != nil {
			log.Print(err)
			continue
		}
		if imageDigest.Digest == "" {
			continue
		}
		util.Dump(imageDigest)

		dat, err := ioutil.ReadFile(".env_dist")
		util.CheckError(err)
		databaseName := database.GetDatabaseName(configContainer, branch)
		stringDat := string(dat)

		stringDat += "\nDATABASE_NAME=" + databaseName + "\n"

		err = ioutil.WriteFile(".env", []byte(stringDat), 0644)
		util.CheckError(err)
		database.CopyDatabaseByBranchName(branch, configContainer)
		err = createApplicationByNamespace(getNamespace(branch), configContainer)
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
