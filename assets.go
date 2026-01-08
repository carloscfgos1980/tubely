package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
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
