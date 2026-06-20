package drive

import "io"

// API is the Drive client interface (allows injection for testing).
type API interface {
	EnsureFolder(path string) (string, error)
	Upload(folderID, filename, mimeType string, content io.Reader, shareEmails []string) (string, error)
}

// Compile-time check that *Client implements API.
var _ API = (*Client)(nil)
