package loader

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestConfig_LoadConfigFromPath(t *testing.T) {
	os.Clearenv()
	appFS := afero.NewMemMapFs()

	var configFile = `cluster:
  type: gcp
  project_id: ###FOO###
  zone: europe-west1-d
  cluster_id: ###FOOBAR###`

	// create test files and directories
	afero.WriteFile(appFS, "src/mainFile", []byte(configFile), 0644)

	oldFileSystem := fileSystemWrapper
	fileSystemWrapper = appFS
	oldEnvReader := envLoader

	defer func() {
		envLoader = oldEnvReader
		fileSystemWrapper = oldFileSystem
	}()

	envLoader = func(filenames ...string) error {
		os.Setenv("FOO", "BAR")
		os.Setenv("FOOBAR", "BARBAR")

		return nil
	}

	config, err := NewConfigLoader().LoadConfigFromPath("src/mainFile")

	assert.NoError(t, err)

	assert.Equal(t, "BAR", config.Cluster.ProjectID)
	assert.Equal(t, "gcp", config.Cluster.Type)
}
