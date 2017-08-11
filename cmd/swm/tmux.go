package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/kalbasit/swm/code"
)

var (
	// codePath is the path to the code folder
	codePath string

	// killPane if true will kill the tmux pane currently running the swm
	killPane bool

	// profile select the profile, defaults to the current profile
	profile string

	// socketPath is the full path of the workspace
	socketPath string

	// select the workspace, defaults to the current workspace
	workspace string

	// fzfPath is the PATH to the fzf executable
	fzfPath string

	// tmuxPath is the PATH to the tmux executable
	tmuxPath string

	// gitPath is the PATH to the git binary
	gitPath string
)

func init() {
	flag.StringVar(&codePath, "code", path.Join(os.Getenv("HOME"), "code"), "The code path to scan")
	flag.BoolVar(&killPane, "kill-pane", false, "kill the pane after the session has been switched; only relevant within TMUX")
	flag.StringVar(&profile, "profile", os.Getenv("ACTIVE_PROFILE"), "the name of the profile")
	flag.StringVar(&workspace, "workspace", "", "the name of the workspace")
	flag.StringVar(&socketPath, "socket-path", os.Getenv("TMUX"), "the path to the socket name")

	var err error
	fzfPath, err = exec.LookPath("fzf")
	if err != nil {
		log.Fatalf("error looking up the fzf executable, is it installed? %s", err)
	}
	tmuxPath, err = exec.LookPath("tmux")
	if err != nil {
		log.Fatalf("error looking up the tmux executable, is it installed? %s", err)
	}
	gitPath, err = exec.LookPath("git")
	if err != nil {
		log.Fatalf("error looking up the git executable, is it installed? %s", err)
	}
}

// func main() {
// 	// parse the flags
// 	flag.Parse()
// 	// parse the socketPath as workspace if needed
// 	if workspace == "" && socketPath != "" {
// 		workspace = strings.Split(path.Base(socketPath), ",")[0]
// 	}
// 	// create the code and load/scan
// 	c := code.New(codePath, regexp.MustCompile("^.snapshots$"))
// 	if err := c.LoadOrScan(); err != nil {
// 		log.Fatalf("error loading the code: %s", err)
// 	}
// 	// select the session using fzf
// 	sessionName, err := withFilter(fzfPath, func(stdin io.WriteCloser) {
// 		for _, sess := range getSessionNames(c) {
// 			io.WriteString(stdin, sess)
// 			io.WriteString(stdin, "\n")
// 		}
// 	})
// 	if err != nil {
// 		log.Fatalf("error filtering the session: %s", err)
// 	}
// 	// start a TMUX session (or attach to an existing one)
// 	if err := tmuxStart(c, sessionName); err != nil {
// 		log.Fatalf("error attaching the session %q: %s", sessionName, err)
// 	}
// }

func tmuxStart(c *code.Code, sessions []string) error {
	// loop over the selected session to start, only the last one will be attached
	for i, sessionName := range sessions {
		// load the project
		project, err := findProject(c, sessionName)
		if err != nil {
			return err
		}
		// run tmux has-session -t sessionName to check if session already exists
		if err := exec.Command(tmuxPath, "-L", project.StoryName, "has-session", "-t", sessionName).Run(); err != nil {
			// session does not exist, we should start it
			for _, args := range [][]string{
				// start the session
				{"-L", project.StoryName, "new-session", "-d", "-s", sessionName},
				// set the active profile
				{"-L", project.StoryName, "set-environment", "-t", sessionName, "ACTIVE_PROFILE", project.ProfileName},
				// set the new GOPATH
				{"-L", project.StoryName, "set-environment", "-t", sessionName, "GOPATH", path.Join(c.Path, project.ProfileName, project.StoryName)},
				// start a new shell on window 1
				{"-L", project.StoryName, "new-window", "-t", sessionName + ":1"},
				// start vim in the first window
				{"-L", project.StoryName, "send-keys", "-t", sessionName + ":0", "clear; vim", "Enter"},
			} {
				cmd := exec.Command(tmuxPath, args...)
				cmd.Dir = project.Path()
				// set the environment to current environment, change only ACTIVE_PROFILE and GOPATH
				cmd.Env = func() []string {
					res := []string{
						fmt.Sprintf("ACTIVE_PROFILE=%s", project.ProfileName),
						fmt.Sprintf("GOPATH=%s", path.Join(c.Path, project.ProfileName, project.StoryName)),
					}
					for _, v := range os.Environ() {
						if k := strings.Split(v, "=")[0]; k != "ACTIVE_PROFILE" && k != "GOPATH" && k != "TMUX" {
							res = append(res, v)
						}
					}

					return res
				}()
				// run the command now
				if err := cmd.Run(); err != nil {
					log.Fatalf("error running tmux with args %v: %s", args, err)
				}
			}
		}
		// attach the last session
		if i == len(sessions)-1 {
			if os.Getenv("TMUX") != "" {
				// kill the pane once attached
				defer func() {
					if killPane {
						exec.Command(tmuxPath, "-L", project.StoryName, "kill-pane").Run()
					}
				}()
				return exec.Command(tmuxPath, "-L", project.StoryName, "switch-client", "-t", sessionName).Run()
			}
			cmd := exec.Command(tmuxPath, "-L", project.StoryName, "attach", "-t", sessionName)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
	}

	return errors.New("no session selected")
}

func findProject(c *code.Code, name string) (*code.Project, error) {
	// find the project for this session
	p, err := c.FindProjectBySessionName(name)
	if err != nil {
		return nil, err
	}
	// if the returned project belongs to the base workspace and we are not on
	// a base workspace then we have to clone it, and update the cache.
	if workspace != "" && workspace != code.BaseStory && p.Base() {
		// deep clone p into p2 and change the workspace
		p2 := &code.Project{}
		*p2 = *p
		p2.StoryName = workspace
		// create the new worktree
		cmd := exec.Command(gitPath, "worktree", "add", "-b", workspace, p2.Path(), "master")
		cmd.Dir = p.Path()
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("error creating a new worktree: %s\n%s", err, string(out))
		}
		// re-scan and save
		c.Scan()
		if err := c.Save(); err != nil {
			return nil, fmt.Errorf("error saving the cache: %s", err)
		}

		p = p2
	}

	return p, nil
}

// getSessionNames returns all the sessions under the profile (if profile is
// non empty) and for the current workspace (if not empty). The base workspaces
// will always be returned
// TODO: move this to code as a helper
// TODO: should not return projects in both base and workspace
func getSessionNames(c *code.Code) []string {
	if profile == "" {
		return c.SessionNames()
	}
	p := c.Profiles[profile]
	if p == nil {
		log.Fatalf("profile %q not found", profile)
	}
	if workspace == "" {
		return p.SessionNames()
	}
	w := p.Stories[workspace]
	if w != nil {
		return append(w.SessionNames(), p.BaseStory().SessionNames()...)
	}
	return p.BaseStory().SessionNames()
}

// withFilter filters input using fzf
func withFilter(command string, input func(in io.WriteCloser)) ([]string, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	cmd := exec.Command(shell, "-c", command)
	cmd.Stderr = os.Stderr
	in, _ := cmd.StdinPipe()
	go func() {
		input(in)
		in.Close()
	}()
	result, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	res := strings.Split(string(result), "\n")
	if res[len(res)-1] == "" {
		res = res[0 : len(res)-1]
	}
	return res, nil
}
