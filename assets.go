package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
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

func (cfg *apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func getAssetPath(videoID uuid.UUID, mediaType string) string {

	extension := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
	}

	ext := extension[mediaType]
	return fmt.Sprintf("%s%s", videoID, ext)
}

// another way to get the extention:
// func mediaTypeToExt(mediaType string) string {
// 	parts := strings.Split(mediaType, "/")
// 	if len(parts) != 2 {
// 		return ".bin"
// 	}
// 	return "." + parts[1]
// }
