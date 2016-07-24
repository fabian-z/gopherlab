package revert

import _ "fmt"

import _ "image/png"

func F() {
	var x, y, z int
	_ = y
	_ = z

	if true {
		_ = x
	}

	for i := range []int{} {
		_ = i
	}
}
