package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-github/v84/github"
)

type PRInfo struct {
	Repo         string
	Owner        string
	Number       int
	Author       string
	Draft        bool
	ReviewStatus string // "approved", "changes_requested", "pending"
}

func collectPRs(ctx context.Context, client *github.Client, repos []string) []PRInfo {
	var result []PRInfo
	for _, repo := range repos {
		owner, name, ok := parseRepo(repo)
		if !ok {
			log.Printf("skipping invalid repo %q (expected owner/repo)", repo)
			continue
		}
		prs, err := listOpenPRs(ctx, client, owner, name)
		if err != nil {
			log.Printf("error listing PRs for %s: %v", repo, err)
			continue
		}
		result = append(result, prs...)
	}
	return result
}

func parseRepo(repo string) (owner, name string, ok bool) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func listOpenPRs(ctx context.Context, client *github.Client, owner, repo string) ([]PRInfo, error) {
	var result []PRInfo
	opts := &github.PullRequestListOptions{
		State:       "open",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		prs, resp, err := client.PullRequests.List(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("list PRs: %w", err)
		}
		for _, pr := range prs {
			reviewStatus := getReviewStatus(ctx, client, owner, repo, pr.GetNumber())
			result = append(result, PRInfo{
				Repo:         repo,
				Owner:        owner,
				Number:       pr.GetNumber(),
				Author:       pr.GetUser().GetLogin(),
				Draft:        pr.GetDraft(),
				ReviewStatus: reviewStatus,
			})
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return result, nil
}

func getReviewStatus(ctx context.Context, client *github.Client, owner, repo string, prNumber int) string {
	opts := &github.ListOptions{PerPage: 100}
	// Track the latest review state per user.
	latestByUser := make(map[string]string)
	for {
		reviews, resp, err := client.PullRequests.ListReviews(ctx, owner, repo, prNumber, opts)
		if err != nil {
			log.Printf("error listing reviews for %s/%s#%d: %v", owner, repo, prNumber, err)
			return "pending"
		}
		for _, r := range reviews {
			user := r.GetUser().GetLogin()
			state := r.GetState() // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED
			// Only track actionable states.
			switch state {
			case "APPROVED", "CHANGES_REQUESTED":
				latestByUser[user] = state
			case "DISMISSED":
				delete(latestByUser, user)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	hasApproval := false
	for _, state := range latestByUser {
		if state == "CHANGES_REQUESTED" {
			return "changes_requested"
		}
		if state == "APPROVED" {
			hasApproval = true
		}
	}
	if hasApproval {
		return "approved"
	}
	return "pending"
}
