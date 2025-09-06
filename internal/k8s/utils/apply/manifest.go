package apply

import (
	"regexp"
	"strings"
)

var regex = regexp.MustCompile("(?:^|\\s*\n)---\\s*")

func SplitManifests(bigFile string) []string {

	res := make([]string, 0)

	bigFileTmp := strings.TrimSpace(bigFile)
	docs := regex.Split(bigFileTmp, -1)
	var count int
	for _, doc := range docs {

		if doc == "" {
			continue
		}

		doc = strings.TrimSpace(doc)
		res = append(res, doc)
		count = count + 1
	}
	return res
}
