package routing

func testAll(l int, f func(int) bool) bool {
	for i := 0; i < l; i++ {
		if !f(i) {
			return false
		}
	}
	return true
}

func testAny(l int, f func(int) bool) bool {
	for i := 0; i < l; i++ {
		if f(i) {
			return true
		}
	}
	return false
}
