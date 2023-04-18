package utils

import (
	"io/fs"
	"log"
	"path/filepath"
	"sync"
)

func processFile(path string, d fs.DirEntry, resultChan chan int64, wg *sync.WaitGroup) {
	defer wg.Done()

	if !d.IsDir() {
		fileInfo, err := d.Info()
		if err != nil {
			log.Printf("Error getting file info: %v", err)
			return
		}

		resultChan <- fileInfo.Size()
	}
}

func SizeAndCount(dir string, workerLimit int) (int64, int, error) {
	semaphore := make(chan struct{}, workerLimit)
	resultChan := make(chan int64)
	var wg sync.WaitGroup

	go func() {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			wg.Add(1)
			semaphore <- struct{}{}
			go func() {
				processFile(path, d, resultChan, &wg)
				<-semaphore
			}()

			return nil
		})
		if err != nil {
			log.Printf("Error walking directory: %v", err)
			close(resultChan)
			return
		}
		wg.Wait()
		close(resultChan)
	}()

	totalSize := int64(0)
	fileCount := 0
	for size := range resultChan {
		totalSize += size
		fileCount++
	}

	return totalSize, fileCount, nil
}
