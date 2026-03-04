package internal

import (
	"fmt"
	"os/exec"
	"strings"
)

type Repo struct {
	Dir string
}

func NewRepo(dir string) *Repo {
	return &Repo{Dir: dir}
}

func (r *Repo) git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
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
