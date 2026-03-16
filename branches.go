package lakefs

import (
	"context"
	"fmt"
	"net/http"
)

type Branches service

type ListBranchesOptions struct {
	ListOptions `url:",inline"`

	Prefix     string `url:"prefix,omitempty"`
	ShowHidden bool   `url:"show_hidden,omitempty"`
}

func (b *Branches) ListBranches(ctx context.Context, repository string, opts *ListBranchesOptions) (*Records[Ref], error) {
	u := fmt.Sprintf("repositories/%s/branches", repository)
	var records Records[Ref]
	_, err := b.client.InvokeWithCredential(ctx, http.MethodGet, u, opts, &records)
	if err != nil {
		return nil, err
	}
	return &records, err
}

func (b *Branches) GetBranch(ctx context.Context, repository, branch string) (*Ref, error) {
	u := fmt.Sprintf("repositories/%s/branches/%s", repository, branch)
	var row Ref
	_, err := b.client.InvokeWithCredential(ctx, http.MethodGet, u, nil, &row)
	if err != nil {
		return nil, err
	}
	return &row, err
}

type BranchCreation struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Force  bool   `json:"force,omitempty"`
	Hidden bool   `json:"hidden,omitempty"`
}

func (b *Branches) CreateBranch(ctx context.Context, repository string, branchCreation *BranchCreation) (*Ref, error) {
	u := fmt.Sprintf("repositories/%s/branches", repository)
	var commitID string
	_, err := b.client.InvokeWithCredential(ctx, http.MethodPost, u, branchCreation, &commitID)
	if err != nil {
		return nil, err
	}
	return &Ref{
		ID:       branchCreation.Name,
		CommitID: commitID,
	}, err
}

type DeleteBranchOptions struct {
	Force bool `url:"force,omitempty"`
}

func (b *Branches) DeleteBranch(ctx context.Context, repository, branch string, opts ...*DeleteBranchOptions) error {
	u := fmt.Sprintf("repositories/%s/branches/%s", repository, branch)
	if len(opts) > 0 && opts[0] != nil && opts[0].Force {
		u += "?force=true"
	}
	_, err := b.client.InvokeWithCredential(ctx, http.MethodDelete, u, nil, nil)
	return err
}

func (b *Branches) Reset(ctx context.Context, repository, branch string, resetCreation *ResetCreation) error {
	u := fmt.Sprintf("repositories/%s/branches/%s", repository, branch)
	_, err := b.client.InvokeWithCredential(ctx, http.MethodPut, u, resetCreation, nil)
	return err
}

func (b *Branches) Revert(ctx context.Context, repository, branch string, revertCreation *RevertCreation) error {
	u := fmt.Sprintf("repositories/%s/branches/%s/revert", repository, branch)
	_, err := b.client.InvokeWithCredential(ctx, http.MethodPost, u, revertCreation, nil)
	return err
}

func (b *Branches) CherryPick(ctx context.Context, repository, branch string, cherryPickCreation *CherryPickCreation) error {
	u := fmt.Sprintf("repositories/%s/branches/%s/cherry-pick", repository, branch)
	_, err := b.client.InvokeWithCredential(ctx, http.MethodPost, u, cherryPickCreation, nil)
	return err
}

type DiffOptions struct {
	ListOptions `url:",inline"`

	Prefix    string `url:"prefix,omitempty"`
	Delimiter string `url:"delimiter,omitempty"`
}

func (b *Branches) Diff(ctx context.Context, repository, branch string, opts *DiffOptions) (*Records[Diff], error) {
	u := fmt.Sprintf("repositories/%s/branches/%s/diff", repository, branch)
	var records Records[Diff]
	_, err := b.client.InvokeWithCredential(ctx, http.MethodGet, u, opts, &records)
	if err != nil {
		return nil, err
	}
	return &records, err
}
