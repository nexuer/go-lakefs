package lakefs

import (
	"fmt"
	"strconv"
	"time"
)

type APIVersion string

const (
	V1 APIVersion = "v1"
)

type Pagination struct {
	HasMore    bool   `json:"has_more"`
	NextOffset string `json:"next_offset"`
	Results    int64  `json:"results"`
	MaxPerPage int64  `json:"max_per_page"`
}

type Records[T any] struct {
	Pagination Pagination `json:"pagination"`
	Results    []*T       `json:"results"`
}

type Error struct {
	Message string `json:"message"`
}

func (e Error) Error() string {
	return e.Message
}

type Timestamp struct {
	time.Time
}

func NewTimestamp(t time.Time) *Timestamp {
	return &Timestamp{t}
}

func NewTimestampFromUnix(timestamp int64) *Timestamp {
	return &Timestamp{time.Unix(timestamp, 0)}
}

func (t *Timestamp) ToTime() time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.Time
}

func (t *Timestamp) ToUnix() int64 {
	if t == nil {
		return time.Time{}.Unix()
	}
	return t.Time.Unix()
}

func (t *Timestamp) MarshalJSON() ([]byte, error) {
	if t == nil || t.IsZero() {
		return []byte("0"), nil
	}
	ts := t.Unix()
	return []byte(fmt.Sprintf("%d", ts)), nil
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	ts, err := strconv.Atoi(string(b))
	if err != nil {
		return err
	}
	t.Time = time.Unix(int64(ts), 0)
	return nil
}

type ListOptions struct {
	After  string `url:"after,omitempty"`
	Amount int    `url:"amount,omitempty"`
}

type ResetType string

const (
	ObjectResetType       ResetType = "object"
	CommonPrefixResetType ResetType = "common_prefix"
	ResetResetType        ResetType = "reset"
)

type ResetCreation struct {
	Type  ResetType `json:"type,omitempty"`
	Path  string    `json:"path,omitempty"`
	Force bool      `json:"force,omitempty"`
}

type RevertCreation struct {
	Ref             string           `json:"ref,omitempty"`
	ParentNumber    int              `json:"parent_number,omitempty"`
	Force           bool             `json:"force,omitempty"`
	AllowEmpty      bool             `json:"allow_empty,omitempty"`
	CommitOverrides *CommitOverrides `json:"commit_overrides,omitempty"`
}

type CommitOverrides struct {
	Message  string            `json:"message,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type CherryPickCreation struct {
	Ref             string           `json:"ref,omitempty"`
	ParentNumber    int              `json:"parent_number,omitempty"`
	Force           bool             `json:"force,omitempty"`
	CommitOverrides *CommitOverrides `json:"commit_overrides,omitempty"`
}

type PathType string

const (
	CommonPrefixPathType PathType = "common_prefix"
	ObjectPathType       PathType = "object"
)

type DiffType string

const (
	Added         DiffType = "added"
	Removed       DiffType = "removed"
	Changed       DiffType = "changed"
	Conflict      DiffType = "conflict"
	PrefixChanged DiffType = "prefix_changed"
)

type Diff struct {
	Type      DiffType        `json:"type,omitempty"`
	Path      string          `json:"path,omitempty"`
	PathType  PathType        `json:"path_type,omitempty"`
	SizeBytes int64           `json:"size_bytes,omitempty"`
	Right     *DiffObjectStat `json:"right,omitempty"`
}

type DiffObjectStat struct {
	Checksum    string             `json:"checksum,omitempty"`
	Mtime       *Timestamp         `json:"mtime,omitempty"`
	ContentType string             `json:"content_type,omitempty"`
	Metadata    ObjectUserMetadata `json:"metadata,omitempty"`
}
