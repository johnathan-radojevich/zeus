package services

import "testing"

func TestFindXMLForRuleKeyNotImplemented(t *testing.T) {
	_, ok := FindXMLForRuleKey("MY-RULE-001")
	if ok {
		t.Fatal("expected search to be unimplemented")
	}

	_, ok = FindXMLForRuleKey("")
	if ok {
		t.Fatal("expected empty key to miss")
	}
}
