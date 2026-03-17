package lakefs

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"testing"
)

func TestS3Client_CreateObject(t *testing.T) {
	c, err := newS3Client(testCredential())
	if err != nil {
		t.Fatal(err)
	}

	file, err := os.Open("./go.mod")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	err = c.CreateObject(context.Background(), "quickstart", "main", file, &CreateObjectOptions{
		Path: "go.mod",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestS3Client_GetObject(t *testing.T) {
	c, err := newS3Client(testCredential())
	if err != nil {
		t.Fatal(err)
	}
	path := "go.mod"

	object, err := c.GetObjectContent(context.Background(), "quickstart", "main", &GetObjectContentOptions{
		Path: path,
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
	fmt.Printf("---------------------------- %s ----------------------------\n", path)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
	if scanner.Err() != nil {
		t.Fatal(scanner.Err())
	}
}
