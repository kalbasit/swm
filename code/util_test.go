package code

import "github.com/spf13/afero"

func prepareFilesystem(test string) {
	createProjects(test)
}

func createProjects(name string) {
	// projects outside GOPATH
	AppFs.MkdirAll("/code/"+name+"/invalid/repo/.git", 0755)

	// projects under base
	AppFs.MkdirAll("/code/"+name+"/base/src/github.com/kalbasit/swm/.git", 0755)
	AppFs.MkdirAll("/code/"+name+"/base/src/github.com/kalbasit/dotfiles/.git", 0755)

	// projects under STORY-123
	AppFs.MkdirAll("/code/"+name+"/stories/STORY-123/src/github.com/kalbasit/dotfiles", 0755)
	afero.WriteFile(AppFs, "/code/"+name+"/stories/STORY-123/src/github.com/kalbasit/dotfiles/.git", []byte(
		"gitdir: /code/"+name+"/base/src/github.com/kalbasit/.git/worktrees/dotfiles",
	), 0644)

	// projects ignored

	// projects outside GOPATH
	AppFs.MkdirAll("/code/.snapshots/"+name+"/invalid/repo/.git", 0755)

	// projects under base
	AppFs.MkdirAll("/code/.snapshots/"+name+"/base/src/github.com/kalbasit/swm/.git", 0755)
	AppFs.MkdirAll("/code/.snapshots/"+name+"/base/src/github.com/kalbasit/dotfiles/.git", 0755)

	// projects under STORY-123
	AppFs.MkdirAll("/code/.snapshots/"+name+"/stories/STORY-123/src/github.com/kalbasit/dotfiles", 0755)
	afero.WriteFile(AppFs, "/code/.snapshots/"+name+"/stories/STORY-123/src/github.com/kalbasit/dotfiles/.git", []byte(
		"gitdir: /code/"+name+"/base/src/github.com/kalbasit/.git/worktrees/dotfiles",
	), 0644)
}
