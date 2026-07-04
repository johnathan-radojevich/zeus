package services

import "testing"

func TestFindImplementingClassNotImplemented(t *testing.T) {
	_, ok := FindImplementingClass("PolicyValidator")
	if ok {
		t.Fatal("expected search to be unimplemented")
	}

	_, ok = FindImplementingClass("")
	if ok {
		t.Fatal("expected empty query to miss")
	}
}
