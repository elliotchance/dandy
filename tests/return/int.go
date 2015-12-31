package main

func Int8() int8 {
  return 101
}

func Int16() int16 {
  return 102
}

func Int32() int32 {
  return 103
}

func Int64() int64 {
  return 104
}

func Uint8() uint8 {
  return 105
}

func Uint16() uint16 {
  return 106
}

func Uint32() uint32 {
  return 107
}

func Uint64() uint64 {
  return 108
}

// byte is an alias for uint8
func Byte() byte {
  return 109
}

// rune is an alias for int32
func Rune() rune {
  return 110
}

// There is also a set of predeclared numeric types with implementation-specific
// sizes:

func Int() int {
  return 111
}

func Uint() uint {
  return 112
}

func Uintptr() uintptr {
  return 113
}
