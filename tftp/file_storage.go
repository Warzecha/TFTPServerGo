package tftp

type FileStorage interface {
	StartNewUpload(filename string) FileMetadata
	AppendData(filename string, blockNum int, data []byte)
	CompleteUpload(filename string)
	ReadFileBytes(filename string, start int, end int) []byte
	GetFileMetadata(filename string) (FileMetadata, bool)
}
