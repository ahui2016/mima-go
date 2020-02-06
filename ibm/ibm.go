package ibm

import (
	"fmt"
	"github.com/IBM/ibm-cos-sdk-go/aws"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials/ibmiam"
	"github.com/IBM/ibm-cos-sdk-go/aws/session"
	"github.com/IBM/ibm-cos-sdk-go/service/s3"
	"os"
	"path/filepath"
)

type COS struct {
	apiKey            string
	serviceInstanceID string // resource_instance_id
	authEndpoint      string
	serviceEndpoint   string
	bucketLocation    string
	bucketName        string

	// Object keys can be up to 1024 characters in length, and it's best to avoid
	// any characters that might be problematic in a web address. For example, ?, =, <,
	// and other special characters might cause unwanted behavior if not URL-encoded.
	objKeyPrefix string // 用半角括号括住并加上横杠, 详见 COS.makeObjKey

	conf *aws.Config
}

func NewCOS(apiKey, serInsID, authEP, serEP, bucLoc, bucName, prefix string) *COS {
	return &COS{
		apiKey:            apiKey,
		serviceInstanceID: serInsID,
		authEndpoint:      authEP,
		serviceEndpoint:   serEP,
		bucketLocation:    bucLoc,
		bucketName:        bucName,
		objKeyPrefix:      prefix,
	}
}

func (cos *COS) makeConfig() {
	cos.conf = aws.NewConfig().
		//WithRegion("ap-east-1").
		WithEndpoint(cos.serviceEndpoint).
		WithCredentials(ibmiam.NewStaticCredentials(
			aws.NewConfig(), cos.authEndpoint, cos.apiKey, cos.serviceInstanceID)).
		WithS3ForcePathStyle(true)
}

func (cos *COS) makeObjKey(name string) (objectKeyWithPrefix string) {
	return fmt.Sprintf("(%s)-%s", cos.objKeyPrefix, name)
}

func (cos *COS) uploadFile(localFile string) (*s3.PutObjectOutput, error) {
	file, err := os.Open(localFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sess := session.Must(session.NewSession())
	client := s3.New(sess, cos.conf)

	input := s3.PutObjectInput{
		Bucket: aws.String(cos.bucketName),
		Key:    aws.String(cos.makeObjKey(filepath.Base(localFile))),
		Body:   file,
	}
	return client.PutObject(&input)
}

func (cos *COS) UploadFile(localFile string) (*s3.PutObjectOutput, error) {
	if cos.conf == nil {
		cos.makeConfig()
	}
	return cos.uploadFile(localFile)
}
