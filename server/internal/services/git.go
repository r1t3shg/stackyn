package services

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"go.uber.org/zap"
)

// GitService handles Git repository operations
type GitService struct {
	logger      *zap.Logger
	cloneDir    string // Base directory for cloning repos
}

// NewGitService creates a new Git service
func NewGitService(logger *zap.Logger, cloneDir string) *GitService {
	return &GitService{
		logger:   logger,
		cloneDir: cloneDir,
	}
}

// CloneOptions represents options for cloning a repository
type CloneOptions struct {
	RepoURL   string
	Branch    string
	Shallow   bool // Enable shallow clone
	Depth     int  // Depth for shallow clone (default: 1)
}

// CloneResult represents the result of a clone operation
type CloneResult struct {
	Path      string // Path to cloned repository
	CommitSHA string // SHA of the checked out commit
	Branch    string // Branch that was checked out
}

// ValidatePublicRepo validates that a repository is public and accessible
func (s *GitService) ValidatePublicRepo(ctx context.Context, repoURL string) error {
	// Parse GitHub URL
	if !strings.Contains(repoURL, "github.com") {
		return fmt.Errorf("only GitHub repositories are supported")
	}

	// Convert SSH URL to HTTPS if needed
	httpsURL := s.normalizeGitHubURL(repoURL)

	// Try to access the repository via GitHub API or HTTP
	// For GitHub, we can check if the repo is public by trying to access it
	apiURL := s.getGitHubAPIURL(httpsURL)
	if apiURL == "" {
		return fmt.Errorf("invalid GitHub repository URL: %s", repoURL)
	}

	// Make a HEAD request to check if repo is accessible (public)
	req, err := http.NewRequestWithContext(ctx, "HEAD", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second, // 10 second timeout
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("repository is not accessible (may be private): %w", err)
	}
	defer resp.Body.Close()

	// Check if repository is accessible (200 or 301/302 for redirects)
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("repository not found or is private")
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("repository is not accessible (status: %d)", resp.StatusCode)
	}

	s.logger.Info("Repository validated as public",
		zap.String("repo_url", repoURL),
		zap.Int("status_code", resp.StatusCode),
	)

	return nil
}

// Clone clones a repository to a temporary directory
func (s *GitService) Clone(ctx context.Context, opts CloneOptions) (*CloneResult, error) {
	// Validate repository is public
	if err := s.ValidatePublicRepo(ctx, opts.RepoURL); err != nil {
		return nil, fmt.Errorf("repository validation failed: %w", err)
	}

	// Normalize URL to HTTPS
	httpsURL := s.normalizeGitHubURL(opts.RepoURL)

	// Create unique directory for this clone
	repoName := s.extractRepoName(httpsURL)
	clonePath := filepath.Join(s.cloneDir, repoName)

	// Clean up existing directory if it exists
	if _, err := os.Stat(clonePath); err == nil {
		s.logger.Warn("Clone directory already exists, removing", zap.String("path", clonePath))
		if err := os.RemoveAll(clonePath); err != nil {
			return nil, fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}

	// Ensure clone directory exists
	if err := os.MkdirAll(s.cloneDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create clone directory: %w", err)
	}

	// Prepare clone options
	gitCloneOpts := &git.CloneOptions{
		URL:      httpsURL,
		Progress: nil, // No progress output
	}

	// Set branch reference
	if opts.Branch != "" {
		gitCloneOpts.ReferenceName = plumbing.NewBranchReferenceName(opts.Branch)
		gitCloneOpts.SingleBranch = true
	}

	// Configure shallow clone
	if opts.Shallow {
		depth := opts.Depth
		if depth == 0 {
			depth = 1 // Default to depth 1 for shallow clone
		}
		gitCloneOpts.Depth = depth
	}

	s.logger.Info("Cloning repository",
		zap.String("repo_url", httpsURL),
		zap.String("branch", opts.Branch),
		zap.Bool("shallow", opts.Shallow),
		zap.Int("depth", gitCloneOpts.Depth),
		zap.String("clone_path", clonePath),
	)

	// Clone repository
	repo, err := git.PlainClone(clonePath, false, gitCloneOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Get the checked out commit
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	commitSHA := ref.Hash().String()
	branch := ref.Name().Short()

	s.logger.Info("Repository cloned successfully",
		zap.String("clone_path", clonePath),
		zap.String("commit_sha", commitSHA),
		zap.String("branch", branch),
	)

	return &CloneResult{
		Path:      clonePath,
		CommitSHA: commitSHA,
		Branch:    branch,
	}, nil
}

// Cleanup removes a cloned repository
func (s *GitService) Cleanup(clonePath string) error {
	if err := os.RemoveAll(clonePath); err != nil {
		return fmt.Errorf("failed to remove clone directory: %w", err)
	}
	s.logger.Info("Cleaned up clone directory", zap.String("path", clonePath))
	return nil
}

// normalizeGitHubURL converts SSH URLs to HTTPS and normalizes GitHub URLs
func (s *GitService) normalizeGitHubURL(url string) string {
	// Convert SSH URL to HTTPS
	if strings.HasPrefix(url, "git@github.com:") {
		url = strings.Replace(url, "git@github.com:", "https://github.com/", 1)
	}

	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// Ensure it's an HTTPS URL
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		// Assume it's a GitHub repo and add https://
		if strings.Contains(url, "github.com") {
			url = "https://" + url
		}
	}

	return url
}

// extractRepoName extracts repository name from URL
func (s *GitService) extractRepoName(url string) string {
	// Remove protocol and .git suffix
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, ".git")

	// Extract owner/repo
	parts := strings.Split(url, "/")
	if len(parts) >= 3 {
		// Format: github.com/owner/repo
		return fmt.Sprintf("%s_%s", parts[1], parts[2])
	}

	// Fallback: use sanitized URL
	return strings.ReplaceAll(url, "/", "_")
}

// getGitHubAPIURL converts a GitHub repository URL to GitHub API URL
func (s *GitService) getGitHubAPIURL(repoURL string) string {
	// Remove protocol
	url := strings.TrimPrefix(repoURL, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, ".git")

	// Parse github.com/owner/repo
	parts := strings.Split(url, "/")
	if len(parts) < 3 || parts[0] != "github.com" {
		return ""
	}

	owner := parts[1]
	repo := parts[2]

	// GitHub API endpoint for repository info
	return fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
}

