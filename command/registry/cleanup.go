package registry

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli"

	"kube-helper/config"
	"kube-helper/util"
)

func CmdCleanup(c *cli.Context) error {

	configContainer := config.LoadConfigFromPath(c.String("config"))

	imagePath := configContainer.Cleanup.ImagePath

	branches := util.GetBranches(configContainer.Cleanup.RepoUrl)
	cmd := exec.Command("gcloud", "beta", "container", "images", "list-tags", imagePath, "--format=value(tags)")

	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
		os.Exit(1)
	}

	imagesToDelete := []string{}

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			tags := strings.Split(scanner.Text(), ",")

			// Display all elements.
			cleanup := true
			for _, tag := range tags {
				if strings.HasPrefix(tag, "staging-") == false {
					continue
				}

				if tag == "staging-latest" {
					cleanup = false
					break
				}

				if strings.HasSuffix(tag, "latest") {

					branchName := strings.TrimSuffix(strings.TrimPrefix(tag, "staging-"), "-latest")

					//do not cleanup if branch exists
					if inArray(branches, branchName) {
						cleanup = false
						break
					}

				}
			}

			if cleanup {
				imagesToDelete = append(imagesToDelete, imagePath+":"+tags[0])

			}
		}
	}()

	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
		os.Exit(1)
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
		os.Exit(1)
	}

	for _, image := range imagesToDelete {
		otherCmd := exec.Command("gcloud", "beta", "container", "images", "delete", image, "--resolve-tag-to-digest", "--force-delete-tags")
		stdoutStderr, err := otherCmd.CombinedOutput()
		fmt.Printf("Output:%s\n", stdoutStderr)
		util.CheckError(err)
	}

	return nil
}

func inArray(haystack []string, needle string) bool {
	for _, el := range haystack {
		if el == needle {
			return true
		}
	}

	return false
}
