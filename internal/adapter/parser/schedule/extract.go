package schedule

import (
	"regexp"
	"strings"
)

func extract(data string, exp *regexp.Regexp) map[string]string {
	match := exp.FindStringSubmatch(data)
	if len(match) > 0 {
		return extractData(match, exp)
	}
	return nil
}

func extractData(match []string, re *regexp.Regexp) map[string]string {
	data := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			data[name] = strings.TrimSpace(match[i])
		}
	}
	return data
}
