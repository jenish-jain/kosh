package documents

import (
	"bytes"
	"fmt"
	"strings"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfmodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// DecryptPDF returns the plaintext bytes of a password-protected PDF so it
// can be handed to Claude for parsing. The password lives only in this
// function call — it is never logged, persisted, or forwarded to Drive
// (the original encrypted file is what gets stored there).
func DecryptPDF(data []byte, password string) ([]byte, error) {
	pdfmodel.ConfigPath = "disable" // no on-disk config — keep this stateless
	conf := pdfmodel.NewDefaultConfiguration()
	conf.UserPW = password
	conf.OwnerPW = password

	var out bytes.Buffer
	err := pdfapi.Decrypt(bytes.NewReader(data), &out, conf)
	if err == nil {
		return out.Bytes(), nil
	}
	if strings.Contains(err.Error(), "not encrypted") {
		return data, nil // no password needed — use as-is
	}
	return nil, fmt.Errorf("could not open the PDF — check the password and try again")
}

// MimeFromFilename returns the MIME type for a filename based on its extension.
func MimeFromFilename(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	default:
		return "application/pdf"
	}
}
