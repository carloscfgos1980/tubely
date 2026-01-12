package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func (cfg *apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func (cfg *apiConfig) getVideoURL(key string) string {

	//https://<bucket-name>.s3.<region>.amazonaws.com/<key>
	return fmt.Sprintf("http://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
}

func (cfg *apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func getAssetPath(mediaType string) string {
	// generate a random file name
	mySlice := make([]byte, 32)
	name := base64.RawURLEncoding.EncodeToString(mySlice)

	extension := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"video/mp4":  ".mp4",
	}

	ext := extension[mediaType]
	return fmt.Sprintf("%s%s", name, ext)
}

// another way to get the extention:
// func mediaTypeToExt(mediaType string) string {
// 	parts := strings.Split(mediaType, "/")
// 	if len(parts) != 2 {
// 		return ".bin"
// 	}
// 	return "." + parts[1]
// }

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	var bucket, key string
	url := video.VideoURL

	if url == nil {
		return video, nil
	}

	result := strings.Split(*url, ",")
	if len(result) != 2 {
		return video, nil
	}

	bucket = result[0]
	key = result[1]

	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, 15*time.Minute)
	if err != nil {
		return video, err
	}

	video.VideoURL = &presignedURL
	return video, nil

}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	// Create a presign client
	presignClient := s3.NewPresignClient(s3Client)

	// Create the GetObject input
	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	// Generate the presigned URL
	presignedURL, err := presignClient.PresignGetObject(context.TODO(), getObjectInput, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}

	return presignedURL.URL, nil
}
