package interfaces

// ContentLimits defines size limits for content inclusion
type ContentLimits struct {
	MaxFileSize   int64
	MaxTotal      int64
	AllowOversize bool
}