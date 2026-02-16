package utils

import "strings"

// Where complements the selection for the query.
func Where(parameters []string) string {
	if len(parameters) == 0 {
		return ""
	}

	return " WHERE " + strings.Join(parameters, " AND ")
}
