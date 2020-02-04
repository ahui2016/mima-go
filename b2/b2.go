package b2

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var BucketId string

type ResponseError struct {
	Status  int
	Code    string
	message string
}

func (err ResponseError) Error() string {
	return fmt.Sprintf("%d: %s", err.Status, err.Code)
}

type AuthResponse struct {
	AccountId          string
	AuthorizationToken string
	ApiUrl             string
	DownloadUrl        string
	// 省略了一些我用不到的信息
	// 参考: https://www.backblaze.com/b2/docs/b2_authorize_account.html
}

// 参考: https://www.backblaze.com/b2/docs/b2_authorize_account.html
func AuthorizeAccount(id, key string) (*AuthResponse, error) {
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
	resp, err := new(http.Client).Do(req)
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

type UploadUrlResponse struct {
	BucketId           string
	UploadUrl          string
	AuthorizationToken string
}

type UploadUrlBody struct {
	BucketId string `json:"bucketId"`
}

func GetUploadUrl(auth *AuthResponse) (*UploadUrlResponse, error) {
	body, err := json.Marshal(UploadUrlBody{BucketId: BucketId})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/b2api/v2/b2_get_upload_url", auth.ApiUrl),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth.AuthorizationToken)
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
	uploadUrlResp := new(UploadUrlResponse)
	if err := json.Unmarshal(data, uploadUrlResp); err != nil {
		return nil, err
	}
	return uploadUrlResp, nil
}
