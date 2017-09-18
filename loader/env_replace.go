package loader

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/afero"
	"os"
)

var envLoader = godotenv.Load

type Callable func([]string) error

type ReplaceFunc func(fileSystem afero.Fs, path string, functionCall Callable) error

func ReplaceVariablesInFile(fileSystem afero.Fs, path string, functionCall Callable) error {
	file, err := fileSystem.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	err = envLoader()
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
			value, ok := os.LookupEnv(variableName)

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
