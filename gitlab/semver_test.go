package gitlab

import (
	"sort"
	"testing"
)

func TestSemVer(t *testing.T) {
	t.Run("test parsing", func(t *testing.T) {
		s := parseTag(tagInfo{
			Name: "v1.2.3",
			Commit: commitInfo{
				Date: "test_date",
				Id:   "abcdef1234567890",
			},
		})
		if s.major != 1 {
			t.Errorf("unexpected major version found")
		}
		if s.minor != 2 {
			t.Errorf("unexpected minor version found")
		}
		if s.patch != 3 {
			t.Errorf("unexpected patch version found")
		}
		if s.raw != "v1.2.3" {
			t.Errorf("unexpected raw tag found")
		}
		if s.date != "test_date" {
			t.Errorf("unexpected date found")
		}
	})
	t.Run("test sort by major", func(t *testing.T) {
		s1 := semVer{
			major: 5,
			minor: 1,
			patch: 1,
			raw:   "v5.1.1",
			date:  "test_date",
		}
		s2 := semVer{
			major: 4,
			minor: 5,
			patch: 5,
			raw:   "v4.5.5",
			date:  "test_date",
		}
		semVerList := semVerList{s1, s2}
		sort.Sort(semVerList)
		if semVerList[0].major != 4 {
			t.Errorf("sorted incorrectly")
		}
	})
	t.Run("test sort by minor", func(t *testing.T) {
		s1 := semVer{
			major: 1,
			minor: 2,
			patch: 1,
			raw:   "v1.2.1",
			date:  "test_date",
		}
		s2 := semVer{
			major: 1,
			minor: 1,
			patch: 5,
			raw:   "v1.1.5",
			date:  "test_date",
		}
		semVerList := semVerList{s1, s2}
		sort.Sort(semVerList)
		if semVerList[0].minor != 1 {
			t.Errorf("sorted incorrectly")
		}
	})
	t.Run("test sort by patch", func(t *testing.T) {
		s1 := semVer{
			major: 1,
			minor: 1,
			patch: 5,
			raw:   "v1.1.5",
			date:  "test_date",
		}
		s2 := semVer{
			major: 1,
			minor: 1,
			patch: 1,
			raw:   "v1.1.1",
			date:  "test_date",
		}
		semVerList := semVerList{s1, s2}
		sort.Sort(semVerList)
		if semVerList[0].patch != 1 {
			t.Errorf("sorted incorrectly")
		}
	})
}
