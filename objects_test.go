package lakefs

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestObjects_CreateObject(t *testing.T) {
	client := testClient(true)

	path := "./go.mod"
	repoPath := "go.mod"

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	obj, err := client.Objects.CreateObject(ctx, "quickstart", "main", file, &CreateObjectOptions{
		Path: repoPath,
		//S3UploadOptions: &S3UploadOptions{
		//	Concurrency:   10,
		//	PartSizeBytes: 1024 * 1024 * 10,
		//},
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(obj)

	file2, err := os.Open("./go.sum")
	if err != nil {
		t.Fatal(err)
	}
	defer file2.Close()
	obj, err = client.Objects.CreateObject(context.Background(), "quickstart", "main", file2, &CreateObjectOptions{
		Path: "subdir/go.sum",
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(obj)
}

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
	client := testClient(true)

	path := "go.mod"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	object, err := client.Objects.GetObjectContent(ctx, "quickstart", "main", &GetObjectContentOptions{
		Path: path,
		//Range: &RangeByteSize{
		//	Start: 0,
		//	End:   10,
		//},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer object.Body.Close()
	scanner := bufio.NewScanner(object.Body)
	fmt.Printf("---------------------------- %s ----------------------------\n", path)
	for scanner.Scan() {
		//line := scanner.Text()
		//fmt.Println(line)
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

func TestObjects_GetObjectMetadata(t *testing.T) {
	client := testClient()
	//presign := true
	object, err := client.Objects.GetObjectMetadata(context.Background(), "ua-1715352869", "staging-ds", &GetObjectMetadataOptions{
		Path: "Trainingdata.jsonl.1773628775650",
		//Presign: &presign,
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Metadata: %+v\n", object)
}
