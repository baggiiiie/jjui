package jj

import (
	"regexp"
	"strings"
)

// ansiRegex matches ANSI escape sequences
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

const (
	moveBookmarkTemplate   = `separate(";", name, if(remote, "remote", "."), tracked, conflict, normal_target.contained_in("%s"), normal_target.commit_id().shortest(1)) ++ "\n"`
	allBookmarkTemplate    = `separate(";", name, if(remote, remote, "."), tracked, conflict, 'false', normal_target.commit_id().shortest(1)) ++ "\n"`
	simpleBookmarkTemplate = `
  if(conflict,
    label("bookmark", name) ++ " (conflict)",
    label("bookmark", name)
  ) ++ " " ++
  coalesce(
    normal_target.change_id().shortest(6),
    "(deleted)"
  ) ++ "\n"
`
)

type BookmarkRemote struct {
	Remote   string
	CommitId string
	Tracked  bool
}

type Bookmark struct {
	Name      string
	Local     *BookmarkRemote
	Remotes   []BookmarkRemote
	Conflict  bool
	Backwards bool
	CommitId  string
}

func (b Bookmark) IsDeletable() bool {
	return b.Local != nil
}

func (b Bookmark) IsTrackable() bool {
	return b.Local != nil && len(b.Remotes) == 0
}

// ParseSimpleBookmarkListOutput parses the "name [conflict] change_id" format
func ParseSimpleBookmarkListOutput(output string) []Bookmark {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var bookmarks []Bookmark

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by whitespace (one or more spaces)
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		// Parse name and conflict status (strip ANSI codes from name)
		name := stripAnsi(parts[0])
		conflict := false
		changeIdIdx := 1

		// Check if second part is "(conflict)"
		if len(parts) > 2 && parts[1] == "(conflict)" {
			conflict = true
			changeIdIdx = 2
		}

		if changeIdIdx >= len(parts) {
			continue
		}

		changeId := parts[changeIdIdx]

		bookmarks = append(bookmarks, Bookmark{
			Name:     name,
			Conflict: conflict,
			CommitId: changeId,
			Local: &BookmarkRemote{
				Remote:   ".",
				CommitId: changeId,
			},
		})
	}

	return bookmarks
}

func ParseBookmarkListOutput(output string) []Bookmark {
	lines := strings.Split(output, "\n")
	bookmarkMap := make(map[string]*Bookmark)
	var orderedNames []string

	for _, b := range lines {
		parts := strings.Split(b, ";")
		if len(parts) < 6 {
			continue
		}

		name := parts[0]
		name = strings.Trim(name, "\"")
		remoteName := parts[1]
		tracked := parts[2] == "true"
		conflict := parts[3] == "true"
		backwards := parts[4] == "true"
		commitId := parts[5]

		if remoteName == "git" {
			continue
		}

		bookmark, exists := bookmarkMap[name]
		if !exists {
			bookmark = &Bookmark{
				Name:      name,
				Conflict:  conflict,
				Backwards: backwards,
				CommitId:  commitId,
			}
			bookmarkMap[name] = bookmark
			orderedNames = append(orderedNames, name)
		}

		if remoteName == "." {
			bookmark.Local = &BookmarkRemote{
				Remote:   ".",
				CommitId: commitId,
				Tracked:  tracked,
			}
			bookmark.CommitId = commitId
		} else {
			remote := BookmarkRemote{
				Remote:   remoteName,
				Tracked:  tracked,
				CommitId: commitId,
			}
			if remoteName == "origin" {
				bookmark.Remotes = append([]BookmarkRemote{remote}, bookmark.Remotes...)
			} else {
				bookmark.Remotes = append(bookmark.Remotes, remote)
			}
		}
	}

	if len(orderedNames) == 0 {
		return nil
	}

	bookmarks := make([]Bookmark, len(orderedNames))
	for i, name := range orderedNames {
		bookmarks[i] = *bookmarkMap[name]
	}
	return bookmarks
}
