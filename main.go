package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type progressWriter struct {
	progress func(int)
}

func downloadFile(url string, filename string, fsize int64) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusOK {
		fmt.Println(" request, awaiting response... status 200 OK")
	}
	defer response.Body.Close()
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	progress := 0
	startTime := time.Now()
	progressBar := func(completed int) {
			progress += completed
			percent := float64(progress) / float64(fsize) * 100
			elapsed := time.Since(startTime).Seconds()
			speed := float64(progress) / 1024 / 1024 / elapsed

			fmt.Printf("\rsaving file to: %s\n%.2f KiB / %.2f KiB [", filename, float64(progress)/1024, float64(fsize)/1024)
			for i := 0; i < 50; i++ {
					if i < int(percent/2) {
							fmt.Print("=")
					} else {
							fmt.Print(" ")
					}
			}
			fmt.Printf("] %.2f%% %.2f MiB/s %.0f s", percent, speed, elapsed)
			if percent == 100 {
					fmt.Println()
			}
	}

	reader := io.TeeReader(response.Body, &progressWriter{progressBar})
	size, err := io.Copy(file, reader)
	if size == 0 {
		err = errors.New("could not downlaod")
	}
	return err
}

func main() {
	url := os.Args[1]
	filename := getFilename(url)

	l := log.New(os.Stdout, "Start at ", log.Ldate|log.Ltime)
	l.Print()
	size, err := getDownloadSize(url)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("content size: %d [~%.2fMB]\n", size, float64(size)/1024/1024)
	//fmt.Printf("saving file to: ./%s\n", filename)
	err = downloadFile(url, filename, size)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nDownloaded [%s] \n", url)
	j := log.New(os.Stdout, "finished at ", log.Ldate|log.Ltime)
	j.Print()
}

func getDownloadSize(url string) (int64, error) {
	response, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	fsize, err := strconv.Atoi(response.Header.Get("Content-Length"))
	if err != nil {
		return 0, err
	}
	return int64(fsize), err
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.progress(n)
	return n, nil
}

func getFilename(url string) string {
	names := strings.Split(url, "/")
	return names[len(names)-1]
}
