package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type VideoInfo struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func main() {
	var (
		inputFile = flag.String("input", "", "Input MP4 video file")
		outputDir = flag.String("output", "", "Output directory for thumbnails (optional)")
		width     = flag.Int("width", 320, "Thumbnail width")
		height    = flag.Int("height", 240, "Thumbnail height")
	)
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Usage: thumbs -input video.mp4 [-output output_dir] [-width 320] [-height 240]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *outputDir == "" {
		ext := filepath.Ext(*inputFile)
		name := strings.TrimSuffix(filepath.Base(*inputFile), ext)
		*outputDir = name + "_thumbs"
	}

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	duration, err := getVideoDuration(*inputFile)
	if err != nil {
		log.Fatalf("Failed to get video duration: %v", err)
	}

	fmt.Printf("Video duration: %.2f seconds\n", duration)
	fmt.Printf("Generating thumbnails in directory: %s\n", *outputDir)

	if err := generateThumbnails(*inputFile, *outputDir, duration, *width, *height); err != nil {
		log.Fatalf("Failed to generate thumbnails: %v", err)
	}

	totalThumbs := int(duration)
	fmt.Printf("Generated %d thumbnails\n", totalThumbs)

	if err := generateCombinedImages(*outputDir, totalThumbs, *width, *height); err != nil {
		log.Fatalf("Failed to generate combined images: %v", err)
	}

	fmt.Println("Generated 5 combined images")
}

func getVideoDuration(inputFile string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		inputFile,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	var info VideoInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return 0, fmt.Errorf("failed to parse video info: %w", err)
	}

	duration, err := strconv.ParseFloat(info.Format.Duration, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return duration, nil
}

func generateThumbnails(inputFile, outputDir string, duration float64, width, height int) error {
	for i := 0; i < int(duration); i++ {
		timestamp := fmt.Sprintf("%d", i)
		outputFile := filepath.Join(outputDir, fmt.Sprintf("thumb_%04d.jpeg", i))

		if err := generateThumbnail(inputFile, outputFile, timestamp, width, height); err != nil {
			return fmt.Errorf("failed to generate thumbnail at %ds: %w", i, err)
		}

		fmt.Printf("Generated: %s\n", outputFile)
	}

	return nil
}

func generateThumbnail(inputFile, outputFile, timestamp string, width, height int) error {
	cmd := exec.Command("ffmpeg",
		"-i", inputFile,
		"-ss", timestamp,
		"-vframes", "1",
		"-f", "image2",
		"-vf", fmt.Sprintf("scale=%d:%d", width, height),
		"-q:v", "2",
		"-y",
		outputFile,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	return nil
}

func generateCombinedImages(outputDir string, totalThumbs, thumbWidth, thumbHeight int) error {
	maxWidth := 8000
	thumbsPerRow := maxWidth / thumbWidth
	if thumbsPerRow < 1 {
		thumbsPerRow = 1
	}
	
	thumbsPerImage := (totalThumbs + 4) / 5
	
	for i := 0; i < 5; i++ {
		startIdx := i * thumbsPerImage
		endIdx := startIdx + thumbsPerImage
		if endIdx > totalThumbs {
			endIdx = totalThumbs
		}
		
		if startIdx >= totalThumbs {
			break
		}
		
		actualThumbsInImage := endIdx - startIdx
		rows := (actualThumbsInImage + thumbsPerRow - 1) / thumbsPerRow
		actualThumbsInLastRow := actualThumbsInImage % thumbsPerRow
		if actualThumbsInLastRow == 0 {
			actualThumbsInLastRow = thumbsPerRow
		}
		
		combinedWidth := thumbsPerRow * thumbWidth
		combinedHeight := rows * thumbHeight
		
		combined := image.NewRGBA(image.Rect(0, 0, combinedWidth, combinedHeight))
		
		for j := 0; j < actualThumbsInImage; j++ {
			thumbIdx := startIdx + j
			thumbPath := filepath.Join(outputDir, fmt.Sprintf("thumb_%04d.jpeg", thumbIdx))
			
			thumbFile, err := os.Open(thumbPath)
			if err != nil {
				continue
			}
			
			thumbImg, err := jpeg.Decode(thumbFile)
			thumbFile.Close()
			if err != nil {
				continue
			}
			
			row := j / thumbsPerRow
			col := j % thumbsPerRow
			x := col * thumbWidth
			y := row * thumbHeight
			
			draw.Draw(combined, image.Rect(x, y, x+thumbWidth, y+thumbHeight), thumbImg, image.Point{0, 0}, draw.Src)
		}
		
		outputPath := filepath.Join(outputDir, fmt.Sprintf("combined_%d.jpeg", i+1))
		outFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create combined image %d: %w", i+1, err)
		}
		
		options := &jpeg.Options{Quality: 90}
		if err := jpeg.Encode(outFile, combined, options); err != nil {
			outFile.Close()
			return fmt.Errorf("failed to encode combined image %d: %w", i+1, err)
		}
		outFile.Close()
		
		fmt.Printf("Generated combined image: %s (%dx%d, %d thumbnails in %d rows)\n", 
			outputPath, combinedWidth, combinedHeight, actualThumbsInImage, rows)
	}
	
	return nil
}
