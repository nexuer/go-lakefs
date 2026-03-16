package lakefs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type Commits service

type Commit struct {
	ID           string            `json:"id,omitempty"`
	Parents      []string          `json:"parents,omitempty"`
	Committer    string            `json:"committer,omitempty"`
	Message      string            `json:"message,omitempty"`
	CreationDate Timestamp         `json:"creation_date,omitempty"`
	MetaRangeID  string            `json:"meta_range_id,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Generation   int64             `json:"generation,omitempty"`
	Version      int               `json:"version,omitempty"`
}

func (c *Commits) GetCommit(ctx context.Context, repository, commitID string) (*Commit, error) {
	u := fmt.Sprintf("repositories/%s/commits/%s", repository, commitID)
	var commit Commit
	_, err := c.client.InvokeWithCredential(ctx, http.MethodGet, u, nil, &commit)
	if err != nil {
		return nil, err
	}
	return &commit, err
}

type CommitCreation struct {
	SourceMetarange string `json:"-"`

	Message    string            `json:"message,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Date       *Timestamp        `json:"date,omitempty"`
	AllowEmpty bool              `json:"allow_empty,omitempty"`
	Force      bool              `json:"force,omitempty"`
}

func (c *Commits) CreateCommit(ctx context.Context, repository, branch string, commitCreation *CommitCreation) (*Commit, error) {
	u := fmt.Sprintf("repositories/%s/branches/%s/commits", repository, branch)
	if commitCreation != nil && commitCreation.SourceMetarange != "" {
		u += fmt.Sprintf("?source_metarange=%s", url.QueryEscape(commitCreation.SourceMetarange))
	}
	var row Commit
	_, err := c.client.InvokeWithCredential(ctx, http.MethodPost, u, commitCreation, &row)
	if err != nil {
		return nil, err
	}
	return &row, err
}
