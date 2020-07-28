package testhelper

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	// gitPath is the PATH to the git binary
	gitPath string
)

func init() {
	var err error
	gitPath, err = exec.LookPath("git")
	if err != nil {
		log.Fatal().Msgf("error looking up the git executable, is it installed? %s", err)
	}
}

// CreateProjects creates projects in a filesystem to prepare a coder
func CreateProjects(t *testing.T, basePath string) error {
	// initialize repositories that should make it as part of the scan
	for _, importPath := range []string{"github.com/owner1/repo1", "github.com/owner2/repo2", "github.com/owner3/repo3"} {
		if err := gitInitRepo(path.Join(basePath, "repositories", importPath)); err != nil {
			return err
		}
	}

	// initialize repositories that should not make it as part of the scan
	for _, importPath := range []string{"github.com/owner4/repo4", "github.com/owner5/repo5", "github.com/owner6/repo6"} {
		if err := gitInitRepo(path.Join(basePath, ".snapshots", importPath)); err != nil {
			return err
		}
	}

	return nil
}

func gitInitRepo(p string) error {
	if _, err := os.Stat(p); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		os.MkdirAll(p, 0755)
	}

	cmd := exec.Command(gitPath, "init")
	cmd.Dir = p
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "error creating the Git repository")
	}

	f, err := os.Create(path.Join(p, "some-file"))
	if err != nil {
		return errors.Wrap(err, "error creating a file in the Git repository")
	}

	if _, err := f.WriteString("Hello, World!"); err != nil {
		return errors.Wrap(err, "error writing content in the test file")
	}

	cmd = exec.Command(gitPath, "add", "-A", ".")
	cmd.Dir = p
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "error adding files to the index")
	}

	cmd = exec.Command(gitPath, "commit", "--no-verify", "--no-gpg-sign", "--message", "initial import")
	cmd.Dir = p
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "error adding files to the index")
	}

	return nil
}
