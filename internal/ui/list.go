package ui

import "github.com/charmbracelet/bubbles/list"

// ListDelegateOptions configures shared list presentation settings.
type ListDelegateOptions struct {
	Height              int
	PaddingLeft         int
	SelectedPaddingLeft int
}

// NewListDelegate provides shared list focus styles.
func NewListDelegate(theme Theme, opts ListDelegateOptions) list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(theme.Primary).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(theme.DescriptionHighlight)
	if opts.Height > 0 {
		delegate.SetHeight(opts.Height)
	}
	if opts.PaddingLeft > 0 {
		delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Padding(0, 0, 0, opts.PaddingLeft)
		delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Padding(0, 0, 0, opts.PaddingLeft)
	}
	if opts.SelectedPaddingLeft > 0 {
		delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Padding(0, 0, 0, opts.SelectedPaddingLeft)
		delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Padding(0, 0, 0, opts.SelectedPaddingLeft)
	}
	return delegate
}
