package tftp

type MemoryFileStorage struct {
	files map[string]FileMetadata
}

func (s *MemoryFileStorage) StartNewUpload(filename string) {

}
