package tftp

import (
	"fmt"
	"math"
)

type MemoryFileStorage struct {
	files        map[string]*FileMetadata
	fileContents map[string][]byte
}

func (s *MemoryFileStorage) StartNewUpload(filename string) FileMetadata {
	newFile := FileMetadata{
		Filename:     filename,
		IsComplete:   false,
		LastBlockNum: 0,
	}

	s.files[filename] = &newFile
	s.fileContents[filename] = []byte{}
	return newFile
}

func (s *MemoryFileStorage) AppendData(filename string, blockNum int, data []byte) {
	s.fileContents[filename] = append(s.fileContents[filename], data...)
	s.files[filename].LastBlockNum = blockNum

}

func (s *MemoryFileStorage) CompleteUpload(filename string) {
	s.files[filename].IsComplete = true
	fmt.Printf("Completing upload of file: %s \n", filename)
}

func (s *MemoryFileStorage) ReadFileBytes(filename string, start int, end int) []byte {

	fullContent := s.fileContents[filename]

	start = int(math.Min(float64(start), float64(len(fullContent))))
	end = int(math.Min(float64(end), float64(len(fullContent))))
	return s.fileContents[filename][start:end]
}
