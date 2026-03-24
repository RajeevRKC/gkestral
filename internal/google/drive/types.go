// Package drive provides a client for Google Drive API v3.
// Built on the shared transport layer -- no google-api-go-client dependency.
package drive

import "time"

// DriveFile represents a file or folder in Google Drive.
type DriveFile struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	MIMEType     string    `json:"mimeType"`
	Size         int64     `json:"size,string"`
	ModifiedTime time.Time `json:"modifiedTime"`
	Parents      []string  `json:"parents"`
	WebViewLink  string    `json:"webViewLink"`
	IconLink     string    `json:"iconLink"`
	Starred      bool      `json:"starred"`
	Trashed      bool      `json:"trashed"`
}

// DriveFileList is the API response for file listing.
type driveFileListResponse struct {
	Files            []DriveFile `json:"files"`
	NextPageToken    string      `json:"nextPageToken"`
	IncompleteSearch bool        `json:"incompleteSearch"`
}

// IsFolder reports whether the file is a Google Drive folder.
func (f DriveFile) IsFolder() bool {
	return f.MIMEType == "application/vnd.google-apps.folder"
}

// IsGoogleDoc reports whether the file is a Google Docs native format.
func (f DriveFile) IsGoogleDoc() bool {
	return isGoogleFormat(f.MIMEType)
}

// Google Workspace MIME types that require export (not direct download).
var googleFormats = map[string]string{
	"application/vnd.google-apps.document":     "text/plain",
	"application/vnd.google-apps.spreadsheet":  "text/csv",
	"application/vnd.google-apps.presentation": "text/plain",
	"application/vnd.google-apps.drawing":      "image/png",
}

// isGoogleFormat reports whether a MIME type is a Google Workspace native format.
func isGoogleFormat(mimeType string) bool {
	_, ok := googleFormats[mimeType]
	return ok
}

// DefaultExportFormat returns the default export MIME type for a Google format.
// Returns empty string if the MIME type is not a Google format.
func DefaultExportFormat(mimeType string) string {
	return googleFormats[mimeType]
}
