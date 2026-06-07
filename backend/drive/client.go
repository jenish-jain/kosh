package drive

import (
	"context"
	"fmt"
	"io"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Client struct {
	svc *drive.Service
}

func NewClient(credentialsPath string) (*Client, error) {
	ctx := context.Background()
	svc, err := drive.NewService(ctx, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("creating drive service: %w", err)
	}
	return &Client{svc: svc}, nil
}

// NewClientFromToken builds a Drive client acting as the user themselves,
// using a short-lived OAuth access token obtained client-side (scope
// drive.file). Files land directly in the user's own Drive with no storage
// quota or sharing concerns — unlike a service account, which has neither.
func NewClientFromToken(ctx context.Context, accessToken string) (*Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	svc, err := drive.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("creating drive service: %w", err)
	}
	return &Client{svc: svc}, nil
}

// EnsureFolder walks or creates the folder path (e.g. "Kosh/FD") under root
// and returns the leaf folder ID.
func (c *Client) EnsureFolder(path string) (string, error) {
	parentID := "root"
	for _, name := range strings.Split(path, "/") {
		id, err := c.findOrCreate(name, parentID)
		if err != nil {
			return "", err
		}
		parentID = id
	}
	return parentID, nil
}

func (c *Client) findOrCreate(name, parentID string) (string, error) {
	q := fmt.Sprintf(
		"name=%q and mimeType='application/vnd.google-apps.folder' and %q in parents and trashed=false",
		name, parentID,
	)
	list, err := c.svc.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return "", fmt.Errorf("listing folders: %w", err)
	}
	if len(list.Files) > 0 {
		return list.Files[0].Id, nil
	}
	f, err := c.svc.Files.Create(&drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{parentID},
	}).Fields("id").Do()
	if err != nil {
		return "", fmt.Errorf("creating folder %q: %w", name, err)
	}
	return f.Id, nil
}

// Upload stores a file in the given folder and shares it with shareEmails as
// readers. Returns the web view link.
func (c *Client) Upload(folderID, filename, mimeType string, content io.Reader, shareEmails []string) (string, error) {
	f, err := c.svc.Files.Create(&drive.File{
		Name:    filename,
		Parents: []string{folderID},
	}).Media(content).Fields("id,webViewLink").Do()
	if err != nil {
		return "", fmt.Errorf("uploading %q: %w", filename, err)
	}

	for _, email := range shareEmails {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}
		c.svc.Permissions.Create(f.Id, &drive.Permission{
			Type:         "user",
			Role:         "reader",
			EmailAddress: email,
		}).SendNotificationEmail(false).Do() // best-effort
	}

	return f.WebViewLink, nil
}
