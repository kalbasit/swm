package tmux

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"syscall"

	"github.com/kalbasit/swm/ifaces"
	"github.com/kalbasit/swm/story"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	dotChar   = "\u2022"
	colonChar = "\uFF1A"
)

var (
	// ErrProjectNotFoundForGivenSessionName is returned by SwitchClient if the
	// selected session (via fzf currently) was not found. This usually means
	// that fzf output was not one of the input.
	ErrProjectNotFoundForGivenSessionName = errors.New("project not found for the given session name")

	// ErrVimSessionFound is returned by KillServer(closeVim bool) if a vim was
	// found running on the server and closeVim is false
	ErrVimSessionFound = errors.New("vim was found, cannot exit server to avoid data loss")

	// tmuxPath is the PATH to the tmux executable
	tmuxPath string

	// fzfPath is the PATH to the fzf executable
	fzfPath string

	// path path to the ps tool
	psPath string

	psGrepRegexp = regexp.MustCompile(`(?m)^(?:[^TXZ ])+ +(?:((?:\S)+/))?g?(n?vim?x?)$`)
)

func init() {
	var err error

	tmuxPath, err = exec.LookPath("tmux")
	if err != nil {
		log.Fatal().Msgf("error looking up the tmux executable, is it installed? %s", err)
	}

	fzfPath, err = exec.LookPath("fzf")
	if err != nil {
		log.Fatal().Msgf("error looking up the fzf executable, is it installed? %s", err)
	}

	psPath, err = exec.LookPath("ps")
	if err != nil {
		log.Fatal().Msgf("error looking up the ps executable, is it installed? %s", err)
	}
}

// Manager represents a TMUX manager
type Manager struct {
	code  ifaces.Code
	story ifaces.Story
}

// New returns a new tmux manager
func New(c ifaces.Code, storyName string) (*Manager, error) {
	s, err := story.Load(storyName)
	if err != nil {
		return nil, errors.Wrap(err, "error loading the story")
	}

	return &Manager{code: c, story: s}, nil
}

func (t *Manager) KillServer(closeVim bool) error {
	// find out if we have any running vim session and if we do, act on closeVim
	targets, err := t.getTargetsRunningVim()
	if err != nil {
		return err
	}
	if len(targets) > 0 {
		if !closeVim {
			return ErrVimSessionFound
		}
		// ask vim to exit
		if err := t.VimExit(); err != nil {
			return err
		}
	}

	return exec.Command(tmuxPath, "-L", t.socketName(), "kill-server").Run()
}

// socketName returns the session name
func (t *Manager) socketName() string {
	return strings.Replace(fmt.Sprintf("swm-%s", t.story.GetName()), "/", "_", -1)
}

// withFilter filters input using fzf
func (t *Manager) withFilter(input func(in io.WriteCloser)) (string, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	cmd := exec.Command(shell, "-c", fzfPath)
	cmd.Stderr = os.Stderr
	in, _ := cmd.StdinPipe()
	go func() {
		input(in)
		in.Close()
	}()
	result, err := cmd.Output()
	if err != nil {
		return "", err
	}
	res := strings.Split(string(result), "\n")
	if res[len(res)-1] == "" {
		res = res[0 : len(res)-1]
	}
	return res[0], nil
}

// SwitchClient switches the TMUX to a different client
func (t *Manager) SwitchClient(killPane bool) error {
	// get all the sessions
	sessionNameProjects, err := t.getSessionNameProjects()
	if err != nil {
		return err
	}
	// select the session using fzf
	sessionName, err := t.withFilter(func(stdin io.WriteCloser) {
		for name := range sessionNameProjects {
			io.WriteString(stdin, name)
			io.WriteString(stdin, "\n")
		}
	})
	if err != nil {
		return err
	}
	// get the project for the selected name
	project, ok := sessionNameProjects[sessionName]
	if !ok {
		return ErrProjectNotFoundForGivenSessionName
	}
	// make sure the project exists on disk
	if err := project.CreateStory(t.story); err != nil {
		return err
	}
	// run tmux has-session -t sessionName to check if session already exists
	if err := exec.Command(tmuxPath, "-L", t.socketName(), "has-session", "-t="+sessionName).Run(); err != nil {
		// session does not exist, we should start it
		for _, args := range [][]string{
			// start the session
			{"-L", t.socketName(), "new-session", "-c", project.Path(t.story), "-d", "-s", sessionName},
			// set the active story name
			{"-L", t.socketName(), "set-environment", "-t", sessionName, "SWM_STORY_NAME", t.story.GetName()},
			// set the branch name
			{"-L", t.socketName(), "set-environment", "-t", sessionName, "SWM_STORY_BRANCH_NAME", t.story.GetBranchName()},
			// start a new shell on window 1
			{"-L", t.socketName(), "new-window", "-c", project.Path(t.story), "-t", sessionName + ":1"},
			// start vim in the first window
			{"-L", t.socketName(), "send-keys", "-t", sessionName + ":0", "type vim_ready &>/dev/null && vim_ready; clear; vim", "Enter"},
		} {
			cmd := exec.Command(tmuxPath, args...)
			cmd.Dir = project.Path(t.story)
			// set the environment to current environment, change only ACTIVE_PROFILE, ACTIVE_STORY  and GOPATH
			cmd.Env = func() []string {
				res := []string{
					fmt.Sprintf("SWM_STORY_NAME=%s", t.story.GetName()),
					fmt.Sprintf("SWM_STORY_BRANCH_NAME=%s", t.story.GetBranchName()),
				}
				for _, v := range os.Environ() {
					if k := strings.Split(v, "=")[0]; k != "SWM_STORY_NAME" && k != "TMUX" {
						res = append(res, v)
					}
				}

				return res
			}()
			// run the command now
			if err := cmd.Run(); err != nil {
				log.Fatal().Strs("args", args).Err(err).Msg("error running the tmux comand")
			}
		}
	}
	// attach the session now
	if tmuxSocketPath := os.Getenv("TMUX"); tmuxSocketPath != "" {
		// kill the pane once attached
		if killPane {
			defer func() {
				exec.Command(tmuxPath, "-L", strings.Split(path.Base(tmuxSocketPath), ",")[0], "kill-pane").Run()
			}()
		}
		return exec.Command(tmuxPath, "-L", t.socketName(), "switch-client", "-t", sessionName).Run()
	}
	// NOTE: the following Exec calls kernel's execve, which means that this will
	// never return and the current swm binary will be replaced by tmux. This is
	// precisely what we want as there is no sence in keeping swm running after
	// attaching to a tmux session.
	return syscall.Exec(tmuxPath, []string{"tmux", "-L" + t.socketName(), "attach", "-t" + sessionName}, os.Environ())
}

// getSessionNameProjects returns a map of a project session name to the project
func (t *Manager) getSessionNameProjects() (map[string]ifaces.Project, error) {
	sessionNameProjects := make(map[string]ifaces.Project)

	// loop over all projects and get the session name
	for _, prj := range t.code.Projects() {
		// assign it to the map
		sessionNameProjects[sanitizeSessionName(prj.String())] = prj
	}

	return sessionNameProjects, nil
}

func sanitizeSessionName(name string) string {
	name = strings.Replace(name, ".", dotChar, -1)
	name = strings.Replace(name, ":", colonChar, -1)

	return name
}

func (t *Manager) VimExit() error {
	// get the list of panes that are running vim
	targets, err := t.getTargetsRunningVim()
	if err != nil {
		return err
	}
	log.Debug().Msgf("found the following tmux targets running vim: %v", targets)
	// iterate over all the panes that has vim, and ask it to close itself
	for _, target := range targets {
		// Send the escape key, in the case we are in a vim like program. This is
		// repeated because the send-key command is not waiting for vim to complete
		// its action.
		// Credit: https://gist.github.com/debugish/2773454
		for i := 0; i < 25; i++ {
			if err := exec.Command(tmuxPath, "-L", t.socketName(), "send-keys", "-t", target, "C-[").Run(); err != nil {
				return err
			}
		}
		// ask Vim to exit
		if err := exec.Command(tmuxPath, "-L", t.socketName(), "send-keys", "-t", target, ":", "x", "a", "C-m").Run(); err != nil {
			return err
		}
	}

	return nil
}

func (t *Manager) getTargetsRunningVim() ([]string, error) {
	var targets []string

	// get the list of sessions
	var sessionNames []string
	{
		cmd := exec.Command(tmuxPath, "-L", t.socketName(), "list-sessions", "-F", "#{session_name}")
		out, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(out), "\n") {
			if line != "" {
				sessionNames = append(sessionNames, line)
			}
		}
	}
	log.Debug().Msgf("found the following tmux sessions: %v", sessionNames)
	// iterate over the list of sessions, and for each session iterate over the
	// list of windows, then over the panes and check what they are running
	for _, sessionName := range sessionNames {
		// get the list of windows for this session
		var windowIDs []string
		{
			cmd := exec.Command(tmuxPath, "-L", t.socketName(), "list-windows", "-t", sessionName, "-F", "#I")
			out, err := cmd.Output()
			if err != nil {
				return nil, err
			}
			for _, line := range strings.Split(string(out), "\n") {
				if line != "" {
					windowIDs = append(windowIDs, line)
				}
			}
		}
		// iterate over the list of windows, get the list of panes, and for each pane
		// find out if it is running vim, nvim then will build targets
		for _, windowID := range windowIDs {
			// build the map of pane to tty
			paneTTY := make(map[string]string)
			{
				cmd := exec.Command(tmuxPath, "-L", t.socketName(), "list-panes", "-t", sessionName+":"+windowID, "-F", "#P@#{pane_tty}")
				out, err := cmd.Output()
				if err != nil {
					return nil, err
				}
				paneTTYs := strings.Split(string(out), "\n")
				for _, paneTTYStr := range paneTTYs {
					paneTTYArr := strings.Split(paneTTYStr, "@")
					if len(paneTTYArr) < 2 {
						continue
					}
					paneTTY[paneTTYArr[0]] = paneTTYArr[1]
				}
			}
			// now iterate over the pane/tty, check what is running on that TTY
			for paneID, ttyPath := range paneTTY {
				// TODO: replace this sub-exec with real /proc parsing library for processes
				// see https://github.com/mitchellh/go-ps/blob/master/process_linux.go
				cmd := exec.Command(psPath, "-o", "state=", "-o", "comm=", "-t", ttyPath)
				out, err := cmd.Output()
				if err != nil {
					return nil, err
				}
				// test the output against the psGrepRegexp
				if psGrepRegexp.Match(out) {
					targets = append(targets, fmt.Sprintf("%s:%s.%s", sessionName, windowID, paneID))
				}
			}
		}

	}

	return targets, nil
}
