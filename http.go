package gofirefox

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var headers = map[string]string{
	"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8", // default Accept header sent by Firefox
	"Accept-Language": "en-US,en;q=0.5",
	"DNT":             "1",
}

func downloadFile(ctx context.Context, client *http.Client, fileURL, filePath string) error {
	resp, err := openURLHTTP(ctx, client, fileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 399 {
		return fmt.Errorf("status_code=%d", resp.StatusCode)
	}
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %s", err)
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write to file %s - error: %s", filePath, err)
	}
	return nil
}

const userAgent = "Mozilla/5.0 (Windows NT 10.0; rv:91.0) Gecko/20100101 Firefox/91.0"

func openURLHTTP(ctx context.Context, client *http.Client, pageURL string) (*http.Response, error) {
	var resp *http.Response
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled")
		default:
		}
		if attempt > 0 {
			dur := time.Duration(attempt*attempt) * time.Second
			log.Printf("HTTP request failed: %s - retrying in %v", err, dur)
			time.Sleep(dur)
		}
		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, "GET", pageURL, nil)
		if err != nil {
			err = fmt.Errorf("http.NewRequest failed: %s", err)
			continue
		}
		req.Header.Set("User-Agent", userAgent)
		for headerKey, headerValue := range headers {
			req.Header.Set(headerKey, headerValue)
		}
		resp, err = client.Do(req)
		if err != nil {
			err = fmt.Errorf("HTTP request failed: %s", err)
			continue
		}
		// no errors => exit loop
		break
	}
	return resp, err
}
