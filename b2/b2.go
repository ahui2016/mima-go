package b2

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

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

type Bucket struct {
	id     string
	appId  string
	appKey string
	auth   *AuthResponse
}

func NewBucket(id, appId, appKey string) *Bucket {
	return &Bucket{id, appId, appKey, nil}
}

func (b *Bucket) String() string {
	return fmt.Sprintf("appId: %s, appKey: %s", b.appId, b.appKey)
}

// 参考: https://www.backblaze.com/b2/docs/b2_authorize_account.html
func (b *Bucket) AuthorizeAccount() error {
	idKey := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", b.appId, b.appKey)))
	resp, err := makeRequest(
		http.MethodGet,
		"https://api.backblazeb2.com/b2api/v2/b2_authorize_account",
		nil,
		map[string]string{"Authorization": fmt.Sprintf("Basic %s", idKey)},
	)
	if err != nil {
		return err
	}
	auth := new(AuthResponse)
	if err := json.Unmarshal(resp, auth); err != nil {
		return err
	}
	b.auth = auth
	return nil
}

func (b *Bucket) GetUploadUrl() (*UploadUrl, error) {
	if b.auth == nil {
		return nil, errors.New("unauthorized")
	}
	body, err := json.Marshal(map[string]string{"bucketId": b.id})
	if err != nil {
		return nil, err
	}
	resp, err := makeRequest(
		http.MethodPost,
		fmt.Sprintf("%s/b2api/v2/b2_get_upload_url", b.auth.ApiUrl),
		bytes.NewReader(body),
		map[string]string{"Authorization": b.auth.AuthorizationToken},
	)
	uploadUrl := new(UploadUrl)
	if err := json.Unmarshal(resp, uploadUrl); err != nil {
		return nil, err
	}
	return uploadUrl, nil
}

type UploadUrl struct {
	BucketId           string
	UploadUrl          string
	AuthorizationToken string
}

type UploadResponse struct {
	AccountId       string
	Action          string
	BucketId        string
	ContentSha1     string
	ContentType     string
	FileId          string
	FileName        string
	UploadTimestamp int64
}

/*
func (uu *UploadUrl) UploadFile(filePath string, uploadUrl *UploadUrl) (*UploadResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	data, err := makeRequest(
		http.MethodPost,
		uploadUrl.UploadUrl,
		file,
		map[string]string{
			"Authorization": uploadUrl.AuthorizationToken,
			"X-Bz-File-Name":
		}
	)
}
*/
func makeRequest(method, urlPath string, body io.Reader, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(method, urlPath, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
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
	return data, nil
}
