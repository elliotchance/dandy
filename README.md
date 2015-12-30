Automatically generate tests from your Go source code.

```go
package main

func Sign(x int) int {
  if x < 0 {
    return 1
  }
  if x > 0 {
    return 2
  }

  return 3
}
```

```bash
go run dandy.go -- myfile.go
```

```go
func TestSignXIsLessThan0(t *testing.T) {
	result := Sign(-1)
	if result != 1 {
		t.Error("Failed")
	}
}

func TestSignXIsGreaterThan0(t *testing.T) {
	result := Sign(1)
	if result != 2 {
		t.Error("Failed")
	}
}

func TestSign(t *testing.T) {
	result := Sign(0)
	if result != 3 {
		t.Error("Failed")
	}
}
```

The tests are rendered from the intermediate JSON format that provides
information on individual paths, steps taken, input and return values:

```json
{
  "Functions": {
    "Sign": {
      "Type": "int",
      "Args": {
        "x": "int"
      },
      "Paths": {
        "": {
          "Steps": [
            "4: if x \u003c 0 {",
            "7: if x \u003e 0 {",
            "11: return"
          ],
          "Params": {
            "x": 0
          },
          "Result": 3
        },
        "XIsGreaterThan0": {
          "Steps": [
            "4: if x \u003c 0 {",
            "7: if x \u003e 0 {",
            "8: return"
          ],
          "Params": {
            "x": 1
          },
          "Result": 2
        },
        "XIsLessThan0": {
          "Steps": [
            "4: if x \u003c 0 {",
            "5: return"
          ],
          "Params": {
            "x": -1
          },
          "Result": 1
        }
      }
    }
  }
}
```
