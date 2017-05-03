package util

import (
  "bufio"
  "fmt"
  "os"
  "os/exec"
  "strings"
)


func GetBranches(repoURL string) []string {
  cmd := exec.Command("git", "ls-remote", "--heads", repoURL)
  cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
		os.Exit(1)
	}

  var branches []string

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
      words := strings.Fields(scanner.Text())
      branches = append(branches, strings.Replace(words[1], "refs/heads/", "", 1))
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

  return branches
}
