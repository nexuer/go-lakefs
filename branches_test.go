package lakefs

import (
	"context"
	"fmt"
	"testing"
)

func TestBranches_ListBranches(t *testing.T) {
	client := testClient()

	result, err := client.Branches.ListBranches(context.Background(), "quickstart", &ListBranchesOptions{
		//ListOptions: ListOptions{
		//	Amount: 1,
		//	After:  "main",
		//},
		//Prefix:     "no-resp",
		ShowHidden: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range result.Results {
		fmt.Println(v.ID, v.CommitID)
	}
}

func TestBranches_GetBranch(t *testing.T) {
	client := testClient()

	result, err := client.Branches.GetBranch(context.Background(), "quickstart", "main")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result)
}

func TestBranches_CreateBranch(t *testing.T) {
	client := testClient()

	result, err := client.Branches.CreateBranch(context.Background(), "quickstart", &BranchCreation{
		Name:   "features-v1",
		Source: "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result)
}

func TestBranches_DeleteBranch(t *testing.T) {
	client := testClient()

	err := client.Branches.DeleteBranch(context.Background(), "quickstart", "features-v1", &DeleteBranchOptions{
		Force: true,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBranches_Reset(t *testing.T) {
	client := testClient()

	err := client.Branches.Reset(context.Background(), "quickstart", "features-v1", &ResetCreation{
		Type: ResetResetType,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBranches_Revert(t *testing.T) {
	client := testClient()

	err := client.Branches.Revert(context.Background(), "quickstart", "features-v1", &RevertCreation{})
	if err != nil {
		t.Fatal(err)

	}
}

func TestBranches_CherryPick(t *testing.T) {
	client := testClient()

	err := client.Branches.CherryPick(context.Background(), "quickstart", "features-v1", &CherryPickCreation{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBranches_Diff(t *testing.T) {
	client := testClient()

	result, err := client.Branches.Diff(context.Background(), "quickstart", "features-v1", &DiffOptions{
		//Delimiter: "/",
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result)
}
