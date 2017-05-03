package config

import (
  "io/ioutil"

  "gopkg.in/yaml.v2"

  "kube-helper/util"
)

type Cleanup struct {
  ImagePath string `yaml:"image_path"`
  RepoUrl string `yaml:"repo_url"`
}

type DNSConfig struct {
  ProjectID string `yaml:"project_id"`
  ManagedZone string `yaml:"managed_zone"`
  DomainSuffix string `yaml:"domain_suffix"`
  CNameSuffix []string `yaml:"cname_suffix"`
}

type Config struct {
  KubernetesConfigFilepath string `yaml:"kubernetes_config_filepath"`
  ProjectID string `yaml:"project_id"`
  ClusterID string `yaml:"cluster_id"`
  Zone string
  Cleanup Cleanup
  DNS DNSConfig `yaml:"dns"`
  Database struct {
    Instance string
    BaseName string `yaml:"base_name"`
    PrefixBranchDatabase string `yaml:"prefix_branch_database"`
    Bucket string
  }
}
const tmpSplitFile = "tmp.yml"

func LoadConfigFromPath(filepath string) Config {
	config := Config{}

  util.ReplaceVariablesInFile(filepath, tmpSplitFile, func() {
    configHelper, err := ioutil.ReadFile(tmpSplitFile)
  	util.CheckError(err)

    err = yaml.Unmarshal(configHelper, &config)
    util.CheckError(err)
  })

  return config
}
