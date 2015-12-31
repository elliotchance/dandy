package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func readFile(path string) string {
	buf := bytes.NewBuffer(nil)
	file, err := os.Open(path)
	check(err)
	io.Copy(buf, file)
	file.Close()
	return string(buf.Bytes())
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runTest(name string, path string) (bool, int) {
	out, err := exec.Command("go", "run", "dandy.go", "--", path).CombinedOutput()

	success := false
	functions := 0
	if fileExists(path[:len(path)-3] + ".txt") {
		if strings.Contains(string(out), readFile(path[:len(path)-3]+".txt")) {
			success = true
		}
	} else {
		check(err)

		var data map[string]interface{}
		err = json.Unmarshal(out, &data)
		check(err)

		functions += len(data["Functions"].(map[string]interface{}))

		if strings.TrimSpace(string(out)) ==
			strings.TrimSpace(readFile(path[:len(path)-3]+".json")) {
			success = true
		}
	}

	if success {
		green := color.New(color.FgGreen).SprintFunc()
		term := fmt.Sprintf(" (%d functions)", functions)
		if functions == 1 {
			term = fmt.Sprintf(" (%d function)", functions)
		}
		if functions == 0 {
			term = ""
		}
		fmt.Printf("  %s%s\n", green("✓ "+name), term)
	} else {
		color.Red("  ✗ %s\n%s\n", name, string(out))
	}

	return success, functions
}

func runSuite(name string) (int, int, int) {
	c := color.New(color.FgYellow).Add(color.Bold).Add(color.Underline)
	c.Println("\n" + name)

	files, err := ioutil.ReadDir("tests/" + name)
	check(err)
	passed := 0
	failed := 0
	functions := 0
	for _, file := range files {
		if fileName := file.Name(); fileName[len(fileName)-3:] == ".go" {
			success, functionCount := runTest(fileName, "tests/"+name+"/"+fileName)
			if success {
				passed += 1
			} else {
				failed += 1
			}
			functions += functionCount
		}
	}

	return failed, passed, functions
}

func runSuites() (int, int, int) {
	var failedCount, passedCount, functionCount int
	suites, err := ioutil.ReadDir("tests")
	check(err)

	for _, suite := range suites {
		failed, passed, functions := runSuite(suite.Name())
		failedCount += failed
		passedCount += passed
		functionCount += functions
	}

	return failedCount, passedCount, functionCount
}

func main() {
	failedCount, passedCount, functionCount := runSuites()

	fmt.Printf("\n%d functions, %d tests passed, %d tests failed.\n\n", functionCount,
		passedCount, failedCount)
	if failedCount > 0 {
		os.Exit(1)
	}
}
