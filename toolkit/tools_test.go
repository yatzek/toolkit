package toolkit

import "testing"

func TestRandomString(t *testing.T) {
	var tools Tools
	s := tools.RandomString(10)
	if len(s) != 10 {
		t.Error("RandomString should return a string of length 10")
	}
}
