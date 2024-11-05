package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var downloadWg sync.WaitGroup

func DownloadFile(urlStr, fileName string, background bool, rateLimit int64) error {
	startTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("start at %s\n", startTime)

	client := &http.Client{}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error: got status %s", resp.Status)
	}
	fmt.Printf("sending request, awaiting response... status %s\n", resp.Status)

	contentLength := resp.ContentLength
	if contentLength == -1 {
		contentLength = 0
	}
	if float64(contentLength)/1000/1000 > 1000 {
		fmt.Printf("content size: %d [~%.2fGB]\n", contentLength, float64(contentLength)/1000/1000/1000)
	} else {
		fmt.Printf("content size: %d [~%.2fMB]\n", contentLength, float64(contentLength)/1000/1000)
	}

	out, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	defer out.Close()

	fmt.Printf("saving file to: ./%s\n", fileName)

	var reader io.Reader = resp.Body
	if rateLimit > 0 {
		fmt.Printf("Rate limit set to: %.2f KB/s\n", float64(rateLimit)/1024)
		reader = NewRateLimitReader(resp.Body, rateLimit)
	}

	if background {
		_, err := io.Copy(out, reader)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}
	} else {
		bar := NewProgressBar(contentLength, 50)
		bar.StartTimer()

		_, err = io.Copy(io.MultiWriter(out, bar), reader)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}
	}

	endTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("Downloaded [%s]\nfinished at %s\n", urlStr, endTime)

	return nil
}

func DownloadWithLogging(urlStr string, fileName string, background bool, rateLimit int64) {
	if background {
		// Check if this is the child process
		if len(os.Args) > 1 && os.Args[len(os.Args)-1] == "background-download" {
			// Open log file with O_TRUNC flag instead of O_APPEND to clear existing content
			logFile, err := os.OpenFile("wget-log", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return
			}
			defer logFile.Close()

			// Redirect stdout to log file
			os.Stdout = logFile
			os.Stderr = logFile

			// Perform the download
			err = DownloadFile(urlStr, fileName, true, rateLimit)
			if err != nil {
				fmt.Fprintf(logFile, "Error: %v\n", err)
			}
			return
		}

		// This is the parent process
		fmt.Println("Output will be written to \"wget-log\".")

		// Get the path to the current executable
		executable, err := os.Executable()
		if err != nil {
			fmt.Printf("Error getting executable path: %v\n", err)
			return
		}

		// Create command for the child process
		args := append([]string{}, os.Args[1:]...)    // Copy original args
		args = append(args, "background-download")     // Add background flag
		cmd := exec.Command(executable, args...)

		// Detach the process from terminal
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil

		// Start the detached process
		err = cmd.Start()
		if err != nil {
			fmt.Printf("Error starting background process: %v\n", err)
			return
		}

		// Detach it from the parent
		cmd.Process.Release()

	} else {
		err := DownloadFile(urlStr, fileName, background, rateLimit)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func GetFileName(url string) string {
	s := strings.Split(url, "/")
	return s[len(s)-1]
}
