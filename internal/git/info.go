package git

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type GitInfo struct {
	Repository *git.Repository
	CommitHash string
	Branch     string
	Tag        []string
	IsDirty    bool
}

func GetGitInfo(repoPath string) (*GitInfo, error) {
	// Check if the provided path is a valid Git repository, seeking upwards if necessary

	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, fmt.Errorf("failed to find a Git repository that path %q belongs to: %w", repoPath, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree for repository %q: %w", repoPath, err)
	}

	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD reference for repository %q: %w", repoPath, err)
	}

	branchName := headRef.Name()
	commitHash := headRef.Hash().String()

	// Find all tags pointing to the current commit
	var tags []string
	tagRefs, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	err = tagRefs.ForEach(func(ref *plumbing.Reference) error {
		revHash, err := repo.ResolveRevision(plumbing.Revision(ref.Name()))
		if err != nil {
			return fmt.Errorf("failed to get tag commit object for tag %q: %w", ref.Name().Short(), err)
		}
		if *revHash == headRef.Hash() {
			tags = append(tags, ref.Name().Short())
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate over tags: %w", err)
	}

	// Check if there is any uncommitted change in the working tree
	status, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree status for repository %q: %w", repoPath, err)
	}

	gitInfo := &GitInfo{
		Repository: repo,
		CommitHash: commitHash,
		Branch:     branchName.Short(),
		Tag:        tags,
		IsDirty:    !status.IsClean(),
	}

	return gitInfo, nil
}
