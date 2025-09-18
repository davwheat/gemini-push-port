package rawstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"gemini-push-port/logging"
	"io"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var ukTimezone *time.Location

func init() {
	var err error
	ukTimezone, err = time.LoadLocation("Europe/London")
	if err != nil {
		panic(err)
	}
}

func DumpToBucketJob(s3client *s3.Client) {
	logging.Logger.Infof("Starting dump to bucket job...")

	nowTime := time.Now().In(ukTimezone)

	// upload the current hour's file and the previous hour's file
	hourlyFiles := []string{
		XmlMessageWithTime{MessageTime: nowTime.Add(-1 * time.Hour)}.GetFilePath(),
		XmlMessageWithTime{MessageTime: nowTime}.GetFilePath(),
	}

	for _, filePath := range hourlyFiles {
		err := uploadToS3(s3client, filePath)
		if err != nil {
			logging.Logger.Errorf(err, "failed to upload file %s to S3", filePath)
			continue
		} else {
			logging.Logger.Infof("successfully uploaded file %s to S3", filePath)
		}
	}
}

func uploadToS3(s3client *s3.Client, filePath string) error {
	workDir := os.Getenv("PUSH_PORT_DUMP_WORKDIR")
	r2PathPrefix := os.Getenv("PUSH_PORT_DUMP_R2_PATH_PREFIX")
	bucketName := os.Getenv("CLOUDFLARE_R2_BUCKET_NAME")

	localFilePath := path.Join(workDir, filePath)
	remoteFilePath := path.Join(r2PathPrefix, filePath+".gz")

	file, err := os.OpenFile(localFilePath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logging.Logger.ErrorE("failed to close file", err)
		}
	}(file)

	// gzip the data in memory
	var b bytes.Buffer
	gzWriter := gzip.NewWriter(&b)
	_, err = io.Copy(gzWriter, file)
	if err != nil {
		return err
	}

	// It's important to close the writer before reading from the buffer.
	// Closing the writer flushes any buffered data and writes the gzip footer.
	if err := gzWriter.Close(); err != nil {
		return err
	}

	_, err = s3client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(remoteFilePath),
		Body:   bytes.NewReader(b.Bytes()),
	})
	if err != nil {
		return err
	}

	return nil
}
