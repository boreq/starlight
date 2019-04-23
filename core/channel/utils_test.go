package channel

import (
	"testing"
)

func TestValidateIdTooLong(t *testing.T) {
	channelId := make([]byte, IdLength+1)
	if result := ValidateId(channelId); result != false {
		t.Fatalf("id should be too long")
	}
}

func TestValidateId(t *testing.T) {
	channelId := make([]byte, IdLength)
	if result := ValidateId(channelId); result != true {
		t.Fatalf("id should be valid")
	}
}
