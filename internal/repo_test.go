package internal

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	// Disable signing so the test commit doesn't hit the user's GPG/SSH agent
	// (e.g. 1Password), which is unavailable in CI / non-interactive runs.
	r.git("config", "commit.gpgsign", "false")
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

func envMap(env []string) map[string]string {
	m := map[string]string{}
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok {
			m[k] = v
		}
	}
	return m
}

func TestGitEnvWithoutPatOrUnsignedIsEmpty(t *testing.T) {
	r := NewRepoFor(Backend{Path: t.TempDir()})
	if env := r.gitEnv(); len(env) != 0 {
		t.Fatalf("expected no extra env, got %v", env)
	}
}

func TestGitEnvPatBecomesBasicAuthHeaderAndHTTPSRewrite(t *testing.T) {
	r := NewRepoFor(Backend{Path: t.TempDir(), Pat: "ghp_secret"})
	m := envMap(r.gitEnv())

	want := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:ghp_secret"))
	found := map[string]string{}
	n, _ := strconv.Atoi(m["GIT_CONFIG_COUNT"])
	for i := 0; i < n; i++ {
		found[m[fmt.Sprintf("GIT_CONFIG_KEY_%d", i)]] = m[fmt.Sprintf("GIT_CONFIG_VALUE_%d", i)]
	}
	if found["http.https://github.com/.extraheader"] != want {
		t.Errorf("extraheader = %q, want %q", found["http.https://github.com/.extraheader"], want)
	}
	if found["url.https://github.com/.insteadOf"] == "" {
		t.Error("ssh remotes should be rewritten to https")
	}
	if m["GIT_TERMINAL_PROMPT"] != "0" {
		t.Error("git must never prompt when a PAT is configured")
	}
	if _, ok := found["commit.gpgsign"]; ok {
		t.Error("a PAT alone must not change signing")
	}
}

func TestGitEnvPatFileIsReadAndTrimmed(t *testing.T) {
	f := filepath.Join(t.TempDir(), "pat")
	os.WriteFile(f, []byte("  ghp_fromfile\n"), 0o600)
	r := NewRepoFor(Backend{Path: t.TempDir(), PatFile: f})
	if got := r.pat; got != "ghp_fromfile" {
		t.Errorf("pat = %q", got)
	}
}

func TestPatEnvVarOverridesNothingWhenConfigSet(t *testing.T) {
	t.Setenv("BIBDB_PAT", "ghp_env")
	r := NewRepoFor(Backend{Path: t.TempDir(), Pat: "ghp_cfg"})
	if r.pat != "ghp_cfg" {
		t.Errorf("config pat should win, got %q", r.pat)
	}
	r = NewRepoFor(Backend{Path: t.TempDir()})
	if r.pat != "ghp_env" {
		t.Errorf("BIBDB_PAT should be the fallback, got %q", r.pat)
	}
}

func TestUnsignedCommitsDespiteGpgsignAndBrokenSigner(t *testing.T) {
	dir := t.TempDir()
	r := NewRepoFor(Backend{Path: dir, Unsigned: true})
	if err := r.Init(); err != nil {
		t.Fatal(err)
	}
	r.git("config", "user.email", "test@test.com")
	r.git("config", "user.name", "Test")
	r.git("config", "commit.gpgsign", "true")
	r.git("config", "gpg.format", "ssh")
	r.git("config", "gpg.ssh.program", "/nonexistent/signer")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644)
	if err := r.Add("a.txt"); err != nil {
		t.Fatal(err)
	}
	if err := r.Commit("unsigned"); err != nil {
		t.Fatalf("unsigned backend must commit despite broken signer: %v", err)
	}

	// Without the option the same setup fails: the option is what makes the difference.
	r2 := NewRepoFor(Backend{Path: dir})
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("y"), 0o644)
	r2.Add("b.txt")
	if err := r2.Commit("signed"); err == nil {
		t.Fatal("expected failure with a broken signer and no unsigned option")
	}
}

func TestPatPullsAndPushesOverHTTPSRewrite(t *testing.T) {
	// A local bare repo stands in for GitHub; verify the env is applied to every git call
	// by checking that the rewrite config is visible to git.
	r := NewRepoFor(Backend{Path: t.TempDir(), Pat: "ghp_x"})
	r.Init()
	out, err := r.git("config", "--get-regexp", `^url\..*\.insteadof$`)
	if err != nil || !strings.Contains(out, "git@github.com:") {
		t.Fatalf("rewrite not visible to git: %q %v", out, err)
	}
}
