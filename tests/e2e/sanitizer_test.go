package e2e

import "testing"

func TestMain(m *testing.M) {
	SetUp()
	m.Run()
	TearDown()
}
