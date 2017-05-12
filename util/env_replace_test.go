package util

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestEnvReplace(t *testing.T) {
	appFS := afero.NewMemMapFs()
	// create test files and directories
	afero.WriteFile(appFS, "src/mainFile", []byte("key: ###FOO###\n---\ntest: ###FOOBAR###"), 0644)

	oldEnvReader := EnvReader
	// as we are exiting, revert sqlOpen back to oldSqlOpen at end of function
	defer func() { EnvReader = oldEnvReader }()

	EnvReader = func(filenames ...string) (map[string]string, error) {
		return map[string]string{
			"FOO":    "BAR",
			"FOOBAR": "BARBAR",
		}, nil

	}

	wasRun := false
	splitLinesData := []string{}
	err := ReplaceVariablesInFile(appFS, "src/mainFile", func(splitLines []string) {
		wasRun = true
		splitLinesData = append(splitLinesData, splitLines...)
	})
	assert.NoError(t, err)

	assert.Equal(t, "key: BAR\ntest: BARBAR", strings.Join(splitLinesData, "\n"))

	assert.True(t, wasRun)
}

func TestEnvReplaceWithEnvironmentVariableNotFound(t *testing.T) {
	appFS := afero.NewMemMapFs()
	// create test files and directories
	afero.WriteFile(appFS, "src/mainFile", []byte("key: ###FOO###\ntest: ###FOOBAR###"), 0644)

	oldEnvReader := EnvReader
	// as we are exiting, revert sqlOpen back to oldSqlOpen at end of function
	defer func() { EnvReader = oldEnvReader }()

	EnvReader = func(filenames ...string) (map[string]string, error) {
		return map[string]string{}, nil

	}

	err := ReplaceVariablesInFile(appFS, "src/mainFile", func(splitLines []string) {})
	assert.EqualError(t, err, "The Variables were not found in .env file: FOO, FOOBAR")

}

func TestEnvReplaceWithEnvironmentVariableNotFoundForSplitFile(t *testing.T) {
	appFS := afero.NewMemMapFs()
	// create test files and directories
	afero.WriteFile(appFS, "src/mainFile", []byte("key: ###FOOBAR###\n---\ntest: ###FOO###"), 0644)

	oldEnvReader := EnvReader
	// as we are exiting, revert sqlOpen back to oldSqlOpen at end of function
	defer func() { EnvReader = oldEnvReader }()

	EnvReader = func(filenames ...string) (map[string]string, error) {
		return map[string]string{}, nil

	}

	err := ReplaceVariablesInFile(appFS, "src/mainFile", func(splitLines []string) {})
	assert.EqualError(t, err, "The Variables were not found in .env file: FOOBAR")

}
