package main

import (
	"archive/zip"
	"bufio"
	"flag"
	"fmt"
	"hash/crc32"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type FileInfo struct {
	Name  string
	Size  uint64
	CRC32 uint32
}

func getFileInfoFromZip(zipFilePath string) (map[string]FileInfo, error) {
	// Open the zip file
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open the zip file: %v", err)
	}
	defer r.Close()

	// Initialize the map to store the file info
	fileInfoMap := make(map[string]FileInfo)

	// Iterate over the files in the zip file
	for _, f := range r.File {
		// Get the file size and CRC32
		fileInfo := FileInfo{
			Name:  f.Name,
			Size:  f.UncompressedSize64,
			CRC32: f.CRC32,
		}

		// Store the file info in the map
		fileInfoMap[f.Name] = fileInfo
	}

	return fileInfoMap, nil
}

var charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
var charsetWithSymbols = charset + "!@#$%^&*()_+-=[]{};':\",./<>?\\|`~"
var printMutex = &sync.Mutex{}
var attempts uint64

func generateAllStrings(current string, size uint64, crc32Check uint32, wg *sync.WaitGroup, charset string, index *uint64) {
	defer wg.Done()

	if uint64(len(current)) == size {
		atomic.AddUint64(&attempts, 1)
		if crc32.ChecksumIEEE([]byte(current)) == crc32Check {
			printMutex.Lock()
			fmt.Printf("%d. %s\n", *index, current)
			*index++
			printMutex.Unlock()
			os.Exit(0)
		}
	} else {
		for _, c := range charset {
			wg.Add(1)
			go generateAllStrings(current+string(c), size, crc32Check, wg, charset, index)
		}
	}
}

func bruteForce(size uint64, crc32Check uint32, useSymbols bool, index uint64) {
	var wg sync.WaitGroup
	wg.Add(1)

	charsetToUse := charset
	if useSymbols {
		charsetToUse = charsetWithSymbols
	}

	go generateAllStrings("", size, crc32Check, &wg, charsetToUse, &index)

	wg.Wait()
}

func parseUserInput(input string) []int {
	// Initialize a slice to store the result
	var result []int

	// Split the input by comma
	parts := strings.Split(input, ",")

	for _, part := range parts {
		// If the part contains a dash, it's a range
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			start, _ := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			end, _ := strconv.Atoi(strings.TrimSpace(rangeParts[1]))

			// Add all numbers in the range to the result
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
		} else {
			// The part is a single number
			number, _ := strconv.Atoi(strings.TrimSpace(part))
			result = append(result, number)
		}
	}

	return result
}

func main() {
	// Define command line arguments
	filePath := flag.String("f", "", "The path of the zip file")
	useSymbols := flag.Bool("s", false, "Whether to use symbols in the brute force attack")

	// Parse command line arguments
	flag.Parse()

	// Check if file path is provided
	if *filePath == "" {
		flag.Usage()
		return
	}

	// Get the file info from the zip file
	fileInfoMap, err := getFileInfoFromZip(*filePath)
	if err != nil {
		fmt.Printf("failed to get file info from zip file: %v\n", err)
		return
	}

	// Copy the file info to a slice and sort it by size
	fileInfos := make([]FileInfo, 0, len(fileInfoMap))
	for _, info := range fileInfoMap {
		fileInfos = append(fileInfos, info)
	}
	sort.Slice(fileInfos, func(i, j int) bool {
		if fileInfos[i].Size != fileInfos[j].Size {
			return fileInfos[i].Size < fileInfos[j].Size
		}
		return fileInfos[i].Name < fileInfos[j].Name
	})

	// Print the sorted file info with an index
	for i, info := range fileInfos {
		fmt.Printf("%d. Name: %s, Size: %d, CRC32: %x\n", i, info.Name, info.Size, info.CRC32)
	}

	// Get the user's choice
	fmt.Printf("Enter the numbers of the files you want to select (e.g., 0-3,5,7-9): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')

	// Parse the user's choice
	choices := parseUserInput(input)

	// For each choice, get the file info and call the brute force function
	for _, choice := range choices {
		if choice < 0 || choice >= len(fileInfos) {
			fmt.Printf("Invalid choice: %d\n", choice)
			continue
		}

		// Get the selected file's info
		selectedFileInfo := fileInfos[choice]
		// fmt.Printf("You selected %s with size %d and CRC32 %d\n", selectedFileInfo.Name, selectedFileInfo.Size, selectedFileInfo.CRC32)

		// Call your brute force function here with the selected file's size and CRC32
		index := uint64(0)
		bruteForce(selectedFileInfo.Size, selectedFileInfo.CRC32, *useSymbols, index)
	}
}
