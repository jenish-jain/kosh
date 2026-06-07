package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	drv "kosh/drive"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfmodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const maxUploadSize = 20 << 20 // 20 MB

// UploadHandler parses financial documents and stores them in Google Drive.
type UploadHandler struct {
	anthropicKey string
	promptsDir   string
}

func NewUploadHandler(anthropicKey, promptsDir string) *UploadHandler {
	return &UploadHandler{
		anthropicKey: anthropicKey,
		promptsDir:   promptsDir,
	}
}

// UploadResult is returned to the frontend after parsing.
type UploadResult struct {
	Fields   map[string]interface{} `json:"fields"`
	DriveURL string                 `json:"drive_url,omitempty"`
}

var driveFolders = map[string]string{
	"fd":        "Kosh/FD",
	"insurance": "Kosh/Insurance",
	"metals":    "Kosh/Gold & Silver",
}

var promptFiles = map[string]string{
	"fd":        "fd.md",
	"insurance": "insurance.md",
	"metals":    "metals.md",
}

func (u *UploadHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	docType := strings.TrimPrefix(r.URL.Path, "/api/upload/")
	if _, ok := driveFolders[docType]; !ok {
		http.Error(w, fmt.Sprintf("unknown document type %q — use fd, insurance, or metals", docType), http.StatusBadRequest)
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
	if token := r.FormValue("drive_token"); token != "" {
		dc, err := drv.NewClientFromToken(r.Context(), token)
		if err != nil {
			fmt.Printf("⚠ Drive client error: %v\n", err)
		} else if folderID, err := dc.EnsureFolder(driveFolders[docType]); err != nil {
			fmt.Printf("⚠ Drive folder error: %v\n", err)
		} else {
			stamped := fmt.Sprintf("%d-%s", time.Now().Unix(), header.Filename)
			mime := header.Header.Get("Content-Type")
			if mime == "" {
				mime = mimeFromFilename(header.Filename)
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
	if password := r.FormValue("password"); password != "" && mimeFromFilename(header.Filename) == "application/pdf" {
		decrypted, derr := decryptPDF(data, password)
		if derr != nil {
			http.Error(w, derr.Error(), http.StatusBadRequest)
			return
		}
		parseData = decrypted
	}

	// ── 3. Parse with Claude Haiku ────────────────────────────────────────────
	if u.anthropicKey == "" {
		http.Error(w, "ANTHROPIC_API_KEY not configured", http.StatusServiceUnavailable)
		return
	}

	fields, err := u.parseWithClaude(docType, header.Filename, parseData)
	if err != nil {
		http.Error(w, fmt.Sprintf("AI parsing failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, UploadResult{Fields: fields, DriveURL: driveURL})
}

func (u *UploadHandler) parseWithClaude(docType, filename string, data []byte) (map[string]interface{}, error) {
	promptPath := filepath.Join(u.promptsDir, promptFiles[docType])
	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("reading prompt %s: %w", promptPath, err)
	}

	mime := mimeFromFilename(filename)
	encoded := base64.StdEncoding.EncodeToString(data)

	// PDFs use the "document" block type; images use "image".
	var mediaBlock map[string]interface{}
	if mime == "application/pdf" {
		mediaBlock = map[string]interface{}{
			"type": "document",
			"source": map[string]interface{}{
				"type":       "base64",
				"media_type": mime,
				"data":       encoded,
			},
		}
	} else {
		mediaBlock = map[string]interface{}{
			"type": "image",
			"source": map[string]interface{}{
				"type":       "base64",
				"media_type": mime,
				"data":       encoded,
			},
		}
	}

	body, _ := json.Marshal(map[string]interface{}{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 1024,
		"messages": []map[string]interface{}{{
			"role": "user",
			"content": []interface{}{
				mediaBlock,
				map[string]interface{}{
					"type": "text",
					"text": string(promptBytes),
				},
			},
		}},
	})

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", u.anthropicKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic API: %w", err)
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic returned %d: %s", resp.StatusCode, string(respBytes))
	}

	var anthResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBytes, &anthResp); err != nil || len(anthResp.Content) == 0 {
		return nil, fmt.Errorf("unexpected Anthropic response shape: %s", string(respBytes))
	}

	text := strings.TrimSpace(anthResp.Content[0].Text)
	// Strip markdown code fences the model sometimes adds despite instructions.
	if i := strings.Index(text, "{"); i > 0 {
		text = text[i:]
	}
	if i := strings.LastIndex(text, "}"); i >= 0 {
		text = text[:i+1]
	}

	var fields map[string]interface{}
	if err := json.Unmarshal([]byte(text), &fields); err != nil {
		return nil, fmt.Errorf("parsing extracted JSON (%w) — raw: %s", err, text)
	}
	return fields, nil
}

func mimeFromFilename(name string) string {
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

// decryptPDF returns the plaintext bytes of a password-protected PDF so it
// can be handed to Claude for parsing. The password lives only in this
// function call — it is never logged, persisted, or forwarded to Drive
// (the original encrypted file is what gets stored there).
func decryptPDF(data []byte, password string) ([]byte, error) {
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
