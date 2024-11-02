package web

import "fmt"

// RepositoryCollectionError is used in case of error during the metrics collection
// for a borg repository
type RepositoryCollectionError struct {
	Repository string
	Msg        string
	Err        error
	StdErr     string
}

func (e *RepositoryCollectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

// Unwrap allows errors.Is and errors.As to retrieve the original error
func (e *RepositoryCollectionError) Unwrap() error {
	return e.Err
}
