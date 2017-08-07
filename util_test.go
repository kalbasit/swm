package tmx

func prepareFilesystem() {
	// TestWorkspaceScan
	AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceScan/base/src/github.com/kalbasit/tmx/.git", 0755)
	AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceScan/base/src/github.com/kalbasit/dotfiles/.git", 0755)
	AppFs.MkdirAll("/home/kalbasit/code/TestWorkspaceScan/STORY-123/src/github.com/kalbasit/private/.git", 0755)

	// TestProfileScan
	AppFs.MkdirAll("/home/kalbasit/code/TestProfileScan/base/src/github.com/kalbasit/tmx/.git", 0755)
	AppFs.MkdirAll("/home/kalbasit/code/TestProfileScan/base/src/github.com/kalbasit/dotfiles/.git", 0755)
	AppFs.MkdirAll("/home/kalbasit/code/TestProfileScan/STORY-123/src/github.com/kalbasit/private/.git", 0755)

	// TestCodeScan
	AppFs.MkdirAll("/home/kalbasit/code/TestCodeScan/base/src/github.com/kalbasit/tmx/.git", 0755)
	AppFs.MkdirAll("/home/kalbasit/code/TestCodeScan/base/src/github.com/kalbasit/dotfiles/.git", 0755)
	AppFs.MkdirAll("/home/kalbasit/code/TestCodeScan/STORY-123/src/github.com/kalbasit/private/.git", 0755)
	AppFs.MkdirAll("/home/kalbasit/code/.snapshots/STORY-123/src/github.com/kalbasit/private/.git", 0755)
}
