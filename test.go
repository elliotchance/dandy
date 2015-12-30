package main

import (
	"io/ioutil"
  "os/exec"
	"bytes"
	"os"
	"io"
	"strings"
	"github.com/fatih/color"
	"fmt"
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

func runTest(name string, path string) bool {
	out, err := exec.Command("go", "run", "dandy.go", "--", path).CombinedOutput()
	if fileExists(path[:len(path) - 3] + ".txt") {
		if strings.Contains(string(out), readFile(path[:len(path) - 3] + ".txt")) {
			color.Green("  ✓ %s\n", name)
			return true
		}
	} else {
		check(err)
		if strings.TrimSpace(string(out)) == strings.TrimSpace(readFile(path[:len(path) - 3] + ".json")) {
			color.Green("  ✓ %s\n", name)
			return true
		}
	}

	color.Red("  ✗ %s\n%s\n", name, string(out))
	return false
}

func runSuite(name string) (int, int) {
	c := color.New(color.FgYellow).Add(color.Bold).Add(color.Underline)
	c.Println("\n" + name)

	files, err := ioutil.ReadDir("tests/" + name)
	check(err)
	passed := 0
	failed := 0
	for _, file := range files {
		if fileName := file.Name(); fileName[len(fileName) - 3:] == ".go" {
			if runTest(fileName, "tests/" + name + "/" + fileName) {
				passed += 1
			} else {
				failed += 1
			}
		}
	}

	return failed, passed
}

func runSuites() (int, int) {
	var failedCount, passedCount int

	for _, suite := range []string{"failures", "if-statement"} {
		failed, passed := runSuite(suite)
		failedCount += failed
		passedCount += passed
	}

	return failedCount, passedCount
}

func main() {
	failedCount, passedCount := runSuites()

	fmt.Printf("\n%d passed, %d failed.\n\n", passedCount, failedCount)
	if failedCount > 0 {
		os.Exit(1)
	}
}
