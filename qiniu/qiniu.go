package qiniu

import (
	"context"
	"errors"
	"fmt"
	"github.com/qiniu/api.v7/v7/auth"
	"github.com/qiniu/api.v7/v7/storage"
	"path/filepath"
)

var (
	ErrBlankUpToken = errors.New("未生成 upToken")
)

type Qiniu struct {
	accessKey      string
	secretKey      string
	bucket         string
	folder         string
	zone           *storage.Region
	upToken        string
	KeyToOverwrite string
}

func NewQiniu(accessKey, secretKey, bucket, folder string, zone *storage.Region) *Qiniu {
	return &Qiniu{
		accessKey: accessKey,
		secretKey: secretKey,
		bucket:    bucket,
		folder:    folder,
		zone:      zone,
	}
}

func (qn *Qiniu) createUpToken() {
	putPolicy := storage.PutPolicy{}
	if qn.KeyToOverwrite == "" {
		putPolicy.Scope = qn.bucket
	} else {
		putPolicy.Scope = fmt.Sprintf("%s:%s", qn.bucket, qn.KeyToOverwrite)
	}
	qn.upToken = putPolicy.UploadToken(
		auth.New(qn.accessKey, qn.secretKey),
	)
}

func (qn *Qiniu) formUpload(localFile string) (ret storage.PutRet, err error) {
	cfg := storage.Config{
		Zone:     qn.zone,
		UseHTTPS: false,
	}
	formUploader := storage.NewFormUploader(&cfg)
	err = formUploader.PutFile(
		context.Background(),
		&ret,
		qn.upToken,
		fmt.Sprintf("%s/%s", qn.folder, filepath.Base(localFile)),
		localFile,
		nil,
	)
	return
}

// Upload 需要先获取 upToken, 而在获取 upToken 之前要先设置 KeyToOverwrite.
// TODO: 根据错误信息, 重新获取 upToken.
func (qn *Qiniu) Upload(localFile string, overwrite bool) (ret storage.PutRet, err error) {
	if overwrite {
		qn.KeyToOverwrite = fmt.Sprintf("%s/%s", qn.folder, filepath.Base(localFile))
	} else {
		qn.KeyToOverwrite = ""
	}
	if qn.upToken == "" {
		qn.createUpToken()
	}
	return qn.formUpload(localFile)
}
