package handler

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UploadFileRequest represents the SOAP request for uploading a file
type UploadFileRequest struct {
	XMLName  xml.Name `xml:"http://example.com/soap/user UploadFileRequest"`
	FileName string   `xml:"fileName"`
	FileData string   `xml:"fileData"`
}

// UploadFileResponse represents the SOAP response for file upload
type UploadFileResponse struct {
	XMLName  xml.Name `xml:"http://example.com/soap/user UploadFileResponse"`
	FileID   string   `xml:"fileId"`
	FileName string   `xml:"fileName"`
	Size     int64    `xml:"size"`
	Path     string   `xml:"path"`
}

// FileUploadResult stores the result of a file upload
type FileUploadResult struct {
	FileID   string
	FileName string
	Size     int64
	Path     string
}

// UploadFile handles the UploadFile SOAP operation
func UploadFile(uploadDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read and parse the SOAP request body
		var soapEnvelope struct {
			XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
			Body    struct {
				XMLName xml.Name          `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
				Request UploadFileRequest `xml:"UploadFileRequest"`
			}
		}

		if err := xml.NewDecoder(r.Body).Decode(&soapEnvelope); err != nil {
			sendSOAPError(w, "Client", "Invalid XML format", err.Error())
			return
		}

		fileName := soapEnvelope.Body.Request.FileName
		fileData := soapEnvelope.Body.Request.FileData

		// Validate input
		if fileName == "" {
			sendSOAPError(w, "Client", "Invalid input", "File name is required")
			return
		}

		if fileData == "" {
			sendSOAPError(w, "Client", "Invalid input", "File data is required")
			return
		}

		// Decode base64 file data
		decodedData, err := base64.StdEncoding.DecodeString(fileData)
		if err != nil {
			sendSOAPError(w, "Client", "Invalid file data", "Failed to decode base64 data: "+err.Error())
			return
		}

		// Generate unique file ID
		fileID := uuid.New().String()

		// Create upload directory if it doesn't exist
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			sendSOAPError(w, "Server", "Internal error", "Failed to create upload directory: "+err.Error())
			return
		}

		// Sanitize filename and create file path
		safeFileName := sanitizeFileName(fileName)
		uniqueFileName := fmt.Sprintf("%s_%s", fileID, safeFileName)
		filePath := filepath.Join(uploadDir, uniqueFileName)

		// Write file to disk
		if err := os.WriteFile(filePath, decodedData, 0644); err != nil {
			sendSOAPError(w, "Server", "Internal error", "Failed to save file: "+err.Error())
			return
		}

		// Get file size
		fileSize := int64(len(decodedData))

		// Create response
		response := UploadFileResponse{
			FileID:   fileID,
			FileName: fileName,
			Size:     fileSize,
			Path:     fmt.Sprintf("/uploads/%s", uniqueFileName),
		}

		sendSOAPResponse(w, "UploadFileResponse", response)

		// Log the upload
		fmt.Printf("[%s] File uploaded: ID=%s, Name=%s, Size=%d bytes, Path=%s\n",
			time.Now().Format("2006-01-02 15:04:05"), fileID, fileName, fileSize, filePath)
	}
}

// sanitizeFileName removes potentially dangerous characters from filename
func sanitizeFileName(name string) string {
	// Remove path separators and dangerous characters
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.ReplaceAll(name, "\x00", "")

	// Limit length
	if len(name) > 255 {
		name = name[:255]
	}

	return name
}
