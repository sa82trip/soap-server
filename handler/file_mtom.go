package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UploadFileMTOMRequest represents the SOAP request for uploading a file via MTOM
type UploadFileMTOMRequest struct {
	XMLName  xml.Name `xml:"http://example.com/soap/user UploadFileMTOMRequest"`
	FileName string   `xml:"fileName"`
	FileData string   `xml:"fileData"` // Can be base64 or XOP include reference
}

// UploadFileMTOMResponse represents the SOAP response for MTOM file upload
type UploadFileMTOMResponse struct {
	XMLName  xml.Name `xml:"http://example.com/soap/user UploadFileMTOMResponse"`
	FileID   string   `xml:"fileId"`
	FileName string   `xml:"fileName"`
	Size     int64    `xml:"size"`
	Path     string   `xml:"path"`
}

// XOPInclude represents an XOP Include element for MTOM
type XOPInclude struct {
	XMLName   xml.Name `xml:"http://www.w3.org/2004/08/xop/include Include"`
	Href      string   `xml:"href,attr"`
	ContentID string   // Extracted from href (e.g., "cid:example" -> "example")
}

// MultipartPart represents a parsed MIME part
type MultipartPart struct {
	ContentID string
	ContentType string
	Data []byte
}

// UploadFileMTOM handles the UploadFileMTOM SOAP operation with MTOM/XOP support
func UploadFileMTOM(uploadDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")

		fmt.Printf("[%s] MTOM Request - ContentType: %s\n",
			time.Now().Format("2006-01-02 15:04:05"), contentType)

		var fileName string
		var fileData []byte
		var err error

		// Check if this is a MTOM multipart/related request
		if strings.HasPrefix(contentType, "multipart/related") {
			fileName, fileData, err = parseMTOMRequest(r)
			if err != nil {
				sendSOAPError(w, "Client", "Invalid MTOM request", err.Error())
				return
			}
		} else {
			// Fallback to regular SOAP with base64 (for non-MTOM clients)
			fileName, fileData, err = parseBase64SOAPRequest(r)
			if err != nil {
				sendSOAPError(w, "Client", "Invalid SOAP request", err.Error())
				return
			}
		}

		// Validate input
		if fileName == "" {
			sendSOAPError(w, "Client", "Invalid input", "File name is required")
			return
		}

		if len(fileData) == 0 {
			sendSOAPError(w, "Client", "Invalid input", "File data is required")
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
		if err := os.WriteFile(filePath, fileData, 0644); err != nil {
			sendSOAPError(w, "Server", "Internal error", "Failed to save file: "+err.Error())
			return
		}

		// Get file size
		fileSize := int64(len(fileData))

		// Create response
		response := UploadFileMTOMResponse{
			FileID:   fileID,
			FileName: fileName,
			Size:     fileSize,
			Path:     fmt.Sprintf("/uploads/%s", uniqueFileName),
		}

		sendSOAPResponse(w, "UploadFileMTOMResponse", response)

		// Log the upload
		fmt.Printf("[%s] MTOM File uploaded: ID=%s, Name=%s, Size=%d bytes, Path=%s\n",
			time.Now().Format("2006-01-02 15:04:05"), fileID, fileName, fileSize, filePath)
	}
}

// parseMTOMRequest parses a MTOM multipart/related SOAP request
func parseMTOMRequest(r *http.Request) (string, []byte, error) {
	contentType := r.Header.Get("Content-Type")

	// Parse the Content-Type header to get the boundary
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse content-type: %w", err)
	}

	boundary, ok := params["boundary"]
	if !ok {
		return "", nil, fmt.Errorf("boundary not found in content-type")
	}

	// Read the entire body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Parse multipart
	mr := multipart.NewReader(bytes.NewReader(body), boundary)

	var parts []MultipartPart
	var soapPart string

	// Read all parts
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, fmt.Errorf("failed to read multipart part: %w", err)
		}

		contentID := part.Header.Get("Content-ID")
		// Remove angle brackets from Content-ID if present
		contentID = strings.Trim(contentID, "<>")

		partContentType := part.Header.Get("Content-Type")

		data, err := io.ReadAll(part)
		if err != nil {
			part.Close()
			return "", nil, fmt.Errorf("failed to read part data: %w", err)
		}
		part.Close()

		if strings.Contains(partContentType, "application/xop+xml") ||
		   strings.Contains(partContentType, "text/xml") ||
		   strings.Contains(partContentType, "application/soap+xml") {
			// This is the SOAP envelope part
			soapPart = string(data)
		} else {
			// This is a binary attachment part
			parts = append(parts, MultipartPart{
				ContentID: contentID,
				ContentType: partContentType,
				Data: data,
			})
		}
	}

	// Parse the SOAP envelope to extract file name and XOP references
	fileName, xopRefs, err := parseMTOMSOAPEnvelope(soapPart)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse SOAP envelope: %w", err)
	}

	// Resolve XOP references to actual binary data
	var fileData []byte
	for _, xopRef := range xopRefs {
		found := false
		for _, part := range parts {
			if part.ContentID == xopRef {
				fileData = part.Data
				found = true
				break
			}
		}
		if !found {
			return "", nil, fmt.Errorf("XOP reference not found: %s", xopRef)
		}
	}

	if len(fileData) == 0 {
		return "", nil, fmt.Errorf("no file data found in MTOM request")
	}

	return fileName, fileData, nil
}

// parseMTOMSOAPEnvelope parses the SOAP envelope from MTOM request
func parseMTOMSOAPEnvelope(soapEnvelope string) (string, []string, error) {
	// Parse the XML to extract the request
	var envelope struct {
		XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
		Body    struct {
			XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
			Request struct {
				XMLName  xml.Name `xml:"http://example.com/soap/user UploadFileMTOMRequest"`
				FileName string   `xml:"fileName"`
				FileData string   `xml:"fileData"`
			} `xml:"UploadFileMTOMRequest"`
		}
	}

	if err := xml.Unmarshal([]byte(soapEnvelope), &envelope); err != nil {
		return "", nil, fmt.Errorf("XML parse error: %w", err)
	}

	fileName := envelope.Body.Request.FileName
	fileDataElement := envelope.Body.Request.FileData

	var xopRefs []string

	// Check if fileData contains an XOP Include reference
	// XOP include format: <xop:Include xmlns:xop="http://www.w3.org/2004/08/xop/include" href="cid:..."/>
	if strings.Contains(fileDataElement, "<xop:Include") || strings.Contains(fileDataElement, "Include") {
		// Extract Content-ID from XOP Include
		re := regexp.MustCompile(`href=["']cid:([^"']+)["']`)
		matches := re.FindStringSubmatch(fileDataElement)
		if len(matches) > 1 {
			xopRefs = append(xopRefs, matches[1])
		}
	}

	return fileName, xopRefs, nil
}

// parseBase64SOAPRequest parses a regular SOAP request with base64 encoded file data
func parseBase64SOAPRequest(r *http.Request) (string, []byte, error) {
	var soapEnvelope struct {
		XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
		Body    struct {
			XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
			Request UploadFileMTOMRequest `xml:"UploadFileMTOMRequest"`
		}
	}

	if err := xml.NewDecoder(r.Body).Decode(&soapEnvelope); err != nil {
		return "", nil, fmt.Errorf("XML decode error: %w", err)
	}

	fileName := soapEnvelope.Body.Request.FileName
	fileData := soapEnvelope.Body.Request.FileData

	// Decode base64
	decodedData, err := base64.StdEncoding.DecodeString(fileData)
	if err != nil {
		return "", nil, fmt.Errorf("base64 decode error: %w", err)
	}

	return fileName, decodedData, nil
}
