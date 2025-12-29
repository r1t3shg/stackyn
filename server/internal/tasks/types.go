package tasks

// Task type constants
const (
	TypeBuildTask   = "build_task"
	TypeDeployTask  = "deploy_task"
	TypeCleanupTask = "cleanup_task"
)

// Task queue names
const (
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"
)

// BuildTaskPayload represents the payload for a build task
type BuildTaskPayload struct {
	AppID        string `json:"app_id"`
	BuildJobID   string `json:"build_job_id"`
	RepoURL      string `json:"repo_url"`
	Branch        string `json:"branch"`
	CommitSHA    string `json:"commit_sha,omitempty"`
}

// DeployTaskPayload represents the payload for a deploy task
type DeployTaskPayload struct {
	AppID         string `json:"app_id"`
	DeploymentID  string `json:"deployment_id"`
	BuildJobID    string `json:"build_job_id"`
	ImageName     string `json:"image_name"`
	Subdomain     string `json:"subdomain,omitempty"`
}

// CleanupTaskPayload represents the payload for a cleanup task
type CleanupTaskPayload struct {
	AppID        string   `json:"app_id"`
	DeploymentID string   `json:"deployment_id,omitempty"`
	ContainerIDs []string `json:"container_ids,omitempty"`
	ImageNames   []string `json:"image_names,omitempty"`
}

