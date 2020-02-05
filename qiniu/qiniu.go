package qiniu

import (
	"context"
	"fmt"
	"github.com/qiniu/api.v7/v7/auth"
	"github.com/qiniu/api.v7/v7/storage"
	"log"
	"path/filepath"
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
	log.Println("create a new token")
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
// 如果 upToken 为空, 则需要获取 upToken,
// 如果已存在 upToken, 但需要覆盖上传 且 KeyToOverwrite发生了变化, 则也要重新获取 upToken.
// TODO: 根据错误信息, 重新获取 upToken.
func (qn *Qiniu) Upload(localFile string, overwrite bool) (ret storage.PutRet, err error) {
	newKey := fmt.Sprintf("%s/%s", qn.folder, filepath.Base(localFile))
	oldKey := qn.KeyToOverwrite
	if overwrite {
		qn.KeyToOverwrite = newKey
	} else {
		qn.KeyToOverwrite = ""
	}
	if qn.upToken == "" {
		qn.createUpToken()
	} else if overwrite && (oldKey != newKey) {
		qn.createUpToken()
	}
	return qn.formUpload(localFile)
}
