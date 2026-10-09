package git

import "testing"

func TestWorktreeRelativeRevision(t *testing.T) {
	for _, revision := range []string{"HEAD", "HEAD~1", "HEAD^", "HEAD^{commit}", "HEAD@{1}", "HEAD:path", "@", "@~2", "@^", "@{-1}", "@{yesterday}", "main@{1}", "ORIG_HEAD", "FETCH_HEAD", "MERGE_HEAD^", "-"} {
		got, relative := WorktreeRelativeRevision(revision)
		if !relative {
			t.Errorf("not relative: %s", revision)
		}
		want := revision
		if revision == "-" {
			want = "@{-1}"
		}
		if got != want {
			t.Errorf("%s -> %s", revision, got)
		}
	}
	for _, revision := range []string{"", "^", "main", "main~1", "refs/heads/HEAD", "refs/tags/@", "HEAD-feature", "v1.0", "0123456789abcdef"} {
		if got, relative := WorktreeRelativeRevision(revision); relative || got != revision {
			t.Errorf("misclassified %s -> %s %v", revision, got, relative)
		}
	}
}
