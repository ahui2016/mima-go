package b2

import (
	"errors"
	"testing"
)

func TestAuthorizeAccount(t *testing.T) {
	id := "000b49aa41a684d0000000005"
	key := "K000m0AMjjIam+cYXU6oZYw3LOpI9eI"
	auth, err := authorizeAccount(id, key)
	var respErr ResponseError
	if errors.As(err, &respErr) {
		t.Log("ResponseError:", respErr)
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Log(*auth)
}
