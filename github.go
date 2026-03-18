package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

const graphqlEndpoint = "https://api.github.com/graphql"

const prQuery = `query($owner: String!, $name: String!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    pullRequests(states: OPEN, first: 100, after: $cursor) {
      pageInfo { hasNextPage endCursor }
      nodes {
        number
        author { login }
        isDraft
        reviewDecision
      }
    }
  }
}`

type PRInfo struct {
	Repo         string
	Owner        string
	Number       int
	Author       string
	Draft        bool
	ReviewStatus string // "approved", "changes_requested", "review_required", "pending"
}

func collectPRs(ctx context.Context, token string, repos []string) []PRInfo {
	var result []PRInfo
	for _, repo := range repos {
		owner, name, ok := parseRepo(repo)
		if !ok {
			log.Printf("skipping invalid repo %q (expected owner/repo)", repo)
			continue
		}
		prs, err := queryOpenPRs(ctx, token, owner, name)
		if err != nil {
			log.Printf("error querying PRs for %s: %v", repo, err)
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

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphqlResponse struct {
	Data struct {
		Repository struct {
			PullRequests struct {
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []struct {
					Number         int    `json:"number"`
					Author         struct{ Login string } `json:"author"`
					IsDraft        bool   `json:"isDraft"`
					ReviewDecision string `json:"reviewDecision"`
				} `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func queryOpenPRs(ctx context.Context, token, owner, name string) ([]PRInfo, error) {
	var result []PRInfo
	var cursor *string

	for {
		vars := map[string]any{"owner": owner, "name": name}
		if cursor != nil {
			vars["cursor"] = *cursor
		}

		resp, err := doGraphQL(ctx, token, graphqlRequest{Query: prQuery, Variables: vars})
		if err != nil {
			return nil, err
		}
		if len(resp.Errors) > 0 {
			return nil, fmt.Errorf("graphql: %s", resp.Errors[0].Message)
		}

		prs := resp.Data.Repository.PullRequests
		for _, node := range prs.Nodes {
			result = append(result, PRInfo{
				Repo:         name,
				Owner:        owner,
				Number:       node.Number,
				Author:       node.Author.Login,
				Draft:        node.IsDraft,
				ReviewStatus: mapReviewDecision(node.ReviewDecision),
			})
		}

		if !prs.PageInfo.HasNextPage {
			break
		}
		cursor = &prs.PageInfo.EndCursor
	}
	return result, nil
}

func doGraphQL(ctx context.Context, token string, payload graphqlRequest) (*graphqlResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("github returned %d: %s", resp.StatusCode, b)
	}

	var result graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

func mapReviewDecision(decision string) string {
	switch decision {
	case "APPROVED":
		return "approved"
	case "CHANGES_REQUESTED":
		return "changes_requested"
	case "REVIEW_REQUIRED":
		return "review_required"
	default:
		return "pending"
	}
}
