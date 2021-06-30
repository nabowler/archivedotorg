package s3

import (
	"fmt"
	"testing"
)

func TestIdentifer(t *testing.T) {
	type testCase struct {
		UploadOptions
		Expected string
	}
	for i, tc := range []testCase{
		{
			UploadOptions: UploadOptions{
				Identifier: "foo",
				Title:      "bar",
			},
			Expected: "foo",
		},
		{
			UploadOptions: UploadOptions{
				Identifier: "FOO",
			},
			Expected: "FOO",
		},
		{
			UploadOptions: UploadOptions{
				Title: "bar",
			},
			Expected: "bar",
		},
		{
			UploadOptions: UploadOptions{
				Title: "BaZ",
			},
			Expected: "baz",
		},
		{
			UploadOptions: UploadOptions{
				Title: "Hello_there-How?Are! You???",
			},
			Expected: "hello-there-how-are--you---",
		},
	} {
		i := i
		tc := tc
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			if ident := tc.identifier(); ident != tc.Expected {
				t.Errorf("Expected %s. Got %s", tc.Expected, ident)
			}

		})
	}
}
