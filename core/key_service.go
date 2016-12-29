package core

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"strings"
)

type KeyService interface {
	Upload(keyName string, selectedApp SuripuApp, environment string) (*KeyUploadResult, error)
	CleanUp(uploadResult *KeyUploadResult) error
}

type S3KeyService struct {
	s3Service  *s3.S3
	ec2Service *ec2.EC2
	keyBucket  string
}

func NewS3KeyService(s3srv *s3.S3, ec2srv *ec2.EC2, keyBucket string) *S3KeyService {
	return &S3KeyService{
		s3Service:  s3srv,
		ec2Service: ec2srv,
		keyBucket:  keyBucket,
	}
}

type KeyUploadResult struct {
	ETag    string
	KeyName string
	Key     string
}

func (s *S3KeyService) Upload(keyName string, selectedApp SuripuApp, environment string) (*KeyUploadResult, error) {
	keyPairParams := &ec2.CreateKeyPairInput{
		KeyName: aws.String(keyName), // Required
		DryRun:  aws.Bool(false),
	}
	keyPairResp, err := s.ec2Service.CreateKeyPair(keyPairParams)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to create KeyPair. %s", err.Error()))
	}

	//Upload key to S3
	key := fmt.Sprintf("/%s/%s/%s.pem", environment, selectedApp.Name, *keyPairResp.KeyName)

	uploadResult, err := s.s3Service.PutObject(&s3.PutObjectInput{
		Body:   strings.NewReader(*keyPairResp.KeyMaterial),
		Bucket: aws.String(s.keyBucket),
		Key:    &key,
	})

	if err != nil {
		return nil, err
	}

	keyUploadResult := &KeyUploadResult{
		ETag:    *uploadResult.ETag,
		KeyName: *keyPairResp.KeyName,
		Key:     key,
	}

	return keyUploadResult, nil
}

func (s *S3KeyService) CleanUp(uploadResult *KeyUploadResult) error {

	//Delete key from EC2
	params := &ec2.DeleteKeyPairInput{
		KeyName: aws.String(uploadResult.KeyName), // Required
		DryRun:  aws.Bool(false),
	}
	_, err := s.ec2Service.DeleteKeyPair(params)

	if err != nil {
		return errors.New(fmt.Sprintf("Failed to delete KeyPair: %s", err.Error()))
	}

	//Delete pem file from s3
	delParams := &s3.DeleteObjectInput{
		Bucket: aws.String(s.keyBucket),      // Required
		Key:    aws.String(uploadResult.Key), // Required
	}
	_, objErr := s.s3Service.DeleteObject(delParams)

	if objErr != nil {
		return errors.New(fmt.Sprintf("Failed to delete S3 Object: %s", objErr.Error()))
	}

	return nil
}
