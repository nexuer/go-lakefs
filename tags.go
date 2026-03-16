package lakefs

import (
	"context"
	"fmt"
	"net/http"
)

type Tags service

type ListTagsOptions struct {
	ListOptions `url:",inline"`

	Prefix string `url:"prefix,omitempty"`
}

func (t *Tags) ListTags(ctx context.Context, repository string, opts *ListTagsOptions) (*Records[Ref], error) {
	u := fmt.Sprintf("repositories/%s/tags", repository)
	var records Records[Ref]
	_, err := t.client.InvokeWithCredential(ctx, http.MethodGet, u, opts, &records)
	if err != nil {
		return nil, err
	}
	return &records, err
}

type TagCreation struct {
	ID    string `json:"id"`
	Ref   string `json:"ref"`
	Force bool   `json:"force"`
}

func (t *Tags) CreateTag(ctx context.Context, repository string, tagCreation *TagCreation) (*Ref, error) {
	u := fmt.Sprintf("repositories/%s/tags", repository)
	var row Ref
	_, err := t.client.InvokeWithCredential(ctx, http.MethodPost, u, tagCreation, &row)
	if err != nil {
		return nil, err
	}
	return &row, err
}

type DeleteTagOptions struct {
	Force bool `url:"force,omitempty"`
}

func (t *Tags) DeleteTag(ctx context.Context, repository string, tag string, opts ...*DeleteTagOptions) error {
	u := fmt.Sprintf("repositories/%s/tags/%s", repository, tag)
	if len(opts) > 0 && opts[0] != nil && opts[0].Force {
		u += "?force=true"
	}
	_, err := t.client.InvokeWithCredential(ctx, http.MethodDelete, u, nil, nil)
	return err
}

func (t *Tags) GetTag(ctx context.Context, repository string, tag string) (*Ref, error) {
	u := fmt.Sprintf("repositories/%s/tags/%s", repository, tag)
	var row Ref
	_, err := t.client.InvokeWithCredential(ctx, http.MethodGet, u, nil, &row)
	if err != nil {
		return nil, err
	}
	return &row, err
}
