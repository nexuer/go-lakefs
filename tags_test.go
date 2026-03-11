package lakefs

import (
	"context"
	"fmt"
	"testing"
)

func TestTags_ListTags(t *testing.T) {
	client := testClient()

	result, err := client.Tags.ListTags(context.Background(), "quickstart", &ListTagsOptions{
		//ListOptions: ListOptions{
		//	Amount: 1,
		//	After:  "quickstart",
		//},
		//Prefix: "no-resp",
		//Search: "quickstart",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range result.Results {
		fmt.Println(v.ID, v.CommitID)
	}
}
