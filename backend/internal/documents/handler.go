package documents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"kosh/drive"
)

const maxUploadSize = 20 << 20 // 20 MB

// driveFolders maps document type keys to their Drive folder paths.
var driveFolders = map[string]string{
	"fd":        "Kosh/FD",
	"insurance": "Kosh/Insurance",
	"metals":    "Kosh/Gold & Silver",
	"nps":       "Kosh/NPS",
	"income":    "Kosh/Payslips",
	"tax_rules": "Kosh/Tax Rules",
}

// UploadResult is returned to the frontend after parsing.
type UploadResult struct {
	Fields   map[string]interface{} `json:"fields"`
	DriveURL string                 `json:"drive_url,omitempty"`
}

// DriveClientFactory creates a Drive API client from a user OAuth access token.
// This is a function type so it can be swapped in tests.
type DriveClientFactory func(ctx context.Context, accessToken string) (drive.API, error)

// UploadHandler parses financial documents and stores them in Google Drive.
type UploadHandler struct {
	driveFactory DriveClientFactory
	parser       Parser
}

// NewUploadHandler creates an UploadHandler.
// driveFactory may be nil to disable Drive uploads.
// parser may be nil to disable AI parsing (returns 503 if called without one).
func NewUploadHandler(driveFactory DriveClientFactory, parser Parser) *UploadHandler {
	return &UploadHandler{
		driveFactory: driveFactory,
		parser:       parser,
	}
}

// Handle processes a multipart POST request, uploads the file to Drive (if a
// drive_token is provided), decrypts it if password-protected, and parses it
// with the configured Parser.
func (h *UploadHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	docType := strings.TrimPrefix(r.URL.Path, "/api/upload/")
	if _, ok := driveFolders[docType]; !ok {
		http.Error(w, fmt.Sprintf("unknown document type %q — use fd, insurance, metals, nps, income, or tax_rules", docType), http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "file too large (max 20 MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing 'file' field in form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "could not read uploaded file", http.StatusInternalServerError)
		return
	}

	// ── 1. Upload to the user's own Drive (best-effort; needs drive_token) ───
	// The frontend obtains a short-lived OAuth access token (scope drive.file)
	// via Google Identity Services and sends it along with the file. This
	// writes directly into the user's Drive — a service account has no
	// storage quota of its own and cannot own files here.
	driveURL := ""
	if token := r.FormValue("drive_token"); token != "" && h.driveFactory != nil {
		dc, err := h.driveFactory(r.Context(), token)
		if err != nil {
			fmt.Printf("⚠ Drive client error: %v\n", err)
		} else if folderID, err := dc.EnsureFolder(driveFolders[docType]); err != nil {
			fmt.Printf("⚠ Drive folder error: %v\n", err)
		} else {
			stamped := fmt.Sprintf("%d-%s", time.Now().Unix(), header.Filename)
			mime := header.Header.Get("Content-Type")
			if mime == "" {
				mime = MimeFromFilename(header.Filename)
			}
			url, err := dc.Upload(folderID, stamped, mime, bytes.NewReader(data), nil)
			if err != nil {
				fmt.Printf("⚠ Drive upload error: %v\n", err)
			} else {
				driveURL = url
			}
		}
	}

	// ── 2. Decrypt if password-protected (in-memory only — never persisted) ──
	parseData := data
	if password := r.FormValue("password"); password != "" && MimeFromFilename(header.Filename) == "application/pdf" {
		decrypted, derr := DecryptPDF(data, password)
		if derr != nil {
			http.Error(w, derr.Error(), http.StatusBadRequest)
			return
		}
		parseData = decrypted
	}

	// ── 3. Parse with Claude ──────────────────────────────────────────────────
	if h.parser == nil {
		http.Error(w, "ANTHROPIC_API_KEY not configured", http.StatusServiceUnavailable)
		return
	}

	mime := MimeFromFilename(header.Filename)
	fields, err := h.parser.Parse(r.Context(), docType, mime, parseData)
	if err != nil {
		http.Error(w, fmt.Sprintf("AI parsing failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, UploadResult{Fields: fields, DriveURL: driveURL})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
