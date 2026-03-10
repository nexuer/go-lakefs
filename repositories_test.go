package lakefs

import (
	"context"
	"fmt"
	"testing"
)

func TestRepositories_ListRepositories(t *testing.T) {
	client := testClient()

	result, err := client.Repositories.ListRepositories(context.Background(), &ListRepositoriesOptions{
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
		fmt.Println(v.ID, v.CreationDate)
	}
}

func TestRepositories_GetRepository(t *testing.T) {
	client := testClient()

	result, err := client.Repositories.GetRepository(context.Background(), "quickstart")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result.ID, result.CreationDate)
}

func TestRepositories_CreateRepository(t *testing.T) {
	client := testClient()

	result, err := client.Repositories.CreateRepository(context.Background(), &RepositoryCreation{
		Name:             "quickstart-1",
		StorageNamespace: "local://quickstart-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result.ID, result.CreationDate)
}

func TestRepositories_CreateBaseRepository(t *testing.T) {
	client := testClient()

	result, err := client.Repositories.CreateBareRepository(context.Background(), &RepositoryCreation{
		Name:             "quickstart-2",
		StorageNamespace: "local://quickstart-2",
		SampleData:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result.ID, result.CreationDate)
}

func TestRepositories_DeleteRepository(t *testing.T) {
	client := testClient()

	err := client.Repositories.DeleteRepository(context.Background(), "quickstart-1", &DeleteRepositoryOptions{
		//Force: true,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRepositories_GetRepositoryMetadata(t *testing.T) {
	client := testClient()

	result, err := client.Repositories.GetRepositoryMetadata(context.Background(), "quickstart")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Metadata: %+v", result)
}

func TestRepositories_GetRepositoryGCRules(t *testing.T) {
	client := testClient()

	result, err := client.Repositories.GetRepositoryGCRules(context.Background(), "quickstart")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("GC Rules: %+v", result)
}

func TestRepositories_CreateRepositoryGCRules(t *testing.T) {
	client := testClient()

	err := client.Repositories.PutRepositoryGCRules(context.Background(), "quickstart", &RepositoryGCRules{
		DefaultRetentionDays: 20,
		Branches: []*BranchGCRule{
			{
				BranchID:      "main",
				RetentionDays: 20,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Repositories.GetRepositoryGCRules(context.Background(), "quickstart")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("GC Rules: %+v", result)
}

func TestRepositories_DeleteRepositoryGCRules(t *testing.T) {
	client := testClient()

	err := client.Repositories.DeleteRepositoryGCRules(context.Background(), "quickstart")
	if err != nil {
		t.Fatal(err)
	}
}

func TestRepositories_GetBranchProtectionRules(t *testing.T) {
	client := testClient()
	result, err := client.Repositories.GetBranchProtectionRules(context.Background(), "quickstart")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Branch Protection Rules: %+v", result)
}

func TestRepositories_PutBranchProtectionRules(t *testing.T) {
	client := testClient()

	err := client.Repositories.PutBranchProtectionRules(context.Background(), "quickstart", &PutBranchProtectionRulesOptions{
		//IfMatch: "main",
		Rules: []*BranchProtectionRule{
			{
				Pattern: "main",
			},
			//{
			//	Pattern: "main-*",
			//},
			//{
			//	Pattern: "main/*",
			//},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
