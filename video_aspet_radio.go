package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	buf := new(bytes.Buffer)

	cmd.Stdout = buf

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	type ffprobeOutput struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	ffprobeData := ffprobeOutput{}
	err = json.Unmarshal(buf.Bytes(), &ffprobeData)
	if err != nil {
		return "", err
	}

	height := ffprobeData.Streams[0].Height
	width := ffprobeData.Streams[0].Width

	actualRatio := float64(width) / float64(height)
	tolerance := 0.01

	if math.Abs(actualRatio-16.0/9.0) < tolerance {
		return "16:9", nil
	} else if math.Abs(actualRatio-9.0/16.0) < tolerance {
		return "9:16", nil
	}
	return "other", nil
}
