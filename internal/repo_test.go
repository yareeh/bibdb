package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func initTestRepo(t *testing.T) *Repo {
	t.Helper()
	dir := t.TempDir()
	r := NewRepo(dir)
	if err := r.Init(); err != nil {
		t.Fatal(err)
	}
	r.git("config", "user.email", "test@test.com")
	r.git("config", "user.name", "Test")
	return r
}

func TestRepoInitAndCommit(t *testing.T) {
	r := initTestRepo(t)
	if !r.IsRepo() {
		t.Fatal("expected repo")
	}

	// Create a file and commit
	os.WriteFile(filepath.Join(r.Dir, "test.txt"), []byte("hello"), 0o644)
	if err := r.Add("test.txt"); err != nil {
		t.Fatal(err)
	}
	if err := r.Commit("initial"); err != nil {
		t.Fatal(err)
	}
}

func TestRepoHasNoRemote(t *testing.T) {
	r := initTestRepo(t)

	if r.HasRemote() {
		t.Error("should not have remote")
	}

	// Push/Pull should be no-ops without remote
	if err := r.Push(); err != nil {
		t.Errorf("push without remote should be no-op: %v", err)
	}
	if err := r.Pull(); err != nil {
		t.Errorf("pull without remote should be no-op: %v", err)
	}
}
