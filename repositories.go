package lakefs

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nexuer/ghttp"
)

type Repositories service

type ListRepositoriesOptions struct {
	ListOptions `url:",inline"`

	Prefix string `url:"prefix,omitempty"`
	Search string `url:"search,omitempty"`
}

type Repository struct {
	ID               string    `json:"id"`
	CreationDate     Timestamp `json:"creation_date"`
	DefaultBranch    string    `json:"default_branch"`
	ReadOnly         bool      `json:"read_only"`
	StorageID        string    `json:"storage_id"`
	StorageNamespace string    `json:"storage_namespace"`
}

func (r *Repositories) ListRepositories(ctx context.Context, opts *ListRepositoriesOptions) (*Records[Repository], error) {
	u := "repositories"
	var records Records[Repository]
	_, err := r.client.InvokeWithCredential(ctx, http.MethodGet, u, opts, &records)
	return &records, err
}

func (r *Repositories) GetRepository(ctx context.Context, id string) (*Repository, error) {
	u := fmt.Sprintf("repositories/%s", id)
	var repository Repository
	_, err := r.client.InvokeWithCredential(ctx, http.MethodGet, u, nil, &repository)
	return &repository, err
}

type RepositoryCreation struct {
	Name             string `json:"name"`
	DefaultBranch    string `json:"default_branch"`
	ReadOnly         bool   `json:"read_only"`
	SampleData       bool   `json:"sample_data"`
	StorageID        string `json:"storage_id"`
	StorageNamespace string `json:"storage_namespace"`
}

func (r *Repositories) CreateRepository(ctx context.Context, repositoryCreation *RepositoryCreation) (*Repository, error) {
	u := "repositories"
	var respData Repository
	_, err := r.client.InvokeWithCredential(ctx, http.MethodPost, u, repositoryCreation, &respData)
	return &respData, err
}

func (r *Repositories) CreateBareRepository(ctx context.Context, repositoryCreation *RepositoryCreation) (*Repository, error) {
	u := "repositories?bare=true"
	var respData Repository
	_, err := r.client.InvokeWithCredential(ctx, http.MethodPost, u, repositoryCreation, &respData)
	return &respData, err
}

type DeleteRepositoryOptions struct {
	Force bool `url:"force,omitempty"`
}

func (r *Repositories) DeleteRepository(ctx context.Context, id string, opts ...*DeleteRepositoryOptions) error {
	u := fmt.Sprintf("repositories/%s", id)
	if len(opts) > 0 && opts[0] != nil && opts[0].Force {
		u += "?force=true"
	}
	_, err := r.client.InvokeWithCredential(ctx, http.MethodDelete, u, nil, nil)
	return err
}

type RepositoryMetadata map[string]string

func (r *Repositories) GetRepositoryMetadata(ctx context.Context, id string) (RepositoryMetadata, error) {
	u := fmt.Sprintf("repositories/%s/metadata", id)
	var metadata RepositoryMetadata
	_, err := r.client.InvokeWithCredential(ctx, http.MethodGet, u, nil, &metadata)
	return metadata, err
}

type RepositoryGCRules struct {
	DefaultRetentionDays int             `json:"default_retention_days"`
	Branches             []*BranchGCRule `json:"branches"`
}

type BranchGCRule struct {
	BranchID      string `json:"branch_id"`
	RetentionDays int    `json:"retention_days"`
}

func (r *Repositories) GetRepositoryGCRules(ctx context.Context, id string) (*RepositoryGCRules, error) {
	u := fmt.Sprintf("repositories/%s/settings/gc_rules", id)
	var rules RepositoryGCRules
	_, err := r.client.InvokeWithCredential(ctx, http.MethodGet, u, nil, &rules)
	return &rules, err
}

func (r *Repositories) PutRepositoryGCRules(ctx context.Context, id string, rules *RepositoryGCRules) error {
	u := fmt.Sprintf("repositories/%s/settings/gc_rules", id)
	_, err := r.client.InvokeWithCredential(ctx, http.MethodPut, u, rules, nil)
	return err
}

func (r *Repositories) DeleteRepositoryGCRules(ctx context.Context, id string) error {
	u := fmt.Sprintf("repositories/%s/settings/gc_rules", id)
	_, err := r.client.InvokeWithCredential(ctx, http.MethodDelete, u, nil, nil)
	return err
}

type BranchProtectionRule struct {
	Pattern string `json:"pattern"`
}

func (r *Repositories) GetBranchProtectionRules(ctx context.Context, id string) ([]*BranchProtectionRule, error) {
	u := fmt.Sprintf("repositories/%s/settings/branch_protection", id)
	var rules []*BranchProtectionRule
	_, err := r.client.InvokeWithCredential(ctx, http.MethodGet, u, nil, &rules)
	return rules, err
}

type PutBranchProtectionRulesOptions struct {
	IfMatch string
	Rules   []*BranchProtectionRule
}

func (p *PutBranchProtectionRulesOptions) Request() []ghttp.RequestFunc {
	if p == nil {
		return nil
	}
	if p.IfMatch == "" {
		return nil
	}
	return []ghttp.RequestFunc{
		func(request *http.Request) error {
			request.Header.Set("If-Match", p.IfMatch)
			return nil
		},
	}
}

func (r *Repositories) PutBranchProtectionRules(ctx context.Context, id string, opts *PutBranchProtectionRulesOptions) error {
	u := fmt.Sprintf("repositories/%s/settings/branch_protection", id)
	_, err := r.client.InvokeWithCredential(ctx, http.MethodPut, u, opts.Rules, nil, opts.Request()...)
	return err
}
