package rate

// ClcError is a user-facing error whose message is meant to be printed as-is
// (already localized) without a "panic"-style stack or Go wrapping noise.
type ClcError struct {
	Msg string
}

func (e *ClcError) Error() string { return e.Msg }
