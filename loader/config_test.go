package loader

import (
	"testing"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"os"
)

func TestConfig_LoadConfigFromPath(t *testing.T) {
	os.Clearenv()
	appFS := afero.NewMemMapFs()
	// create test files and directories
	afero.WriteFile(appFS, "src/mainFile", []byte("project_id: ###FOO###\n---\ntest: ###FOOBAR###"), 0644)

	oldFileSystem := fileSystemWrapper
	fileSystemWrapper = appFS
	oldEnvReader := envLoader
	// as we are exiting, revert sqlOpen back to oldSqlOpen at end of function
	defer func() {
		envLoader = oldEnvReader
		fileSystemWrapper = oldFileSystem
	}()

	envLoader = func(filenames ...string) error {
		os.Setenv("FOO", "BAR")
		os.Setenv("FOOBAR", "BARBAR")

		return nil
	}

	config, err := new(Config).LoadConfigFromPath("src/mainFile")

	assert.NoError(t, err)

	assert.Equal(t, "BAR", config.ProjectID)
}
