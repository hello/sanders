package shared

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
	"strings"
)

type UserMetaDataGenerator struct {
	expectedHash string
	bucket       string
	key          string
	srv          *s3.S3
}

func NewUserMetaDataGenerator(expectedHash, bucket, key string, srv *s3.S3) *UserMetaDataGenerator {
	return &UserMetaDataGenerator{
		bucket:       bucket,
		key:          key,
		expectedHash: expectedHash,
		srv:          srv,
	}
}

type UserMetaDataInput struct {
	AmiVersion    string
	AppName       string
	PackagePath   string
	CanaryPath    string
	DefaultRegion string
	JavaVersion   int
}

func (u *UserMetaDataGenerator) Parse(input *UserMetaDataInput) (string, error) {
	s3params := &s3.GetObjectInput{
		Bucket: aws.String(u.bucket), // Required
		Key:    aws.String(u.key),    // Required
	}
	resp, err := u.srv.GetObject(s3params)

	if err != nil {
		return "", err
	}

	// Pretty-print the response data.
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	userData := buf.String()

	//Verify checksum of userData
	hash := sha1.New()
	io.WriteString(hash, userData)
	userDataHash := fmt.Sprintf("%x", hash.Sum(nil))

	if userDataHash != u.expectedHash {
		return "", errors.New(fmt.Sprintf("hashes don't match %s = %s", userDataHash, u.expectedHash))
	}

	//do token replacement
	userData = strings.Replace(userData, "{app_version}", input.AmiVersion, -1)
	userData = strings.Replace(userData, "{app_name}", input.AppName, -1)
	userData = strings.Replace(userData, "{package_path}", input.PackagePath, -1)
	userData = strings.Replace(userData, "{canary_path}", input.CanaryPath, -1)
	userData = strings.Replace(userData, "{default_region}", "us-east-1", -1)
	userData = strings.Replace(userData, "{java_version}", fmt.Sprintf("%d", input.JavaVersion), -1)

	userData = base64.StdEncoding.EncodeToString([]byte(userData))
	return userData, nil
}
