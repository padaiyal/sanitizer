package e2e

import "testing"

// This will be invoked first, and it's being used as a wrapper around all the tests in package e2e
func TestMain(m *testing.M) {
	SetUp()
	m.Run()
	TearDown()
}
