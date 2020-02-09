package ibm

import (
	"fmt"
	"github.com/IBM/ibm-cos-sdk-go/aws"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials/ibmiam"
	"github.com/IBM/ibm-cos-sdk-go/aws/session"
	"github.com/IBM/ibm-cos-sdk-go/service/s3"
	"github.com/IBM/ibm-cos-sdk-go/service/s3/s3manager"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

const authEndpoint = "https://iam.cloud.ibm.com/identity/token"

type COS struct {
	apiKey            string
	serviceInstanceID string // resource_instance_id
	authEndpoint      string
	serviceEndpoint   string
	bucketLocation    string
	bucketName        string

	// Object keys can be up to 1024 characters in length, and it's best to avoid
	// any characters that might be problematic in a web address. For example, ?, =, <,
	// or other special characters might cause unwanted behavior if not URL-encoded.
	objKeyPrefix string // 用半角括号括住, 详见 COS.MakeObjKey

	conf   *aws.Config
	client *s3.S3
}

func NewCOS(apiKey, serInsID, serEP, bucLoc, bucName, prefix string) *COS {
	return &COS{
		apiKey:            apiKey,
		serviceInstanceID: serInsID,
		authEndpoint:      authEndpoint,
		serviceEndpoint:   serEP,
		bucketLocation:    bucLoc,
		bucketName:        bucName,
		objKeyPrefix:      prefix,
	}
}

func (cos *COS) makeConfig() {
	log.Println("making config...")
	cos.conf = aws.NewConfig().
		WithEndpoint(cos.serviceEndpoint).
		WithCredentials(ibmiam.NewStaticCredentials(
			aws.NewConfig(), cos.authEndpoint, cos.apiKey, cos.serviceInstanceID)).
		WithS3ForcePathStyle(true)

	sess := session.Must(session.NewSession())
	client := s3.New(sess, cos.conf)
	cos.client = client
}

func (cos *COS) MakeObjKey(name string) (objectKeyWithPrefix string) {
	return fmt.Sprintf("(%s)%s", cos.objKeyPrefix, name)
}

func (cos *COS) UploadFile(localFile string) (*s3manager.UploadOutput, error) {
	if cos.conf == nil {
		cos.makeConfig()
	}
	return cos.uploadFile(localFile)
}

func (cos *COS) uploadFile(localFile string) (*s3manager.UploadOutput, error) {
	file, err := os.Open(localFile)
	if err != nil {
		return nil, err
	}
	//noinspection GoUnhandledErrorResult
	defer file.Close()
	name := filepath.Base(localFile)
	return cos.upload(name, file)
}

/*
func (cos *COS) upload(objName string, objBody io.ReadSeeker) (*s3.PutObjectOutput, error) {
	sess := session.Must(session.NewSession())
	client := s3.New(sess, cos.conf)

	input := s3.PutObjectInput{
		Bucket: aws.String(cos.bucketName),
		Key: aws.String(cos.MakeObjKey(objName)),
		Body: objBody,
	}
	return client.PutObject(&input)
}
*/
func (cos *COS) upload(objName string, objBody io.Reader) (*s3manager.UploadOutput, error) {
	sess := session.Must(session.NewSession(cos.conf))
	uploader := s3manager.NewUploader(sess)
	return uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(cos.bucketName),
		Key:    aws.String(cos.MakeObjKey(objName)),
		Body:   objBody,
	})
}

func (cos *COS) Upload(objName string, objBody io.Reader) (*s3manager.UploadOutput, error) {
	if cos.conf == nil {
		cos.makeConfig()
	}
	return cos.upload(objName, objBody)
}

func (cos *COS) getObject(name string) (*s3.GetObjectOutput, error) {
	input := s3.GetObjectInput{
		Bucket: aws.String(cos.bucketName),
		Key:    aws.String(cos.MakeObjKey(name)),
	}
	return cos.client.GetObject(&input)
}

// GetObjectBody 返回 io.ReadCloser, 要记得关闭资源.
func (cos *COS) GetObjectBody(name string) (io.ReadCloser, error) {
	if cos.conf == nil {
		cos.makeConfig()
	}
	output, err := cos.getObject(name)
	if err != nil {
		return nil, err
	}
	return output.Body, nil
}

func (cos *COS) listObjects() (*s3.ListObjectsV2Output, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:            aws.String(cos.bucketName),
		Prefix:            aws.String(fmt.Sprintf("(%s)", cos.objKeyPrefix)),
	}
	return cos.client.ListObjectsV2(input)
}

func (cos *COS) GetLastModified(objName string) (lastModified *time.Time, err error) {
	if cos.conf == nil {
		cos.makeConfig()
	}
	output, err := cos.listObjects()
	if err != nil {
		return nil, err
	}

	objKey := cos.MakeObjKey(objName)
	for _, obj := range output.Contents {
		if *obj.Key == objKey {
			return obj.LastModified, nil
		}
	}
	return nil, fmt.Errorf("NotFound: object key: %s", objKey)
}
