package interactive

import "errors"

// ErrCanceled indicates the user exited an interactive prompt.
var ErrCanceled = errors.New("interactive canceled")
