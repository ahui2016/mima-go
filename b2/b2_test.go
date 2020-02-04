package b2

import (
	"errors"
	"testing"
)

var userInput = map[string]string{
	"id" : "000b49aa41a684d0000000005",
	"key" : "K000m0AMjjIam+cYXU6oZYw3LOpI9eI",
	"bucket" : "9b24c95abab4018a7608041d",
}

// go test -v -run TestAuthorizeAccount
func TestAuthorizeAccount(t *testing.T) {
	auth, err := AuthorizeAccount(userInput["id"], userInput["key"])
	var respErr ResponseError
	if errors.As(err, &respErr) {
		t.Fatal("ResponseError:", respErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Log(*auth)
}

func TestGetUploadUrl(t *testing.T) {
	BucketId = userInput["bucket"]
	auth, err := AuthorizeAccount(userInput["id"], userInput["key"])
	if err != nil {
		t.Fatal(err)
	}
	uploadUrlResp, err := GetUploadUrl(auth)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(*uploadUrlResp)
}