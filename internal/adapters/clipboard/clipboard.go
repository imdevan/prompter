package clipboard

import atotto "github.com/atotto/clipboard"

// Adapter writes content to the system clipboard.
type Adapter struct{}

// WriteText copies text to the clipboard.
func (Adapter) WriteText(text string) error {
	return atotto.WriteAll(text)
}
