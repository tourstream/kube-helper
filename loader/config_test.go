package loader

import (
	"testing"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestConfig_LoadConfigFromPath(t *testing.T) {
	appFS := afero.NewMemMapFs()
	// create test files and directories
	afero.WriteFile(appFS, "src/mainFile", []byte("project_id: ###FOO###\n---\ntest: ###FOOBAR###"), 0644)

	oldFileSystem := fileSystemWrapper
	fileSystemWrapper = appFS
	oldEnvReader := envReader
	// as we are exiting, revert sqlOpen back to oldSqlOpen at end of function
	defer func() {
		envReader = oldEnvReader
		fileSystemWrapper = oldFileSystem
	}()

	envReader = func(filenames ...string) (map[string]string, error) {
		return map[string]string{
			"FOO":    "BAR",
			"FOOBAR": "BARBAR",
		}, nil

	}

	config, err := new(Config).LoadConfigFromPath("src/mainFile")

	assert.NoError(t, err)

	assert.Equal(t, "BAR", config.ProjectID)
}
