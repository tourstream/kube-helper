package util

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
)

type callable func()

func ReplaceVariablesInFile(path string, tmpSplitFile string, functionCall callable) {
	absoulteFilePath, err := filepath.Abs(path)
	CheckError(err)
	file, err := os.Open(absoulteFilePath)
	CheckError(err)
	defer file.Close()

	f, err := os.Create(tmpSplitFile)
	CheckError(err)
	var myEnv map[string]string
	myEnv, err = godotenv.Read()
	CheckError(err)
	defer f.Close()
	scanner := bufio.NewScanner(file)
	re := regexp.MustCompile("###.*###")
	variableNotFound := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		subString := re.FindString(line)
		if (subString != "") {
			variableName := strings.Replace(subString, "#", "", 6)
			value, ok := myEnv[variableName]

			if (ok == false) {
				variableNotFound = append(variableNotFound, variableName)
			}

			line = strings.Replace(line, subString, value, 1)
		}
		if (line == "---") {

			f.Close()
			if (len(variableNotFound) == 0) {
				functionCall()
			}

			f, err = os.Create(tmpSplitFile)

			defer f.Close()
			continue;
		}

		f.WriteString(line + "\n")
		f.Sync()
	}
	f.Close()
	if (len(variableNotFound) > 0) {
		log.Fatalf("The Variables were not found in .env file:\n %s", variableNotFound)
	}
	functionCall()
}
