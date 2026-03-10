package lakefs

import (
	"context"
	"fmt"
	"testing"
)

func TestRefs_ListCommits(t *testing.T) {
	client := testClient()

	result, err := client.Refs.ListCommits(context.Background(), "quickstart", "main", &ListCommitsOptions{
		//Delimiter: "/",
		//Since: time.Now().Format(time.RFC3339),
		Objects: []string{
			//"refs/heads/master",
			//"refs/tags/v1.0.0",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result)
}
