package interfaces

// ContentLimits defines size limits for content inclusion
type ContentLimits struct {
	MaxFileSize   int64
	MaxTotal      int64
	AllowOversize bool
}

// ContentCollector handles file and directory content collection
type ContentCollector interface {
	// CollectFiles collects content from specified file paths
	CollectFiles(paths []string) ([]FileInfo, error)
	
	// CollectDirectory collects content from a directory using the specified strategy
	CollectDirectory(path string, strategy string) ([]FileInfo, error)
	
	// FilterContent applies size limits and filtering rules to collected content
	FilterContent(files []FileInfo, limits ContentLimits) ([]FileInfo, error)
}