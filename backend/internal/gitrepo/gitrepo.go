package gitrepo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Cloner struct {
	WorkDir string
}

func NewCloner(workDir string) *Cloner {
	return &Cloner{WorkDir: workDir}
}

func (c *Cloner) Clone(repoURL string, deploymentID int, branch string) (string, error) {
	repoDir := filepath.Join(c.WorkDir, fmt.Sprintf("deployment-%d", deploymentID))

	// Remove directory if it exists
	if err := os.RemoveAll(repoDir); err != nil {
		return "", fmt.Errorf("failed to clean directory: %w", err)
	}

	// Clone repository with specific branch
	// First clone the repository (shallow clone for the specific branch)
	cmd := exec.Command("git", "clone", "--branch", branch, "--single-branch", "--depth", "1", repoURL, repoDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}

	return repoDir, nil
}

// CheckDockerfile checks if a Dockerfile exists in the repository directory
func CheckDockerfile(repoPath string) error {
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")

	// Check if Dockerfile exists
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("dockerfile not found in repository root directory")
	}

	return nil
}
