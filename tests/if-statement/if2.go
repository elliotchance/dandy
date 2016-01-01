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
