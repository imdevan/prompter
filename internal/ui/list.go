package ui

import "github.com/charmbracelet/bubbles/list"

// ListDelegateOptions configures shared list presentation settings.
type ListDelegateOptions struct {
	Height              int
	PaddingLeft         int
	SelectedPaddingLeft int
}

// NewListModel creates a list with shared styles applied.
func NewListModel(items []list.Item, delegate list.ItemDelegate, width, height int, theme Theme) list.Model {
	model := list.New(items, delegate, width, height)
	ApplyListFilterStyles(&model, theme)
	return model
}

// ApplyListFilterStyles sets shared filter styles for lists.
func ApplyListFilterStyles(model *list.Model, theme Theme) {
	if model == nil {
		return
	}
	model.Styles.FilterPrompt = model.Styles.FilterPrompt.Foreground(theme.Secondary)
	model.Styles.FilterCursor = model.Styles.FilterCursor.Foreground(theme.Secondary)
	model.FilterInput.PromptStyle = model.FilterInput.PromptStyle.Foreground(theme.Secondary)
	model.FilterInput.Cursor.Style = model.FilterInput.Cursor.Style.Foreground(theme.Secondary)
	model.FilterInput.TextStyle = model.FilterInput.TextStyle.Foreground(theme.Text)
	model.Styles.DefaultFilterCharacterMatch = model.Styles.DefaultFilterCharacterMatch.Foreground(theme.Secondary)
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
