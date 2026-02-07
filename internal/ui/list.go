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
	ApplyListStyles(&model, theme)
	return model
}

// ApplyListStyles sets shared list styles.
func ApplyListStyles(model *list.Model, theme Theme) {
	if model == nil {
		return
	}
	ApplyListFilterStyles(model, theme)
	model.Styles.NoItems = model.Styles.NoItems.Foreground(theme.Muted)
	model.Styles.StatusBar = model.Styles.StatusBar.Foreground(theme.Muted)
	model.Styles.StatusEmpty = model.Styles.StatusEmpty.Foreground(theme.Muted)
	model.Styles.StatusBarActiveFilter = model.Styles.StatusBarActiveFilter.Foreground(theme.Secondary)
	model.Styles.StatusBarFilterCount = model.Styles.StatusBarFilterCount.Foreground(theme.Muted)
	model.Styles.HelpStyle = model.Styles.HelpStyle.Foreground(theme.Muted)
	model.Styles.PaginationStyle = model.Styles.PaginationStyle.Foreground(theme.Muted)
	model.Styles.ActivePaginationDot = model.Styles.ActivePaginationDot.Foreground(theme.Secondary)
	model.Styles.InactivePaginationDot = model.Styles.InactivePaginationDot.Foreground(theme.Muted)
	model.Styles.DividerDot = model.Styles.DividerDot.Foreground(theme.Muted)
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
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(theme.TextHighlight).BorderForeground(theme.Primary).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(theme.DescriptionHighlight).BorderForeground(theme.Primary)
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
