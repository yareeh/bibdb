package internal

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Repo struct {
	Dir      string
	pat      string
	unsigned bool
}

func NewRepo(dir string) *Repo {
	return &Repo{Dir: dir}
}

// NewRepoFor builds the repo of a configured backend, with its credentials: the
// backend's pat, else the contents of pat_file, else $BIBDB_PAT.
func NewRepoFor(b Backend) *Repo {
	r := &Repo{Dir: b.Path, pat: strings.TrimSpace(b.Pat), unsigned: b.Unsigned}
	if r.pat == "" && b.PatFile != "" {
		if data, err := os.ReadFile(b.PatFile); err == nil {
			r.pat = strings.TrimSpace(string(data))
		}
	}
	if r.pat == "" {
		r.pat = strings.TrimSpace(os.Getenv("BIBDB_PAT"))
	}
	return r
}

// gitEnv is the extra environment of every git call, as GIT_CONFIG_* pairs so the
// token never shows up in the process list or in .git/config.
func (r *Repo) gitEnv() []string {
	var kv [][2]string
	if r.pat != "" {
		auth := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + r.pat))
		kv = append(kv,
			[2]string{"http.https://github.com/.extraheader", "Authorization: Basic " + auth},
			[2]string{"url.https://github.com/.insteadOf", "git@github.com:"},
			[2]string{"url.https://github.com/.insteadOf", "ssh://git@github.com/"},
		)
	}
	if r.unsigned {
		kv = append(kv, [2]string{"commit.gpgsign", "false"})
	}
	if len(kv) == 0 {
		return nil
	}
	env := []string{fmt.Sprintf("GIT_CONFIG_COUNT=%d", len(kv))}
	for i, p := range kv {
		env = append(env, fmt.Sprintf("GIT_CONFIG_KEY_%d=%s", i, p[0]), fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", i, p[1]))
	}
	if r.pat != "" {
		env = append(env, "GIT_TERMINAL_PROMPT=0")
	}
	return env
}

func (r *Repo) git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	if env := r.gitEnv(); env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *Repo) IsRepo() bool {
	_, err := r.git("rev-parse", "--git-dir")
	return err == nil
}

func (r *Repo) Init() error {
	_, err := r.git("init")
	return err
}

func (r *Repo) HasRemote() bool {
	out, err := r.git("remote")
	return err == nil && strings.TrimSpace(out) != ""
}

func (r *Repo) Pull() error {
	if !r.HasRemote() {
		return nil
	}
	_, err := r.git("pull", "--rebase", "--autostash")
	return err
}

func (r *Repo) Add(paths ...string) error {
	args := append([]string{"add"}, paths...)
	_, err := r.git(args...)
	return err
}

func (r *Repo) Commit(message string) error {
	_, err := r.git("commit", "-m", message)
	return err
}

func (r *Repo) Push() error {
	if !r.HasRemote() {
		return nil
	}
	_, err := r.git("push")
	if err != nil {
		// Retry once after pull
		if pullErr := r.Pull(); pullErr != nil {
			return pullErr
		}
		_, err = r.git("push")
	}
	return err
}

// SyncMutation performs the full sync cycle for a mutation operation.
func (r *Repo) SyncMutation(paths []string, message string) error {
	if err := r.Pull(); err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	if err := r.Add(paths...); err != nil {
		return fmt.Errorf("add: %w", err)
	}
	if err := r.Commit(message); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	if err := r.Push(); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	return nil
}

// Sync performs a manual sync (pull + push).
func (r *Repo) Sync() error {
	if err := r.Pull(); err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	if err := r.Push(); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	return nil
}
