package main

import (
	"context"
	"os"
	"testing"
)

func getToken(t *testing.T) string {
	t.Helper()
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("GITHUB_TOKEN not set, skipping integration test")
	}
	return token
}

func TestQueryOpenPRs(t *testing.T) {
	token := getToken(t)
	prs, err := queryOpenPRs(context.Background(), token, "golang", "go")
	if err != nil {
		t.Fatalf("queryOpenPRs: %v", err)
	}
	if len(prs) == 0 {
		t.Fatal("expected at least one open PR in golang/go")
	}

	// PR #27813 has been open since 2018.
	var found bool
	for _, pr := range prs {
		if pr.Number == 27813 {
			found = true
			if pr.Owner != "golang" {
				t.Errorf("owner = %q, want golang", pr.Owner)
			}
			if pr.Repo != "go" {
				t.Errorf("repo = %q, want go", pr.Repo)
			}
			if pr.Author != "pjebs" {
				t.Errorf("author = %q, want pjebs", pr.Author)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find PR #27813 in golang/go open PRs")
	}

	// Every PR should have valid fields.
	for _, pr := range prs {
		if pr.Number == 0 {
			t.Error("PR has number 0")
		}
		if pr.Author == "" {
			t.Errorf("PR #%d has empty author", pr.Number)
		}
		switch pr.ReviewStatus {
		case "approved", "changes_requested", "review_required", "pending":
		default:
			t.Errorf("PR #%d has unexpected review_status %q", pr.Number, pr.ReviewStatus)
		}
	}
}

func TestCollectPRs(t *testing.T) {
	token := getToken(t)
	prs := collectPRs(context.Background(), token, []string{"golang/go", "invalid", "also/nonexistent-repo-xyz"})

	// Should get results from golang/go and skip invalid repos gracefully.
	if len(prs) == 0 {
		t.Fatal("expected at least one PR from golang/go")
	}
	for _, pr := range prs {
		if pr.Owner != "golang" || pr.Repo != "go" {
			t.Errorf("unexpected repo %s/%s in results", pr.Owner, pr.Repo)
		}
	}
}

func TestParseRepo(t *testing.T) {
	tests := []struct {
		input     string
		owner     string
		name      string
		wantOK    bool
	}{
		{"golang/go", "golang", "go", true},
		{"foo/bar", "foo", "bar", true},
		{"invalid", "", "", false},
		{"", "", "", false},
		{"/repo", "", "", false},
		{"owner/", "", "", false},
	}
	for _, tt := range tests {
		owner, name, ok := parseRepo(tt.input)
		if ok != tt.wantOK || owner != tt.owner || name != tt.name {
			t.Errorf("parseRepo(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.input, owner, name, ok, tt.owner, tt.name, tt.wantOK)
		}
	}
}

func TestMapReviewDecision(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"APPROVED", "approved"},
		{"CHANGES_REQUESTED", "changes_requested"},
		{"REVIEW_REQUIRED", "review_required"},
		{"", "pending"},
		{"UNKNOWN", "pending"},
	}
	for _, tt := range tests {
		got := mapReviewDecision(tt.input)
		if got != tt.want {
			t.Errorf("mapReviewDecision(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
