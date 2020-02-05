package b2

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

type ResponseError struct {
	Status  int
	Code    string
	Message string
}

func (err ResponseError) Error() string {
	return fmt.Sprintf("%d: %s: %s", err.Status, err.Code, err.Message)
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
	folder string
	appId  string
	appKey string
	auth   *AuthResponse
}

func NewBucket(id, folder, appId, appKey string) *Bucket {
	return &Bucket{id, folder, appId, appKey, nil}
}

func (b *Bucket) String() string {
	return fmt.Sprintf("appId: %s, appKey: %s", b.appId, b.appKey)
}

func (b *Bucket) Folder() string {
	return b.folder
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
	if err != nil {
		return nil, err
	}
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

func (uu *UploadUrl) UploadFile(filePath, folder string) (*UploadResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}
	//noinspection GoUnhandledErrorResult
	defer file.Close()
	headers := map[string]string{
		"Content-Length": fmt.Sprintf("%d", fileInfo.Size()),
		//"Authorization": uu.AuthorizationToken,
		"X-Bz-File-Name": fmt.Sprintf("%s/%s", folder, url.QueryEscape("test test+ok.txt")),
		"Content-Type": "b2/x-auto",
		"X-Bz-Content-Sha1": fmt.Sprintf("%x", hash.Sum(nil)),
	}
	resp, err := makeRequest(http.MethodPost, uu.UploadUrl, file, headers)
	if err != nil {
		return nil, err
	}
	uploadResponse := new(UploadResponse)
	if err := json.Unmarshal(resp, uploadResponse); err != nil {
		return nil, err
	}
	return uploadResponse, nil
}

func makeRequest(method, urlPath string, body io.Reader, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(method, urlPath, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := new(http.Transport).RoundTrip(req)
	if err != nil {
		return nil, err
	}
	log.Println("length:", req.Header.Get("Content-Length"))
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
