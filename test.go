package main

import (
	"io/ioutil"
  "os/exec"
	"bytes"
	"os"
	"io"
	"strings"
	"github.com/fatih/color"
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

func runTest(name string, path string) bool {
	out, _ := exec.Command("go", "run", "dandy.go", "--", path).CombinedOutput()
	//check(err)

	if strings.Contains(string(out), readFile(path[:len(path) - 3] + ".txt")) {
		color.Green("  ✓ %s\n", name)
		return true
	} else {
		color.Red("  ✗ %s\n%s\n", name, string(out))
		return false
	}
}

func runSuite(name string) {
	c := color.New(color.FgYellow).Add(color.Bold).Add(color.Underline)
	c.Println(name)

	files, err := ioutil.ReadDir("tests/" + name)
	check(err)
	for _, file := range files {
		if fileName := file.Name(); fileName[len(fileName) - 3:] == ".go" {
			runTest(fileName, "tests/" + name + "/" + fileName)
		}
	}
}

func main() {
	runSuite("failures")
}
