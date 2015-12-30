package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
  "os/exec"
)

func ucFirst(str string) string {
	return strings.ToUpper(str[0:1]) + str[1:]
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func atoi(str string) int {
	x, err := strconv.Atoi(str)
	check(err)
	return x
}

type IntDomain struct {
	min, max, impossible int
}

type Path struct {
	Steps                 []string
	conditionDescriptions []string
	domains               map[string]IntDomain
	Params                map[string]int
  Result                int
}

type Function struct {
	Type  string
	Args  map[string]string
	Paths map[string]Path
}

type File struct {
	Functions map[string]Function
}

const INT_DOMAIN_NOT_SET = -100000

func newIntDomain() IntDomain {
	return IntDomain{INT_DOMAIN_NOT_SET, INT_DOMAIN_NOT_SET, INT_DOMAIN_NOT_SET}
}

func getValue(x ast.Expr) int {
	switch v := x.(type) {
	case *ast.UnaryExpr:
		return -getValue(v.X)
	case *ast.BasicLit:
		value, _ := strconv.Atoi(v.Value)
		return value
	default:
		panic("oh noes 2")
	}
}

func getPathDescription(conditionDescriptions []string) string {
	if len(conditionDescriptions) == 0 {
		return ""
	}

	return conditionDescriptions[len(conditionDescriptions)-1]
}

func printExpr(expr ast.Stmt) {
	b, err := json.MarshalIndent(expr, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}
	os.Stdout.Write(b)
}

func printList(list []ast.Stmt) {
	for _, item := range list {
		printExpr(item)
	}
}

func newPath() *Path {
	path := new(Path)
	path.domains = make(map[string]IntDomain)
	path.Steps = make([]string, 0)
	path.Params = make(map[string]int)
	return path
}

func clonePath(path *Path) *Path {
	newPath := newPath()
	for k, p := range path.domains {
		newPath.domains[k] = p
	}
	for _, v := range path.Steps {
		newPath.Steps = append(newPath.Steps, v)
	}

	return newPath
}

func valueIsLegal(domain IntDomain, value int) bool {
	if domain.min != INT_DOMAIN_NOT_SET && value < domain.min {
		return false
	}
	if domain.max != INT_DOMAIN_NOT_SET && value > domain.max {
		return false
	}
	//if (in_array(value, $this->impossibleValues)) {
	if value == domain.impossible {
		return false
	}

	return true
}

func random(min, max int) int {
	return rand.Intn(max - min) + min
}

func calculateParam(param IntDomain) int {
	// We may have some known values that are outside the permitted range that
	// need to be pruned off now.
	// possibleValues := array_filter(
	//         $this->possibleValues,
	//         $valueIsLegal
	//     );

	// If we still have a remaining possible value we should use them. Let's
	// just take the first one that was set.
	// if (count($this->possibleValues) > 0) {
	//     return $this->possibleValues[0];
	// }

	// Now we try and pick a value that exists within the range of the
	// permissible values. Starting with some predefined values.
	valuesToTry := []int{
		0,
    1,
    -1,
		param.min,
		param.max,
		param.min + 1,
		param.max - 1,
	}

	for _, value := range valuesToTry {
		if value == INT_DOMAIN_NOT_SET {
			continue
		}

		if !valueIsLegal(param, value) {
			continue
		}

		return value
	}

	// Nothing found so far so now we have to start guessing. Setup bounds
	// so that we don't brute-force values that we already know are invalid.
	min := -100 - 1 // -PHP_INT_MAX - 1;
	if param.min != INT_DOMAIN_NOT_SET {
		min = param.min
	}
	max := 100 // PHP_INT_MAX
	if param.max != INT_DOMAIN_NOT_SET {
		max = param.max
	}

	// In case something goes wrong we only test a set amount before giving
	// up. It's very important that we set the random seed so that this
	// method will return repeatable results, even under random conditions.
	rand.Seed(0)
	try := 0
	for try < 100 {
		value := random(min, max)
		if valueIsLegal(param, value) {
			// Make sure we reset the random seed.
			rand.Seed(time.Now().UTC().UnixNano())

			return value
		}
		try += 1
	}

	// Make sure we reset the random seed.
	rand.Seed(time.Now().UTC().UnixNano())

	// We have to give up at this point.
	fmt.Printf("%v", param)
	panic("Could not resolve value.")
}

func calculateParamsForPath(path *Path) {
	for paramName, domain := range path.domains {
		path.Params[paramName] = calculateParam(domain)
	}
}

func getLineNumber(fset *token.FileSet, x interface{}) string {
	switch y := x.(type) {
	case token.Pos:
		return strconv.Itoa(fset.Position(y).Line)
	case *ast.ReturnStmt:
		return getLineNumber(fset, y.Return)
	case *ast.Ident:
		return getLineNumber(fset, y.NamePos)
	case *ast.IfStmt:
		return getLineNumber(fset, y.Cond)
	case *ast.BinaryExpr:
		return getLineNumber(fset, y.X)
	default:
		panic(y)
	}
}

func getConditionDescription(x interface{}, isTrue bool) string {
	switch y := x.(type) {
	case *ast.BasicLit:
		return y.Value
	case *ast.Ident:
		return ucFirst(y.String())
	case *ast.IfStmt:
		return getConditionDescription(y.Cond, isTrue)
	case *ast.BinaryExpr:
		word := ""
		switch y.Op {
		case token.LSS:
			word = "IsLessThan"
		case token.GTR:
			word = "IsGreaterThan"
		default:
			panic(y.Op)
		}
		return getConditionDescription(y.X, true) + word +
			getConditionDescription(y.Y, true)
	default:
		panic(y)
	}
}

func processStatements(fset *token.FileSet, path *Path, stmts []ast.Stmt, lines []string) map[string]Path {
All:
	for i, stmt := range stmts {
		lineNumber := getLineNumber(fset, stmt)
		switch s := stmt.(type) {
		case *ast.IfStmt:
			newPath := clonePath(path)
			numericalLineNumber, _ := strconv.Atoi(lineNumber)
			line := lineNumber + ": " + strings.TrimSpace(lines[numericalLineNumber-1])
			path.Steps = append(path.Steps, line)
			newPath.Steps = append(newPath.Steps, line)
			path.conditionDescriptions = append(path.conditionDescriptions,
				getConditionDescription(s, true))

			switch e := s.Cond.(type) {
			case *ast.BinaryExpr:
				arg := e.X.(*ast.Ident).Name
				value, err := strconv.Atoi(e.Y.(*ast.BasicLit).Value)
				if err != nil {
					panic(err)
				}

				p1 := path.domains[arg]
				p2 := newPath.domains[arg]

				switch e.Op.String() {
				case "<":
					p1.max = value
					p1.impossible = value
					p2.min = value
				case ">":
					p1.min = value
					p1.impossible = value
					p2.max = value
				default:
					panic(e.Op)
				}

				path.domains[arg] = p1
				newPath.domains[arg] = p2
			default:
				panic(e)
			}

			rest := stmts[i+1:]

			a := processStatements(fset, path, append(s.Body.List, rest...), lines)
			b := processStatements(fset, newPath, rest, lines)

			for _, v := range b {
				a[getPathDescription(v.conditionDescriptions)] = v
			}

			for _, v := range a {
				calculateParamsForPath(&v)
			}

			return a
		case *ast.ReturnStmt:
			path.Steps = append(path.Steps, lineNumber+": return")
			break All
		default:
			panic("oh noes")
		}
	}

	paths := make(map[string]Path)
	paths[getPathDescription(path.conditionDescriptions)] = *path
	return paths
}

func generateTests(results map[string]interface{}, file *File) {
  for functionName, function := range file.Functions {
    for pathName, path := range function.Paths {
      fmt.Printf("func Test%s%s(t *testing.T) {\n", functionName, pathName)
      fmt.Printf("\tresult := %s(", functionName)
      for _, paramValue := range path.Params {
        fmt.Printf("%v", paramValue)
      }
      fmt.Printf(")\n\tif result != %v {\n\t\tt.Error(\"Failed\")\n\t}\n}\n\n",
        results[pathName].(string))
    }
  }
}

func writeString(fo *os.File, str string) {
  if _, err := fo.Write([]byte(str)); err != nil {
    panic(err)
  }
}

func generateIntropection(lines []string, file *File) map[string]interface{} {
	tmpFile := "tests/if2_tmp.go"
  fo, err := os.Create(tmpFile)
  if err != nil {
      panic(err)
  }
  defer func() {
    if err := fo.Close(); err != nil {
      panic(err)
    }
  }()

  writeString(fo, lines[0] + "\n")
  writeString(fo, "import (\n\t\"encoding/json\"\n\t\"os\"\n)\n")
  for _, line := range lines[1:] {
    writeString(fo, line + "\n")
  }
  writeString(fo, "\nfunc main() {\n")

  writeString(fo, "\tresults := make(map[string]string)\n")
  writeString(fo, "\tvar b []byte\n")
  for functionName, function := range file.Functions {
    for pathName, path := range function.Paths {
      writeString(fo, fmt.Sprintf("\tb, _ = json.Marshal(%s(", functionName))
      for _, paramValue := range path.Params {
        writeString(fo, fmt.Sprintf("%v", paramValue))
      }
      writeString(fo, "))\n")
      writeString(fo, fmt.Sprintf("\tresults[\"%s\"] = string(b)\n", pathName))
    }
  }
  writeString(fo, "\n\tb, _ = json.MarshalIndent(results, \"\", \"  \")\n")
	writeString(fo, "\tos.Stdout.Write(b)\n")
  writeString(fo, "}\n")

  out, err := exec.Command("go", "run", tmpFile).Output()
  if err != nil {
    panic(err)
  }

  var results map[string]interface{}
  if err = json.Unmarshal(out, &results); err != nil {
    panic(err)
  }

	for functionName, function := range file.Functions {
		for pathName := range function.Paths {
			ref := file.Functions[functionName].Paths[pathName]
			ref.Result = atoi(results[pathName].(string))
			file.Functions[functionName].Paths[pathName] = ref
		}
	}

	os.Remove(tmpFile)

  return results
}

func main() {
	flag.Parse()

	buf := bytes.NewBuffer(nil)
	file, _ := os.Open(flag.Arg(0)) // Error handling elided for brevity.
	io.Copy(buf, file)              // Error handling elided for brevity.
	file.Close()
	src := string(buf.Bytes())

	lines := strings.Split(src, "\n")

	// Create the AST by parsing src.
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		panic(err)
	}

	// Print the AST.
	//ast.Print(fset, f)

	out := File{}
	out.Functions = make(map[string]Function)

	for _, decl := range f.Decls {
		name := decl.(*ast.FuncDecl).Type.Results.List[0].Type.(*ast.Ident).Name

		path := newPath()
		params := make(map[string]string)
		for _, param := range decl.(*ast.FuncDecl).Type.Params.List {
			params[param.Names[0].Name] = param.Type.(*ast.Ident).Name
			path.domains[param.Names[0].Name] = newIntDomain()
		}

		paths := processStatements(fset, path, decl.(*ast.FuncDecl).Body.List, lines)

		out.Functions[decl.(*ast.FuncDecl).Name.Name] = Function{name, params, paths}
	}

  generateIntropection(lines, &out)

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}
	os.Stdout.Write(b)
  fmt.Printf("\n\n")

  //generateTests(results, &out)
}
