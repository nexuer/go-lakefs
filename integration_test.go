package lakefs

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLakeFSIntegration(t *testing.T) {
	client := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	repositoryID := fmt.Sprintf("go-lakefs-it-%d", time.Now().UnixNano())
	repository, err := client.Repositories.CreateRepository(ctx, &RepositoryCreation{
		Name:             repositoryID,
		DefaultBranch:    "main",
		StorageNamespace: integrationStorageNamespace(repositoryID),
	})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := client.Repositories.DeleteRepository(cleanupCtx, repositoryID, &DeleteRepositoryOptions{Force: true}); err != nil && !IsNotFound(err) {
			t.Errorf("cleanup repository: %v", err)
		}
	})
	if repository.ID != repositoryID || repository.DefaultBranch != "main" {
		t.Fatalf("unexpected repository: %+v", repository)
	}

	t.Run("repositories", func(t *testing.T) {
		got, err := client.Repositories.GetRepository(ctx, repositoryID)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != repositoryID {
			t.Fatalf("repository ID = %q", got.ID)
		}
		records, err := client.Repositories.ListRepositories(ctx, &ListRepositoriesOptions{Prefix: repositoryID})
		if err != nil {
			t.Fatal(err)
		}
		if !containsRepository(records.Results, repositoryID) {
			t.Fatalf("repository %q not found in list", repositoryID)
		}
		if _, err := client.Repositories.GetRepositoryMetadata(ctx, repositoryID); err != nil {
			t.Fatal(err)
		}

		rules := &RepositoryGCRules{
			DefaultRetentionDays: 7,
			Branches:             []*BranchGCRule{{BranchID: "main", RetentionDays: 3}},
		}
		if err := client.Repositories.PutRepositoryGCRules(ctx, repositoryID, rules); err != nil {
			t.Fatal(err)
		}
		gotRules, err := client.Repositories.GetRepositoryGCRules(ctx, repositoryID)
		if err != nil {
			t.Fatal(err)
		}
		if gotRules.DefaultRetentionDays != rules.DefaultRetentionDays {
			t.Fatalf("GC rules = %+v", gotRules)
		}
		if err := client.Repositories.DeleteRepositoryGCRules(ctx, repositoryID); err != nil {
			t.Fatal(err)
		}

		protection := &PutBranchProtectionRulesOptions{Rules: []*BranchProtectionRule{{Pattern: "release/*"}}}
		if err := client.Repositories.PutBranchProtectionRules(ctx, repositoryID, protection); err != nil {
			t.Fatal(err)
		}
		gotProtection, err := client.Repositories.GetBranchProtectionRules(ctx, repositoryID)
		if err != nil {
			t.Fatal(err)
		}
		if len(gotProtection) != 1 || gotProtection[0].Pattern != "release/*" {
			t.Fatalf("branch protection = %+v", gotProtection)
		}

		bareID := repositoryID + "-bare"
		bare, err := client.Repositories.CreateBareRepository(ctx, &RepositoryCreation{
			Name:             bareID,
			DefaultBranch:    "main",
			StorageNamespace: integrationStorageNamespace(bareID),
		})
		if err != nil {
			t.Fatal(err)
		}
		if bare.ID != bareID {
			t.Fatalf("bare repository = %+v", bare)
		}
		if err := client.Repositories.DeleteRepository(ctx, bareID, &DeleteRepositoryOptions{Force: true}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("branches", func(t *testing.T) {
		branch, err := client.Branches.GetBranch(ctx, repositoryID, "main")
		if err != nil {
			t.Fatal(err)
		}
		if branch.ID != "main" {
			t.Fatalf("branch = %+v", branch)
		}
		branches, err := client.Branches.ListBranches(ctx, repositoryID, &ListBranchesOptions{ShowHidden: true})
		if err != nil {
			t.Fatal(err)
		}
		if !containsRef(branches.Results, "main") {
			t.Fatal("main branch not found")
		}
	})

	const objectPath = "sdk/input.txt"
	const objectBody = "hello from go-lakefs integration test"
	object, err := client.Objects.CreateObject(ctx, repositoryID, "main", strings.NewReader(objectBody), &CreateObjectOptions{
		Path: objectPath, ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("create object: %v", err)
	}
	if object.Path != objectPath || object.SizeBytes != int64(len(objectBody)) {
		t.Fatalf("unexpected object: %+v", object)
	}

	t.Run("objects", func(t *testing.T) {
		exists, headers, err := client.Objects.ObjectExists(ctx, repositoryID, "main", &ObjectExistsOptions{
			Path: objectPath, Range: &RangeByteSize{Start: 0, End: 4},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !exists || headers == nil || headers.ContentLength != 5 {
			t.Fatalf("exists = %v, headers = %+v", exists, headers)
		}
		listed, err := client.Objects.ListObjects(ctx, repositoryID, "main", &ListObjectOptions{Prefix: "sdk/"})
		if err != nil {
			t.Fatal(err)
		}
		if !containsObject(listed.Results, objectPath) {
			t.Fatalf("object %q not found in list", objectPath)
		}
		metadata, err := client.Objects.GetObjectMetadata(ctx, repositoryID, "main", &GetObjectMetadataOptions{Path: objectPath})
		if err != nil {
			t.Fatal(err)
		}
		if metadata.Path != objectPath {
			t.Fatalf("metadata = %+v", metadata)
		}
		content, err := client.Objects.GetObjectContent(ctx, repositoryID, "main", &GetObjectContentOptions{
			Path: objectPath, Range: &RangeByteSize{Start: 0, End: 4},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer content.Body.Close()
		data, err := io.ReadAll(content.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "hello" {
			t.Fatalf("range content = %q", data)
		}

		copied, err := client.Objects.CopyObject(ctx, repositoryID, "main", "sdk/copied.txt", &ObjectCopyCreation{
			SrcPath: objectPath, SrcRef: "main",
		})
		if err != nil {
			t.Fatal(err)
		}
		if copied.Path != "sdk/copied.txt" {
			t.Fatalf("copied object = %+v", copied)
		}
		if err := client.Objects.RewriteAllObjectMetadata(ctx, repositoryID, "main", &RewriteAllObjectMetadataOptions{
			Path: "sdk/copied.txt", Set: ObjectUserMetadata{"source": "integration"},
		}); err != nil {
			t.Fatal(err)
		}
		if err := client.Objects.DeleteObject(ctx, repositoryID, "main", &DeleteObjectOptions{Path: "sdk/copied.txt"}); err != nil {
			t.Fatal(err)
		}
	})

	commit, err := client.Commits.CreateCommit(ctx, repositoryID, "main", &CommitCreation{
		Message: "integration commit", Metadata: map[string]string{"client": "go-lakefs"},
	})
	if err != nil {
		t.Fatalf("create commit: %v", err)
	}
	if commit.ID == "" {
		t.Fatal("commit ID is empty")
	}

	t.Run("commits refs and tags", func(t *testing.T) {
		got, err := client.Commits.GetCommit(ctx, repositoryID, commit.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != commit.ID {
			t.Fatalf("commit = %+v", got)
		}
		commits, err := client.Refs.ListCommits(ctx, repositoryID, "main", &ListCommitsOptions{ListOptions: ListOptions{Amount: 10}})
		if err != nil {
			t.Fatal(err)
		}
		if !containsCommit(commits.Results, commit.ID) {
			t.Fatalf("commit %q not found in ref log", commit.ID)
		}
		tag, err := client.Tags.CreateTag(ctx, repositoryID, &TagCreation{ID: "integration", Ref: commit.ID})
		if err != nil {
			t.Fatal(err)
		}
		if tag.ID != "integration" {
			t.Fatalf("tag = %+v", tag)
		}
		gotTag, err := client.Tags.GetTag(ctx, repositoryID, "integration")
		if err != nil {
			t.Fatal(err)
		}
		if gotTag.ID != "integration" {
			t.Fatalf("tag = %+v", gotTag)
		}
		tags, err := client.Tags.ListTags(ctx, repositoryID, &ListTagsOptions{Prefix: "integ"})
		if err != nil {
			t.Fatal(err)
		}
		if !containsRef(tags.Results, "integration") {
			t.Fatal("integration tag not found")
		}
		if err := client.Tags.DeleteTag(ctx, repositoryID, "integration"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("branch changes", func(t *testing.T) {
		branch, err := client.Branches.CreateBranch(ctx, repositoryID, &BranchCreation{Name: "feature", Source: "main"})
		if err != nil {
			t.Fatal(err)
		}
		if branch.ID != "feature" {
			t.Fatalf("branch = %+v", branch)
		}
		if _, err := client.Objects.CreateObject(ctx, repositoryID, "feature", strings.NewReader("feature"), &CreateObjectOptions{Path: "sdk/feature.txt"}); err != nil {
			t.Fatal(err)
		}
		diff, err := client.Branches.Diff(ctx, repositoryID, "feature", &DiffOptions{Prefix: "sdk/"})
		if err != nil {
			t.Fatal(err)
		}
		if len(diff.Results) == 0 {
			t.Fatal("branch diff is empty after object creation")
		}
		if err := client.Branches.Reset(ctx, repositoryID, "feature", &ResetCreation{Type: ObjectResetType, Path: "sdk/feature.txt"}); err != nil {
			t.Fatal(err)
		}
		if err := client.Branches.DeleteBranch(ctx, repositoryID, "feature", &DeleteBranchOptions{Force: true}); err != nil {
			t.Fatal(err)
		}

		if _, err := client.Branches.CreateBranch(ctx, repositoryID, &BranchCreation{Name: "source", Source: "main"}); err != nil {
			t.Fatal(err)
		}
		if _, err := client.Objects.CreateObject(ctx, repositoryID, "source", strings.NewReader("source"), &CreateObjectOptions{Path: "sdk/source.txt"}); err != nil {
			t.Fatal(err)
		}
		sourceCommit, err := client.Commits.CreateCommit(ctx, repositoryID, "source", &CommitCreation{Message: "source commit"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.Branches.CreateBranch(ctx, repositoryID, &BranchCreation{Name: "target", Source: "main"}); err != nil {
			t.Fatal(err)
		}
		if err := client.Branches.CherryPick(ctx, repositoryID, "target", &CherryPickCreation{Ref: sourceCommit.ID}); err != nil {
			t.Fatal(err)
		}
		if err := client.Branches.Revert(ctx, repositoryID, "target", &RevertCreation{Ref: sourceCommit.ID, ParentNumber: 1}); err != nil {
			t.Fatal(err)
		}
		if err := client.Branches.DeleteBranch(ctx, repositoryID, "target", &DeleteBranchOptions{Force: true}); err != nil {
			t.Fatal(err)
		}
		if err := client.Branches.DeleteBranch(ctx, repositoryID, "source", &DeleteBranchOptions{Force: true}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("directory transfer", func(t *testing.T) {
		source := t.TempDir()
		if err := os.WriteFile(filepath.Join(source, "one.txt"), []byte("one"), 0o600); err != nil {
			t.Fatal(err)
		}
		uploaded, err := client.Objects.UploadDirectory(ctx, repositoryID, "main", &UploadDirectoryOptions{
			Source: source, Path: "batch", Recursive: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if uploaded.ObjectsUploaded != 1 {
			t.Fatalf("upload stats = %+v", uploaded)
		}
		destination := t.TempDir()
		downloaded, err := client.Objects.DownloadDirectory(ctx, repositoryID, "main", &DownloadDirectoryOptions{
			Path: "batch", Destination: destination,
		})
		if err != nil {
			t.Fatal(err)
		}
		if downloaded.ObjectsDownloaded != 1 {
			t.Fatalf("download stats = %+v", downloaded)
		}
		deleted, err := client.Objects.DeleteDirectory(ctx, repositoryID, "main", &DeleteDirectoryOptions{Path: "batch"})
		if err != nil {
			t.Fatal(err)
		}
		if deleted.ObjectsDeleted != 1 {
			t.Fatalf("delete stats = %+v", deleted)
		}
	})

	t.Run("staging", func(t *testing.T) {
		const path = "sdk/staged.txt"
		const body = "staged"
		location, err := client.Staging.GetBacking(ctx, repositoryID, "main", &GetBackingOptions{Path: path, Presign: true})
		if err != nil {
			t.Fatal(err)
		}
		if location.PhysicalAddress == "" || location.PresignedUrl == "" {
			t.Fatalf("staging location = %+v", location)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, location.PresignedUrl, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("upload staging content: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			t.Fatalf("upload staging content status = %d", resp.StatusCode)
		}
		checksum := strings.Trim(resp.Header.Get("ETag"), `"`)
		if checksum == "" {
			sum := md5.Sum([]byte(body))
			checksum = hex.EncodeToString(sum[:])
		}
		object, err := client.Staging.PutBacking(ctx, repositoryID, "main", &PutBackingOptions{
			IfNoneMatch: "*",
			Path:        path,
			StagingMetadata: &StagingMetadata{
				Staging: location, Checksum: checksum, SizeBytes: int64(len(body)), UserMetadata: map[string]string{}, ContentType: "text/plain", Mtime: NewTimestamp(time.Now()),
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if object.Path != path {
			t.Fatalf("staged object = %+v", object)
		}
	})
}

func integrationStorageNamespace(repositoryID string) string {
	base := os.Getenv("LAKEFS_STORAGE_NAMESPACE")
	if base == "" {
		base = "s3://lakefs-data"
	}
	return strings.TrimRight(base, "/") + "/" + repositoryID
}

func containsRepository(records []*Repository, id string) bool {
	for _, record := range records {
		if record != nil && record.ID == id {
			return true
		}
	}
	return false
}

func containsRef(records []*Ref, id string) bool {
	for _, record := range records {
		if record != nil && record.ID == id {
			return true
		}
	}
	return false
}

func containsObject(records []*ObjectStats, path string) bool {
	for _, record := range records {
		if record != nil && record.Path == path {
			return true
		}
	}
	return false
}

func containsCommit(records []*Commit, id string) bool {
	for _, record := range records {
		if record != nil && record.ID == id {
			return true
		}
	}
	return false
}
