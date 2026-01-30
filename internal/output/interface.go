package output

// Destination describes an output target.
type Destination interface {
	Write(content string) error
}
