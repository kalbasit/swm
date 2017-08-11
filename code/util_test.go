package code

import "github.com/spf13/afero"

func prepareFilesystem(test string) {
	switch test {
	case "TestWorkspaceScan":
		AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceScan/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceScan/base/src/github.com/kalbasit/dotfiles/.git", 0755)

		AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceScan/STORY-123/src/github.com/kalbasit/dotfiles", 0755)
		afero.WriteFile(AppFs, "/home/kalbasit/code/TestWorkspaceScan/STORY-123/src/github.com/kalbasit/dotfiles/.git", []byte(
			"gitdir: /home/kalbasit/code/TestWorkspaceScan/base/src/github.com/kalbasit/.git/worktrees/dotfiles",
		), 0644)

		AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceScan/invalid/repo/.git", 0755)
	case "TestProfileScan":
		AppFs.MkdirAll("/home/kalbasit/code/TestProfileScan/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestProfileScan/base/src/github.com/kalbasit/dotfiles/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestProfileScan/STORY-123/src/github.com/kalbasit/private/.git", 0755)
	case "TestSave":
		AppFs.MkdirAll("/home/kalbasit/code/TestSave/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestSave/base/src/github.com/kalbasit/dotfiles/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestSave/STORY-123/src/github.com/kalbasit/private/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/.snapshots/STORY-123/src/github.com/kalbasit/private/.git", 0755)
	case "TestLoad":
		AppFs.MkdirAll("/home/kalbasit/code/TestLoad/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestLoad/base/src/github.com/kalbasit/dotfiles/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestLoad/STORY-123/src/github.com/kalbasit/private/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/.snapshots/STORY-123/src/github.com/kalbasit/private/.git", 0755)
	case "TestLoadOrScan":
		AppFs.MkdirAll("/home/kalbasit/code/TestLoadOrScan/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestLoadOrScan/base/src/github.com/kalbasit/dotfiles/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestLoadOrScan/STORY-123/src/github.com/kalbasit/private/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/.snapshots/STORY-123/src/github.com/kalbasit/private/.git", 0755)
	case "TestCodeScan":
		AppFs.MkdirAll("/home/kalbasit/code/TestCodeScan/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestCodeScan/base/src/github.com/kalbasit/dotfiles/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestCodeScan/STORY-123/src/github.com/kalbasit/private/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/.snapshots/STORY-123/src/github.com/kalbasit/private/.git", 0755)
	case "TestFindProjectBySessionName":
		AppFs.MkdirAll("/home/kalbasit/code/TestFindProjectBySessionName/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestFindProjectBySessionName/base/src/github.com/kalbasit/dotfiles/.git", 0755)

		AppFs.MkdirAll("/home/kalbasit/code/TestFindProjectBySessionName/STORY-123/src/github.com/kalbasit/tmx", 0755)
		afero.WriteFile(AppFs, "/home/kalbasit/code/TestFindProjectBySessionName/STORY-123/src/github.com/kalbasit/tmx/.git", []byte(
			"gitdir: /home/kalbasit/code/TestFindProjectBySessionName/base/src/github.com/kalbasit/.git/worktrees/tmx",
		), 0644)
	case "TestWorkspaceFindProjectBySessionName":
		AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceFindProjectBySessionName/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceFindProjectBySessionName/base/src/github.com/kalbasit/dotfiles/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceFindProjectBySessionName/STORY-123/src/github.com/kalbasit/private/.git", 0755)
	case "TestCodeSessionNames":
		AppFs.MkdirAll("/home/kalbasit/code/TestCodeSessionNames/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestCodeSessionNames/base/src/github.com/kalbasit/dotfiles/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestCodeSessionNames/STORY-123/src/github.com/kalbasit/private/.git", 0755)
	case "TestProfileSessionNames":
		AppFs.MkdirAll("/home/kalbasit/code/TestProfileSessionNames/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestProfileSessionNames/base/src/github.com/kalbasit/dotfiles/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestProfileSessionNames/STORY-123/src/github.com/kalbasit/private/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestOtherProfileSessionNames/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestOtherProfileSessionNames/base/src/github.com/kalbasit/dotfiles/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestOtherProfileSessionNames/STORY-123/src/github.com/kalbasit/private/.git", 0755)
	case "TestWorkspaceSessionNames":
		AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceSessionNames/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceSessionNames/base/src/github.com/kalbasit/dotfiles/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceSessionNames/STORY-123/src/github.com/kalbasit/private/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestOtherWorkspaceSessionNames/base/src/github.com/kalbasit/tmx/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestOtherWorkspaceSessionNames/base/src/github.com/kalbasit/dotfiles/.git", 0755)
		AppFs.MkdirAll("/home/kalbasit/code/TestOtherWorkspaceSessionNames/STORY-123/src/github.com/kalbasit/private/.git", 0755)
	}
}
