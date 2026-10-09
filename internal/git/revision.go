package git

import "strings"

// WorktreeRelativeRevision identifies expressions whose meaning depends on a
// checkout or its reflog. Callers must not resolve these in a bare Git store.
func WorktreeRelativeRevision(revision string) (string, bool) {
	if revision == "-" {
		return "@{-1}", true
	}
	if strings.Contains(revision, "@{") {
		return revision, true
	}
	for _, head := range []string{"HEAD", "@", "ORIG_HEAD", "FETCH_HEAD", "MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "REBASE_HEAD", "BISECT_HEAD", "AUTO_MERGE"} {
		if revision == head || strings.HasPrefix(revision, head+"^") || strings.HasPrefix(revision, head+"~") || strings.HasPrefix(revision, head+":") {
			return revision, true
		}
	}
	return revision, false
}
