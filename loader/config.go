package loader

import (
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

const StagingEnvironment = "staging"
const ProductionEnvironment = "production"

type ConfigLoaderInterface interface {
	LoadConfigFromPath(filepath string) (Config, error)
}

type Cleanup struct {
	ImagePath string `yaml:"image_path"`
}

type DNSConfig struct {
	ProjectID    string `yaml:"project_id"`
	ManagedZone  string `yaml:"managed_zone"`
	DomainSuffix string `yaml:"domain_suffix"`
	BaseDomain   string `yaml:"base_domain"`
	DomainSpacer string `yaml:"domain_spacer"`
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

var fileSystemWrapper = afero.NewOsFs()

func (c *Config) LoadConfigFromPath(filepath string) (Config, error) {
	config := Config{}

	err := ReplaceVariablesInFile(fileSystemWrapper, filepath, func(splitLines []string) error {
		return yaml.Unmarshal([]byte(strings.Join(splitLines, "\n")), &config)
	})

	return config, err
}
