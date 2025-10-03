package git

import (
	"fmt"
	"testing"
)

func TestLocalGitCheck(t *testing.T) {
	// Get Git info for the current directory -- should complete without error
	info, err := GetGitInfo(".")
	if err != nil {
		t.Fatalf("Failed to get Git info: %v", err)
	}

	// Perform assertions on the Git info
	if info == nil {
		t.Fatal("Expected Git info, got nil")
	}

	fmt.Printf("Repository: %v\n", info.Repository)
	fmt.Printf("Commit Hash: %v\n", info.CommitHash)
	fmt.Printf("Branch: %v\n", info.Branch)
	fmt.Printf("Tags: %v\n", info.Tag)
	fmt.Printf("Is Dirty: %v\n", info.IsDirty)
}
