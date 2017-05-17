package loader

import (
	"github.com/stretchr/testify/assert"
	"github.com/spf13/afero"
	"strings"
	"testing"
	"errors"
)

func TestEnvReplace(t *testing.T) {
	appFS := afero.NewMemMapFs()
	// create test files and directories
	afero.WriteFile(appFS, "src/mainFile", []byte("key: ###FOO###\n---\ntest: ###FOOBAR###"), 0644)

	oldEnvReader := envReader
	// as we are exiting, revert sqlOpen back to oldSqlOpen at end of function
	defer func() { envReader = oldEnvReader }()

	envReader = func(filenames ...string) (map[string]string, error) {
		return map[string]string{
			"FOO":    "BAR",
			"FOOBAR": "BARBAR",
		}, nil

	}

	wasRun := false
	splitLinesData := []string{}
	err := ReplaceVariablesInFile(appFS, "src/mainFile", func(splitLines []string) error {
		wasRun = true
		splitLinesData = append(splitLinesData, splitLines...)

		return nil
	})
	assert.NoError(t, err)

	assert.Equal(t, "key: BAR\ntest: BARBAR", strings.Join(splitLinesData, "\n"))

	assert.True(t, wasRun)
}

func TestEnvReplaceWithErrorForEnvReader(t *testing.T) {
	appFS := afero.NewMemMapFs()
	// create test files and directories
	afero.WriteFile(appFS, "src/mainFile", []byte("key: ###FOO###\n---\ntest: ###FOOBAR###"), 0644)

	err := ReplaceVariablesInFile(appFS, "src/mainFile", func(splitLines []string) error {
		return nil
	})
	assert.EqualError(t, err, "open .env: no such file or directory")

}

func TestEnvReplaceWithErrorForFileOpen(t *testing.T) {
	appFS := afero.NewMemMapFs()
	// create test files and directories

	err := ReplaceVariablesInFile(appFS, "src/mainFile", func(splitLines []string) error {
		return nil
	})
	assert.EqualError(t, err, "open src/mainFile: file does not exist")

}

func TestEnvReplaceWithErrorInCallback(t *testing.T) {
	appFS := afero.NewMemMapFs()
	// create test files and directories
	afero.WriteFile(appFS, "src/mainFile", []byte("key: ###FOO###\n---\ntest: ###FOOBAR###"), 0644)

	oldEnvReader := envReader
	// as we are exiting, revert sqlOpen back to oldSqlOpen at end of function
	defer func() { envReader = oldEnvReader }()

	envReader = func(filenames ...string) (map[string]string, error) {
		return map[string]string{
			"FOO":    "BAR",
			"FOOBAR": "BARBAR",
		}, nil

	}

	wasRun := false
	err := ReplaceVariablesInFile(appFS, "src/mainFile", func(splitLines []string) error {
		wasRun = true

		return errors.New("explode")
	})
	assert.EqualError(t, err, "explode")

	assert.True(t, wasRun)
}

func TestEnvReplaceWithEnvironmentVariableNotFound(t *testing.T) {
	appFS := afero.NewMemMapFs()
	// create test files and directories
	afero.WriteFile(appFS, "src/mainFile", []byte("key: ###FOO###\ntest: ###FOOBAR###"), 0644)

	oldEnvReader := envReader
	// as we are exiting, revert sqlOpen back to oldSqlOpen at end of function
	defer func() { envReader = oldEnvReader }()

	envReader = func(filenames ...string) (map[string]string, error) {
		return map[string]string{}, nil

	}

	err := ReplaceVariablesInFile(appFS, "src/mainFile", func(splitLines []string) error {return nil})
	assert.EqualError(t, err, "The Variables were not found in .env file: FOO, FOOBAR")

}

func TestEnvReplaceWithEnvironmentVariableNotFoundForSplitFile(t *testing.T) {
	appFS := afero.NewMemMapFs()
	// create test files and directories
	afero.WriteFile(appFS, "src/mainFile", []byte("key: ###FOOBAR###\n---\ntest: ###FOO###"), 0644)

	oldEnvReader := envReader
	// as we are exiting, revert sqlOpen back to oldSqlOpen at end of function
	defer func() { envReader = oldEnvReader }()

	envReader = func(filenames ...string) (map[string]string, error) {
		return map[string]string{}, nil

	}

	err := ReplaceVariablesInFile(appFS, "src/mainFile", func(splitLines []string) error {return nil})
	assert.EqualError(t, err, "The Variables were not found in .env file: FOOBAR")

}
