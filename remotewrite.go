package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/castai/promwrite"
)

func buildWriteRequest(prs []PRInfo) *promwrite.WriteRequest {
	now := time.Now()
	ts := make([]promwrite.TimeSeries, len(prs))
	for i, pr := range prs {
		ts[i] = promwrite.TimeSeries{
			Labels: []promwrite.Label{
				{Name: "__name__", Value: "github_pr_open"},
				{Name: "author", Value: pr.Author},
				{Name: "draft", Value: strconv.FormatBool(pr.Draft)},
				{Name: "number", Value: strconv.Itoa(pr.Number)},
				{Name: "owner", Value: pr.Owner},
				{Name: "repo", Value: pr.Repo},
				{Name: "review_status", Value: pr.ReviewStatus},
			},
			Sample: promwrite.Sample{
				Time:  now,
				Value: 1,
			},
		}
	}
	return &promwrite.WriteRequest{TimeSeries: ts}
}

func pushMetrics(ctx context.Context, cfg Config, req *promwrite.WriteRequest) error {
	client := promwrite.NewClient(cfg.RemoteWriteURL, promwrite.HttpClient(&http.Client{
		Transport: &basicAuthTransport{
			username: cfg.RemoteWriteUser,
			password: cfg.RemoteWritePass,
		},
	}))
	_, err := client.Write(ctx, req)
	if err != nil {
		return fmt.Errorf("remote write: %w", err)
	}
	return nil
}

type basicAuthTransport struct {
	username string
	password string
}

func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(t.username, t.password)
	return http.DefaultTransport.RoundTrip(req)
}
