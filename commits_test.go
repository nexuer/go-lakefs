package lakefs

import (
	"context"
	"fmt"
	"testing"
)

func TestCommits_GetCommit(t *testing.T) {
	client := testClient()

	result, err := client.Commits.GetCommit(context.Background(), "quickstart", "features-v1")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result)
}

func TestCommits_CreateCommit(t *testing.T) {
	client := testClient()

	result, err := client.Commits.CreateCommit(context.Background(), "quickstart", "features-v1", &CommitCreation{
		Message: "Hello World",
		//Date:    NewTimestamp(time.Now()),
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result)
}
