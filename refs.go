package lakefs

import (
	"context"
	"fmt"
	"net/http"
)

type Refs service

type Ref struct {
	ID       string `json:"id"`
	CommitID string `json:"commit_id"`
}

type ListCommitsOptions struct {
	ListOptions `url:",inline"`

	Objects     []string `url:"objects,omitempty"`
	Prefixes    []string `url:"prefixes,omitempty"`
	Limit       bool     `url:"limit,omitempty"`
	FirstParent bool     `url:"first_parent,omitempty"`
	Since       string   `url:"since,omitempty"`
	StopAt      string   `url:"stop_at,omitempty"`
}

func (r *Refs) ListCommits(ctx context.Context, repository, ref string, opts *ListCommitsOptions) (*Records[Commit], error) {
	u := fmt.Sprintf("repositories/%s/refs/%s/commits", repository, ref)
	var records Records[Commit]
	_, err := r.client.InvokeWithCredential(ctx, http.MethodGet, u, opts, &records)
	return &records, err
}
