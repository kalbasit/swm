.PHONY: doc
update-doc:
	rm -rf doc
	env SWM_STORY_NAME="" SWM_STORY_BRANCH_NAME="" go run main.go gen-doc markdown --path ./doc
