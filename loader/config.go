package loader

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

type ConfigLoaderInterface interface {
	LoadConfigFromPath(filepath string) (Config, error)
}

type Cleanup struct {
	ImagePath string `yaml:"image_path"`
}

type DNSConfig struct {
	ProjectID    string   `yaml:"project_id"`
	ManagedZone  string   `yaml:"managed_zone"`
	DomainSuffix string   `yaml:"domain_suffix"`
	CNameSuffix  []string `yaml:"cname_suffix"`
}

type Bitbucket struct {
	ClientID       string `yaml:"client_id"`
	ClientSecret   string `yaml:"client_secret"`
	Username       string `yaml:"username"`
	RepositoryName string `yaml:"repository_name"`
	ApiUrl         string `yaml:"api_url"`
	TokenUrl       string `yaml:"token_url"`
}

type Database struct {
	Instance             string
	BaseName             string `yaml:"base_name"`
	PrefixBranchDatabase string `yaml:"prefix_branch_database"`
	Bucket               string
}

type Config struct {
	KubernetesConfigFilepath string `yaml:"kubernetes_config_filepath"`
	ProjectID                string `yaml:"project_id"`
	ClusterID                string `yaml:"cluster_id"`
	Zone                     string
	Bitbucket                Bitbucket
	Cleanup                  Cleanup
	DNS                      DNSConfig `yaml:"dns"`
	Database                 Database
}

var fileSystem = afero.NewOsFs()

var envReader = godotenv.Read

type callable func([]string) error

func (c *Config) LoadConfigFromPath(filepath string) (Config, error) {
	config := Config{}

	err := replaceVariablesInFile(fileSystem, filepath, func(splitLines []string) error {
		return yaml.Unmarshal([]byte(strings.Join(splitLines, "\n")), &config)
	})

	return config, err
}

func replaceVariablesInFile(fileSystem afero.Fs, path string, functionCall callable) error {
	file, err := fileSystem.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var myEnv map[string]string
	myEnv, err = envReader()
	if err != nil {
		return err
	}

	splitLines := []string{}

	scanner := bufio.NewScanner(file)
	re := regexp.MustCompile("###.*###")
	variableNotFound := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		subString := re.FindString(line)
		if subString != "" {
			variableName := strings.Replace(subString, "#", "", 6)
			value, ok := myEnv[variableName]

			if ok == false {
				variableNotFound = append(variableNotFound, variableName)
			}

			line = strings.Replace(line, subString, value, 1)
		}
		if line == "---" {

			err = checkIfVariableWasNotFound(variableNotFound)
			if err != nil {
				return err
			}
			err = functionCall(splitLines)

			if err != nil {
				return err
			}
			splitLines = []string{}

			continue
		}
		splitLines = append(splitLines, line)
	}
	err = checkIfVariableWasNotFound(variableNotFound)
	if err != nil {
		return err
	}
	return functionCall(splitLines)
}

func checkIfVariableWasNotFound(variableNotFound []string) error {
	if len(variableNotFound) > 0 {
		return errors.New(fmt.Sprintf("The Variables were not found in .env file: %s", strings.Join(variableNotFound, ", ")))
	}

	return nil
}
