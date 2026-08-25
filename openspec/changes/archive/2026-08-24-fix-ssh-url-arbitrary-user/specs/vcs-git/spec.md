## MODIFIED Requirements

### Requirement: ParseRemoteURL returns ProjectID
`vcs-git` SHALL implement `VCS.ParseRemoteURL(url)` by parsing the remote URL into a `ProjectID(host, segments[])`. It MUST handle scp-style SSH (`[user@]host:owner/repo.git`), HTTPS (`https://github.com/owner/repo.git`), `file://`, absolute local paths, and git+ssh formats. The `.git` suffix SHALL be stripped. The host MUST compose all on-disk paths; the plugin MUST NOT return any path information.

A remote URL is scp-style when it contains no `://` scheme separator, does not begin with `/`, and contains a `:` that precedes any `/`. For a scp-style URL the SSH user part is optional and MUST NOT be constrained to any particular value: `git`, `forgejo`, `org-1470`, or no user at all are all accepted, and the user is discarded — it never contributes to the host or the segments.

For a URL carrying an explicit scheme, the returned host MUST NOT include the port. Any URL whose parsed path contains an empty segment MUST be rejected with `InvalidArgument` rather than returning a `ProjectID` with an empty segment.

#### Scenario: Parse SSH URL
- **WHEN** `ParseRemoteURL("git@github.com:kalbasit/swm.git")` is called
- **THEN** `ProjectID{host: "github.com", segments: ["kalbasit", "swm"]}` is returned

#### Scenario: Parse scp-style URL with a non-git SSH user
- **WHEN** `ParseRemoteURL("org-1470@git.entreprise.com:Team/Project.git")` is called
- **THEN** `ProjectID{host: "git.entreprise.com", segments: ["Team", "Project"]}` is returned

#### Scenario: Parse scp-style URL with no SSH user
- **WHEN** `ParseRemoteURL("git.entreprise.com:Team/Project.git")` is called
- **THEN** `ProjectID{host: "git.entreprise.com", segments: ["Team", "Project"]}` is returned

#### Scenario: Parse HTTPS URL
- **WHEN** `ParseRemoteURL("https://gitlab.com/foo/bar/baz.git")` is called
- **THEN** `ProjectID{host: "gitlab.com", segments: ["foo", "bar", "baz"]}` is returned

#### Scenario: Schemed URL is not mistaken for scp-style
- **WHEN** `ParseRemoteURL("ssh://git@example.com/owner/repo.git")` is called
- **THEN** `ProjectID{host: "example.com", segments: ["owner", "repo"]}` is returned

#### Scenario: Explicit port is stripped from the host
- **WHEN** `ParseRemoteURL("ssh://git@example.com:2222/owner/repo.git")` is called
- **THEN** `ProjectID{host: "example.com", segments: ["owner", "repo"]}` is returned and the host contains no `:`

#### Scenario: Parse file URL
- **WHEN** `ParseRemoteURL("file:///tmp/foo/bar")` is called
- **THEN** `ProjectID{host: "localhost", segments: ["tmp", "foo", "bar"]}` is returned

#### Scenario: Parse absolute local path
- **WHEN** `ParseRemoteURL("/srv/repos/acme/widget.git")` is called
- **THEN** `ProjectID{host: "localhost", segments: ["srv", "repos", "acme", "widget"]}` is returned

#### Scenario: Empty path segment is rejected
- **WHEN** `ParseRemoteURL("https://gitlab.com/foo//baz.git")` or `ParseRemoteURL("https://gitlab.com/foo/bar/")` is called
- **THEN** a gRPC status error with code `InvalidArgument` is returned and no `ProjectID` with an empty segment is produced

#### Scenario: Unparseable URL
- **WHEN** `ParseRemoteURL("not-a-url")` is called
- **THEN** a gRPC status error with code `InvalidArgument` is returned
