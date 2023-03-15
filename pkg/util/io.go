package util

import (
	"io"

	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func ReadAllAndClose(t test.TestHelper, in io.ReadCloser) []byte {
	defer func() {
		if err := in.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	body, err := io.ReadAll(in)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	return body
}
