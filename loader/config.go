package loader

import (
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
)

const (
	// StagingEnvironment is used to apply a namespace
	StagingEnvironment = "staging"
	// ProductionEnvironment is used to apply a namespace
	ProductionEnvironment = "production"
)

var validate *validator.Validate

// ConfigLoader API
type ConfigLoader interface {
	LoadConfigFromPath(filepath string) (Config, error)
}

type Cleanup struct {
	ImagePath string `yaml:"image_path"`
}

type DNSConfig struct {
	ProjectID    string   `yaml:"project_id"`
	ManagedZone  string   `yaml:"managed_zone"`
	DomainSuffix string   `yaml:"domain_suffix"`
	BaseDomain   string   `yaml:"base_domain"`
	DomainSpacer string   `yaml:"domain_spacer"`
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

type Endpoints struct {
	Enabled bool
}

type Cluster struct {
	Type      string
	ProjectID string `yaml:"project_id" validate:"required"`
	ClusterID string `yaml:"cluster_id" validate:"required"`
	Zone      string
}

type Namespace struct {
	Prefix string
}

// Config for the kube-helper
type Config struct {
	KubernetesConfigFilepath string `yaml:"kubernetes_config_filepath"`
	Endpoints                Endpoints
	Cluster                  Cluster
	Bitbucket                Bitbucket
	Cleanup                  Cleanup
	DNS                      DNSConfig `yaml:"dns"`
	Database                 Database
	Namespace                Namespace `validate:"required"`
}

var fileSystemWrapper = afero.NewOsFs()

type configLoader struct{}

// NewConfigLoader is the constructor method and returns a service which implements ConfigLoader
func NewConfigLoader() ConfigLoader {
	return &configLoader{}
}

// LoadConfigFromPath loads from a config file the Config and validates the config
func (c *configLoader) LoadConfigFromPath(filepath string) (Config, error) {
	config := Config{}

	err := ReplaceVariablesInFile(fileSystemWrapper, filepath, func(splitLines []string) error {
		return yaml.Unmarshal([]byte(strings.Join(splitLines, "\n")), &config)
	})

	validate = validator.New()
	err = validate.Struct(config)

	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			return config, err
		}
	}

	return config, err
}
