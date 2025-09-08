package gitops

type Target struct {
	Repository Repository
	Branch     TargetBranch
	Directory  string
}

type TargetBranch struct {
	Source         string
	Target         string
	UpstreamSource string // Use an empty string to create an orphan source branch if it does not yet exist.
}

type Targeter interface {
	CreateTarget(req *Request) (*Target, error)
}
