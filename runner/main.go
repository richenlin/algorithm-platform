package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	InputURL   string `json:"input_url"`
	OutputURL  string `json:"output_url"`
	WebhookURL string `json:"webhook_url"`
}

func main() {
	configPath := os.Getenv("ALG_CONFIG")
	if configPath == "" {
		log.Fatal("ALG_CONFIG environment variable not set")
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(configData, &cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	minioClient, err := minio.New(os.Getenv("MINIO_ENDPOINT"), &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_SECRET_KEY"), ""),
		Secure: os.Getenv("MINIO_USE_SSL") == "true",
	})
	if err != nil {
		log.Fatalf("Failed to create MinIO client: %v", err)
	}

	inputDir := "/app/input"
	outputDir := "/app/output"

	os.MkdirAll(inputDir, 0755)
	os.MkdirAll(outputDir, 0755)

	if cfg.InputURL != "" {
		if err := downloadFile(minioClient, cfg.InputURL, filepath.Join(inputDir, "data")); err != nil {
			log.Fatalf("Failed to download input: %v", err)
		}
	}

	algoCmd := os.Getenv("ALGO_CMD")
	if algoCmd == "" {
		algoCmd = "python main.py"
	}

	cmd := exec.Command("sh", "-c", algoCmd)
	cmd.Dir = "/app"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("Algorithm execution failed: %v", err)
	}

	if cfg.OutputURL != "" {
		outputFile := filepath.Join(outputDir, "result")
		file, err := os.Open(outputFile)
		if err != nil {
			log.Fatalf("Failed to open output: %v", err)
		}
		defer file.Close()

		if err := uploadFile(minioClient, cfg.OutputURL, file); err != nil {
			log.Fatalf("Failed to upload output: %v", err)
		}
	}

	if cfg.WebhookURL != "" {
		sendWebhook(cfg.WebhookURL, "success", cfg.OutputURL)
	}
}

func downloadFile(client *minio.Client, url, destPath string) error {
	bucket, object := getBucketAndObject(url)
	reader, err := client.GetObject(context.Background(), bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

func uploadFile(client *minio.Client, url string, file *os.File) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	bucket, object := getBucketAndObject(url)
	_, err = client.PutObject(context.Background(), bucket, object, file, stat.Size(), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	return err
}

func sendWebhook(url, status, resultURL string) {
	log.Printf("Sending webhook to %s: status=%s, result=%s", url, status, resultURL)
}

func getBucketAndObject(url string) (string, string) {
	parts := strings.SplitN(url, "/", 2)
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
