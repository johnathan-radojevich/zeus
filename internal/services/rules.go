package services

import "strings"

// RuleXMLFile is a rule definition file located in the repo.
type RuleXMLFile struct {
	Path    string
	Content string
}

// FindXMLForRuleKey searches the project for an XML file containing the rule key.
// Repository scanning is not implemented yet.
func FindXMLForRuleKey(key string) (RuleXMLFile, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return RuleXMLFile{}, false
	}
	return RuleXMLFile{}, false
}
