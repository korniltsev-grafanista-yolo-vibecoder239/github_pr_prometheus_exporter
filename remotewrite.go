package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/snappy"
	"google.golang.org/protobuf/encoding/protowire"
)

// Protobuf field numbers matching the Prometheus remote write spec.
// WriteRequest: field 1 = repeated TimeSeries
// TimeSeries:   field 1 = repeated Label, field 2 = repeated Sample
// Label:        field 1 = name (string), field 2 = value (string)
// Sample:       field 1 = value (double), field 2 = timestamp (int64)

func buildWriteRequest(prs []PRInfo) []byte {
	now := time.Now().UnixMilli()
	var wreq []byte
	for _, pr := range prs {
		ts := buildTimeSeries(pr, now)
		wreq = protowire.AppendTag(wreq, 1, protowire.BytesType)
		wreq = protowire.AppendBytes(wreq, ts)
	}
	return wreq
}

func buildTimeSeries(pr PRInfo, timestampMs int64) []byte {
	labels := [][2]string{
		{"__name__", "github_pr_open"},
		{"author", pr.Author},
		{"draft", strconv.FormatBool(pr.Draft)},
		{"number", strconv.Itoa(pr.Number)},
		{"owner", pr.Owner},
		{"repo", pr.Repo},
		{"review_status", pr.ReviewStatus},
	}

	var ts []byte
	for _, l := range labels {
		lbl := encodeLabel(l[0], l[1])
		ts = protowire.AppendTag(ts, 1, protowire.BytesType)
		ts = protowire.AppendBytes(ts, lbl)
	}

	sample := encodeSample(1.0, timestampMs)
	ts = protowire.AppendTag(ts, 2, protowire.BytesType)
	ts = protowire.AppendBytes(ts, sample)

	return ts
}

func encodeLabel(name, value string) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, name)
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendString(b, value)
	return b
}

func encodeSample(value float64, timestampMs int64) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.Fixed64Type)
	b = protowire.AppendFixed64(b, math.Float64bits(value))
	b = protowire.AppendTag(b, 2, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(timestampMs))
	return b
}

func pushMetrics(ctx context.Context, cfg Config, data []byte) error {
	compressed := snappy.Encode(nil, data)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.RemoteWriteURL, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")
	req.SetBasicAuth(cfg.RemoteWriteUser, cfg.RemoteWritePass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("remote write returned %d: %s", resp.StatusCode, body)
	}
	return nil
}
