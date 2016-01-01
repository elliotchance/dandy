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
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func random(min, max int) int {
	return rand.Intn(max-min) + min
}

func interfaceToFloat(value interface{}) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case float64:
		return v
	default:
		panic(v)
	}
}

// Return the correct value and representation of a constant value. Currently
// only supports numeric types. It will automatically choose the highest
// resolution type to satisfy the value.
func valueFromConstant(str string) interface{} {
	// Try first as a float
	floatValue, err := strconv.ParseFloat(str, 64)
	if err == nil {
		return floatValue
	}

	intValue, err := strconv.Atoi(str)
	if err == nil {
		return intValue
	}

	panic("Cannot understand constant: " + str)
}

// Convert an AST expression to the string representing it's type.
func astTypeToString(expr ast.Expr) string {
	switch rt := expr.(type) {
	case *ast.Ident:
		// This would be something simple like "int32"
		return rt.Name
	case *ast.ArrayType:
		// This would be an array/slice type like "[]int32"
		return "[]" + astTypeToString(rt.Elt)
	case *ast.MapType:
		// This would be something like "map[string]int"
		return "map[" + astTypeToString(rt.Key) + "]" + astTypeToString(rt.Value)
	default:
		panic(rt)
	}
}

// Decode a JSON string. This method expects that theJson is a valid encoded
// JSON string or it will throw a panic.
func decodeJsonString(theJson string) string {
	var result string
	err := json.Unmarshal([]byte(theJson), &result)
	check(err)

	return result
}

// Decode a JSON array. This method expects that theJson is a valid encoded
// JSON array or it will throw a panic.
func decodeJsonArray(theJson string) []interface{} {
	var result []interface{}
	err := json.Unmarshal([]byte(theJson), &result)
	check(err)

	return result
}

// Decode a JSON object. This method expects that theJson is a valid encoded
// JSON object or it will throw a panic.
func decodeJsonObject(theJson string) map[string]interface{} {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(theJson), &result)
	check(err)

	return result
}

// Convert the rawValue which will be a JSON-encoded string into a value for the
// type provided. For example, if the rawValue is "123" and we need a uint32 it
// would return 123 as an integer. This method also supported decoding
// array/slices and maps.
func getValueForType(typeName string, rawValue string) interface{} {
	// Type is an array/slice.
	if strings.HasPrefix(typeName, "[]") {
		return decodeJsonArray(rawValue)
	}

	// Type is a map.
	if strings.HasPrefix(typeName, "map[") {
		return decodeJsonObject(rawValue)
	}

	// It must be a simple type.
	switch typeName {
	case "int", "uint", "uintptr", "byte", "rune",
		"int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64":
		return atoi(rawValue)
	case "bool":
		return atob(rawValue)
	case "float32", "float64":
		return atof(rawValue)
	case "string":
		// The rawValue will be marshalled as JSON, we have to unmarshal it for a
		// string since it will contain escape sequences.
		return decodeJsonString(rawValue)
	}

	notSupported(typeName)
	return nil
}

// This is a convenience method to throw a panic when some feature. The feature
// is just a string that should make sense to the user.
func notSupported(feature string) {
	panic("'" + feature + "' is not supported.")
}

// Convert the first letter of a string to uppercase.
func ucFirst(str string) string {
	return strings.ToUpper(str[0:1]) + str[1:]
}

// This is a convenience method to throw a panic if err is not nil.
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

func atob(str string) bool {
	switch str {
	case "true":
		return true
	case "false":
		return false
	}

	panic(str)
}

func atof(str string) float64 {
	x, err := strconv.ParseFloat(str, 64)
	check(err)
	return x
}

type Domain struct {
	min, max, impossible interface{}
}

type Path struct {
	Steps                 []string
	conditionDescriptions []string
	domains               map[string]Domain
	Params                map[string]interface{}
	Result                interface{}
}

type Function struct {
	Type  string
	Args  map[string]string
	Paths map[string]Path
}

type File struct {
	Functions map[string]Function
}

const DOMAIN_NOT_SET = -100000

func newDomain() Domain {
	return Domain{DOMAIN_NOT_SET, DOMAIN_NOT_SET, DOMAIN_NOT_SET}
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
	path.domains = make(map[string]Domain)
	path.Steps = make([]string, 0)
	path.Params = make(map[string]interface{})
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

func valueIsLegal(domain Domain, value interface{}) bool {
	v := interfaceToFloat(value)

	if domain.min != DOMAIN_NOT_SET && v < interfaceToFloat(domain.min) {
		return false
	}
	if domain.max != DOMAIN_NOT_SET && v > interfaceToFloat(domain.max) {
		return false
	}
	//if (in_array(value, $this->impossibleValues)) {
	if v == interfaceToFloat(domain.impossible) {
		return false
	}

	return true
}

func calculateParam(param Domain) interface{} {
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
	valuesToTry := []interface{}{
		0,
		1,
		-1,
		param.min,
		param.max,
	}

	switch v := param.min.(type) {
	case int:
		valuesToTry = append(valuesToTry, v + 1, v - 1)
	case float64:
		valuesToTry = append(valuesToTry, v + 1.0, v - 1.0)
	default:
		panic(v)
	}

	for _, value := range valuesToTry {
		if value == DOMAIN_NOT_SET {
			continue
		}

		if !valueIsLegal(param, value) {
			continue
		}

		return value
	}

	// Nothing found so far so now we have to start guessing. Setup bounds
	// so that we don't brute-force values that we already know are invalid.
	min := -100.0 - 1 // -PHP_INT_MAX - 1;
	if param.min != DOMAIN_NOT_SET {
		min = param.min.(float64)
	}
	max := 100.0 // PHP_INT_MAX
	if param.max != DOMAIN_NOT_SET {
		max = param.max.(float64)
	}

	// In case something goes wrong we only test a set amount before giving
	// up. It's very important that we set the random seed so that this
	// method will return repeatable results, even under random conditions.
	rand.Seed(0)
	try := 0
	for try < 100 {
		value := random(int(min), int(max))
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
	case *ast.AssignStmt:
		return getLineNumber(fset, y.Lhs)
	case []ast.Expr:
		return getLineNumber(fset, y[0])
	case *ast.IndexExpr:
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
		case token.LEQ:
			word = "IsLessThanOrEqual"
		case token.GEQ:
			word = "IsGreaterThanOrEqual"
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
				value := valueFromConstant(e.Y.(*ast.BasicLit).Value)

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
				case "<=":
					p1.max = value
					p2.impossible = value
					p2.min = value
				case ">=":
					p1.min = value
					p2.impossible = value
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
		case *ast.AssignStmt:
			// Do nothing here. We do not need to explain lines that arn't important
		default:
			panic(s)
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
	tmpFile := "tests/tmp.go"
	fo, err := os.Create(tmpFile)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	writeString(fo, lines[0]+"\n")
	writeString(fo, "import (\n\t\"encoding/json\"\n\t\"os\"\n)\n")
	for _, line := range lines[1:] {
		writeString(fo, line+"\n")
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
			writeString(fo, fmt.Sprintf("\tresults[\"%s:%s\"] = string(b)\n",
				functionName, pathName))
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
	err = json.Unmarshal(out, &results)
	check(err)

	for functionName, function := range file.Functions {
		for pathName := range function.Paths {
			ref := file.Functions[functionName].Paths[pathName]
			result := results[functionName+":"+pathName].(string)
			ref.Result = getValueForType(function.Type, result)
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
	// ast.Print(fset, f)
	// panic("ok")

	out := File{}
	out.Functions = make(map[string]Function)

	for _, decl := range f.Decls {
		returnType := decl.(*ast.FuncDecl).Type.Results.List[0].Type
		name := astTypeToString(returnType)

		path := newPath()
		params := make(map[string]string)
		for _, param := range decl.(*ast.FuncDecl).Type.Params.List {
			params[param.Names[0].Name] = param.Type.(*ast.Ident).Name
			path.domains[param.Names[0].Name] = newDomain()
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
