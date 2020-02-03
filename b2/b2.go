package b2

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// 参考: https://www.backblaze.com/b2/docs/b2_authorize_account.html
func authorizeAccount(id, key string) (*AuthResponse, error) {
	idKey := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", id, key)))
	authorization := fmt.Sprintf("Basic %s", idKey)
	req, err := http.NewRequest(
		http.MethodGet,
		"https://api.backblazeb2.com/b2api/v2/b2_authorize_account",
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", authorization)
	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	//noinspection GoUnhandledErrorResult
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		respErr := new(ResponseError)
		if err := json.Unmarshal(data, respErr); err != nil {
			return nil, err
		}
		return nil, respErr
	}
	auth := new(AuthResponse)
	if err := json.Unmarshal(data, auth); err != nil {
		return nil, err
	}
	return auth, nil
}

type AuthResponse struct {
	AccountId          string
	AuthorizationToken string
	ApiUrl             string
	DownloadUrl        string
	// 省略了一些我用不到的信息
	// 参考: https://www.backblaze.com/b2/docs/b2_authorize_account.html
}

type ResponseError struct {
	Status  int
	Code    string
	message string
}

func (err ResponseError) Error() string {
	return fmt.Sprintf("%d: %s", err.Status, err.Code)
}
