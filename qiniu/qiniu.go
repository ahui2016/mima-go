package qiniu

import (
	"github.com/qiniu/api.v7/v7/auth"
	"github.com/qiniu/api.v7/v7/storage"
)

type Qiniu struct {
	accessKey string
	secretKey string
	bucket    string
}

func NewQiniu(accessKey, secretKey, bucket string) *Qiniu {
	return &Qiniu{accessKey, secretKey, bucket}
}

func (qn *Qiniu) GetUpToken() string {
	putPolicy := storage.PutPolicy{Scope: qn.bucket}
	return putPolicy.UploadToken(
		auth.New(qn.accessKey, qn.secretKey),
	)
}
