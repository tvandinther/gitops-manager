# Flow Strategies

GitOps manager supports the usage of different strategies to manage the flow of changes from a source repository to a target repository.

- [Authorisors](#authorisors)
    - [Static Authorisor](#static-authorisor)
- [Targeters](#targeters)
    - [Branch Targeter](#branch-targeter)
- [URL Authenticators](#url-authenticators)
    - [None](#none)
    - [User Password](#user-password)
- [File Copiers](#file-copiers)
    - [Subpath Copier](#subpath-copier)
- [Committers](#committers)
    - [Standard](#standard)
- [Reviewers](#reviewers)
    - [Dummy](#dummy)
    - [Gitea](#gitea)
    - [Gitlab](#gitlab)


## Authorisors

### Static Authorisor
The static authorisor strategy will always deny or allow a request based on a static configuration.

```go
authorisor := &authorisor.Static{
    Allow: true
}

// There is also a shorthand to the above:
authorisor = authorisor.NoAuthorisation
```

## Targeters

### Branch Targeter
The branch targeter strategy uses branches to define environments. Each environment branch may be prefixed and follows the format `<prefix><environment>`. For example, with a prefix of `environment/`, the `staging` environment would correspond to the `environment/staging` branch. The target branch follows the format `<prefix><environment>/<application-name>/<update-id>`, for example `environment/staging/my-app/issue-01`.

There is also the option to configure a directory name under which changes will be scoped to. This is useful to ensure that changes are contained within a specific directory in the target repository, for example `manifests`.

If an environment branch does not exist, the `Orphan` option can be set to create an orphan branch. This is the recommended approach for using branches as environments. If this is set to `false`, a value for `Upstream` must be provided to base the new environment branch on.
```go
targeter := &targeters.Branch{
    Prefix: "environment/", 
    DirectoryName: "manifests", 
    Orphan: true
    // Upstream: "main", // Required if Orphan is false
}
```

## URL Authenticators

### None
The none URL authenticator does not modify the clone URL in any way. This is suitable for public repositories that do not require authentication. Note that the repository must also be publicly writable to use this authenticator. This is not likely to be the case in most scenarios except for testing.

```go
urlAuthenticator := &authenticator.None{}
```

### User Password
The user password URL authenticator adds basic authentication to the clone URL using a username and password (or access token). This is suitable for most cases utilising HTTP for talking to the remote.

```go
urlAuthenticator := &authenticator.UserPassword{
    Username: "your-username",
    Password: "your-access-token",
}
```

## File Copiers

### Subpath Copier
The subpath copier strategy copies all files from the source filesystem to a specified subpath in the target repository. If the subpath does not exist, it will be created. If it does exist, its contents will be replaced with the new files. This ensures a declarative approach where the target state matches the source state. `ManifestDirectoryName` is required to ensure that destructive actions are contained within a specific directory.

```go
fileCopier := &copier.Subpath{
    ManifestDirectoryName: "manifests", // Required to contain destructive actions
    Subpath: "path/to/subpath", // Relative to ManifestDirectoryName
}
```

## Committers

### Standard
The standard committer strategy creates a commit with a specified message and author information. It allows for customization of the commit message and author details, providing flexibility in how changes are recorded in the target repository. With the standard commiter, a new commit is created for each request made to the GitOps server containing changes.

```go
commiter := &committer.Standard{
    Author: &git.Author{
        Name:  "gitops-manager",
        Email: "gitops-manager@example.com",
    },
    CommitSubject: "Update rendered manifests",
    CommitMessageFn: func(req *gitops.Request) string {
        return fmt.Sprintf("Update rendered manifests for %s", req.AppName)
    },
}
```

## Reviewers

### Dummy
The dummy reviewer does not create a real code review or pull request. Instead, it simulates the creation of a review by returning a static URL and optionally completing the review immediately. This is useful for testing and development purposes where you want to verify the flow without interacting with a real remote repository.

```go
reviewer := &reviewer.Dummy{
    URL:      "https://example.com/review/1",
    Complete: true, // Will return a valid review completion if auto-review is enabled
}
```

### Gitea
The Gitea reviewer creates and manages pull requests in a Gitea repository. It uses the Gitea API to create a pull request with a specified title and description, and it can automatically merge the pull request if desired. The Gitea reviewer requires a Gitea client to interact with the Gitea server.

```go
reviewer := &reviewer.Gitea{
    Client: giteaClient,
    MergeOptions: &gitea.MergePullRequestOption{
        Style:                  gitea.MergeStyleRebase,
        DeleteBranchAfterMerge: true,
    },
}
```

### Gitlab
The Gitlab reviewer creates and manages merge requests in a Gitlab repository. It uses the Gitlab API to create a merge request with a specified title and description, and it can automatically merge the merge request if desired. The Gitlab reviewer requires a Gitlab client to interact with the Gitlab server.

```go
reviewer := &reviewer.Gitlab{
    Client: gitlabClient,
    MergeOptions: &reviewer.GitlabMergeOptions{
        AutoMerge:     true, // Wlll fail with 405 if the project does not have a pipeline
        Squash:        true,
	    CommitMessage: "Merge via gitops-manager",
	    DeleteBranch:  true,
    },
}
```
