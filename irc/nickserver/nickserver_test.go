package nickserver

import (
	"testing"
)

func TestGetUrl(t *testing.T) {
	c, err := NewNickServerClient("http://127.0.0.1:8118", nil)
	if err != nil {
		t.Fatal(err)
	}

	url, err := c.getUrl("nicks")
	if err != nil {
		t.Fatal(err)
	}

	expectedUrl := "http://127.0.0.1:8118/nicks"
	if url != expectedUrl {
		t.Fatalf("got %s but expected %s", url, expectedUrl)
	}
}
