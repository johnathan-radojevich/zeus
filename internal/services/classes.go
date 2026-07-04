package services

import "strings"

// ImplementingClass is a source file that implements the searched type.
type ImplementingClass struct {
	Name string
	Path string
}

// FindImplementingClass searches the project for classes implementing the given type.
// Repository scanning is not implemented yet.
func FindImplementingClass(query string) ([]ImplementingClass, bool) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, false
	}
	return nil, false
}
