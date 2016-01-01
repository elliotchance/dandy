package main

// This test case contains all the basic non-equality numerical comparisons.

func LessThanInt(x int) bool {
	if x < 0 {
		return true
	}

	return false
}

func LessThanFloat(x float32) bool {
	if x < 0.5 {
		return true
	}

	return false
}

func GreaterThanInt(x int) bool {
	if x > 0 {
		return true
	}

	return false
}

func GreaterThanFloat(x float32) bool {
	if x > 0.5 {
		return true
	}

	return false
}

func LessThanEqualInt(x int) bool {
	if x <= 0 {
		return true
	}

	return false
}

func LessThanEqualFloat(x float32) bool {
	if x <= 0.5 {
		return true
	}

	return false
}

func GreaterThanEqualInt(x int) bool {
	if x >= 0 {
		return true
	}

	return false
}

func GreaterThanEqualFloat(x float32) bool {
	if x >= 0.5 {
		return true
	}

	return false
}
