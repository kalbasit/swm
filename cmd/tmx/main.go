package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/kalbasit/tmx"
)

var (
	// codePath is the path to the code folder
	codePath string

	// cachePath is the absolute path of the cache file
	cachePath string

	// killPane if true will kill the tmux pane currently running the tmx
	killPane bool

	// profile select the profile, defaults to the current profile
	profile string

	// select the workspace, defaults to the current workspace
	workspace string

	// fzfPath is the PATH to the fzf executable
	fzfPath string

	// tmuxPath is the PATH to the tmux executable
	tmuxPath string
)

func init() {
	flag.StringVar(&codePath, "code", path.Join(os.Getenv("HOME"), "code"), "The code path to scan")
	flag.StringVar(&cachePath, "cache", path.Join(os.Getenv("HOME"), ".cache", "tmx.json"), "The path of the file used to store the cache")
	flag.BoolVar(&killPane, "kill-pane", false, "kill the pane after the session has been switched; only relevant within TMUX")
	flag.StringVar(&profile, "profile", os.Getenv("ACTIVE_PROFILE"), "the name of the profile")
	flag.StringVar(&workspace, "workspace", parseWorkspace(), "the name of the workspace")

	var err error
	fzfPath, err = exec.LookPath("fzf")
	if err != nil {
		log.Fatalf("error looking up the fzf executable, is it installed? %s", err)
	}
	tmuxPath, err = exec.LookPath("tmux")
	if err != nil {
		log.Fatalf("error looking up the tmux executable, is it installed? %s", err)
	}
}

func main() {
	// parse the flags
	flag.Parse()
	// start the TMUX server
	cmd := exec.Command(tmuxPath, "-L", workspace, "start-server")
	if err := cmd.Run(); err != nil {
		log.Fatalf("error running %q: %s", fmt.Sprintf("%s -L %s start-server", tmuxPath, workspace), err)
	}
	// load the code
	c, err := loadCode()
	if err != nil {
		log.Fatalf("error loading the code, did you run tmxrc?: %s", err)
	}
	// select the session using fzf
	sessionName, err := withFilter(fzfPath, func(stdin io.WriteCloser) {
		for _, sess := range getSessionNames(c) {
			io.WriteString(stdin, sess)
			io.WriteString(stdin, "\n")
		}
	})
	if err != nil {
		log.Fatalf("error filtering the session: %s", err)
	}
	// start a TMUX session (or attach to an existing one)
	tmuxStart(c, sessionName)
}

func tmuxStart(c *tmx.Code, sessions []string) {
	// loop over the selected session to start, only the last one will be attached
	for i, sessionName := range sessions {
		// find the project for this session
		// TODO: if the session was not found but we do have one that match in the
		// base profile, then we must create a new one first.
		project, err := c.FindProjectBySessionName(sessionName)
		if err != nil {
			log.Fatalf("session %q not found", sessionName)
		}
		// run tmux ls -F #{session_name} to find out if we have one running already
		cmd := exec.Command(tmuxPath, "-L", workspace, "list-sessions", "-F", "#{session_name}")
		result, err := cmd.Output()
		if err != nil || !regexp.MustCompile("^"+sessionName+"$").Match(result) {
			// session does not exist, we should start it
			for _, args := range [][]string{
				// start the session
				{"-L", workspace, "new-session", "-d", "-s", sessionName},
				// set the active profile
				{"-L", workspace, "set-environment", "-t", sessionName, "ACTIVE_PROFILE", profile},
				// start a new shell on window 1
				{"-L", workspace, "new-window", "-t", sessionName + ":1"},
				// start vim in the first window
				{"-L", workspace, "send-keys", "-t", sessionName + ":0", "clear; vim", "Enter"},
			} {
				cmd := exec.Command(tmuxPath, args...)
				cmd.Dir = project.Path()
				if err := cmd.Run(); err != nil {
					log.Fatalf("error running tmux with args %v: %s", args, err)
				}
			}
		}
		// attach the last session
		if i == len(sessions)-1 {
			fmt.Printf("must attach now\n")
		}
	}
}

// getSessionNames returns all the sessions under the profile (if profile is
// non empty) and for the current workspace (if not empty). The base workspaces
// will always be returned
// TODO: move this to code as a helper
func getSessionNames(c *tmx.Code) []string {
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
	w := p.Workspaces[workspace]
	if w != nil {
		return append(w.SessionNames(), p.Workspaces["base"].SessionNames())
	}
	return w.SessionNames()
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

func loadCode() (*tmx.Code, error) {
	// open the file
	f, err := os.Open(cachePath)
	if err != nil {
		return nil, err
	}
	// decode it
	var c tmx.Code
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func parseWorkspace() string {
	// TODO: parse os.Getenv("TMUX")
	return "base"
}
