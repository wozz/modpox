package gitlab

import (
	"strconv"
	"strings"
)

type semVer struct {
	major int
	minor int
	patch int
	raw   string
	date  string
}

func parseTag(in tagInfo) semVer {
	if !strings.HasPrefix(in.Name, "v") {
		// TODO support sub modules
		return semVer{raw: in.Name}
	}
	parts := strings.Split(in.Name, ".")
	if len(parts) != 3 {
		return semVer{raw: in.Name}
	}
	major, _ := strconv.Atoi(parts[0][1:])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])
	return semVer{
		major: major,
		minor: minor,
		patch: patch,
		raw:   in.Name,
		date:  in.Commit.Date,
	}
}

type semVerList []semVer

func (s semVerList) Less(i, j int) bool {
	if s[i].major < s[j].major {
		return true
	}
	if s[i].major > s[j].major {
		return false
	}
	if s[i].minor < s[j].minor {
		return true
	}
	if s[i].minor > s[j].minor {
		return false
	}
	return s[i].patch < s[j].patch
}

func (s semVerList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s semVerList) Len() int {
	return len(s)
}
