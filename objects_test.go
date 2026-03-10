package lakefs

import (
	"bufio"
	"context"
	"fmt"
	"testing"
)

func TestObjects_ListObjects(t *testing.T) {
	client := testClient()

	objects, err := client.Objects.ListObjects(context.Background(), "quickstart", "main", &ListObjectOptions{
		//Prefix: "data/",
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(objects)
}

func TestObjects_GetObjectContent(t *testing.T) {
	client := testClient()

	object, err := client.Objects.GetObjectContent(context.Background(), "quickstart", "main", &GetObjectContentOptions{
		Path: "README.md",
		Range: &RangeByteSize{
			Start: 0,
			End:   10,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer object.Body.Close()
	scanner := bufio.NewScanner(object.Body)
	fmt.Println("---------------------------- README.md ----------------------------")
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
	if scanner.Err() != nil {
		t.Fatal(scanner.Err())
	}
}

func TestObjects_ObjectExists(t *testing.T) {
	client := testClient()

	exists, headers, err := client.Objects.ObjectExists(context.Background(), "quickstart", "main", &ObjectExistsOptions{
		Path: "README.md",
		//Range: &RangeByteSize{
		//	Start: 0,
		//	End:   10,
		//},
	})

	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Exists: %t\n", exists)
	fmt.Printf("Headers: %+v\n", headers)
}
