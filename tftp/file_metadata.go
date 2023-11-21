package tftp

type FileMetadata struct {
	Filename     string
	IsComplete   bool
	LastBlockNum int
	// ID
	// packet number
}
