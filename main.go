package main

import (
	"fmt"
	"log"
	"net/http"
	"soap-server/handler"
	"strings"
	"time"
)

func main() {
	// Get upload directory from environment or use default
	uploadDir := "./uploads"

	// Create a new ServeMux for routing SOAP operations
	soapMux := http.NewServeMux()

	// SOAP endpoint that routes to different operations based on SOAPAction
	soapMux.HandleFunc("/soap", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
			return
		}

		// Check SOAPAction header to determine the operation
		soapAction := r.Header.Get("SOAPAction")

		// Also try to determine operation from the request body
		contentType := r.Header.Get("Content-Type")

		fmt.Printf("[%s] SOAP Request - Method: %s, SOAPAction: %s, ContentType: %s\n",
			getCurrentTime(), r.Method, soapAction, contentType)

		// Route based on SOAPAction header or parse body to determine operation
		if soapAction != "" {
			// Remove quotes from SOAPAction if present
			soapAction = stripQuotes(soapAction)
			switch soapAction {
			case "http://example.com/soap/user/GetUser":
				handler.GetUser(w, r)
				return
			case "http://example.com/soap/user/UploadFile":
				handler.UploadFile(uploadDir)(w, r)
				return
			case "http://example.com/soap/user/UploadFileMTOM":
				handler.UploadFileMTOM(uploadDir)(w, r)
				return
			}
		}

		// Fallback: try to parse the body to determine operation
		body := r.Body
		defer body.Close()

		// Read first 512 bytes to peek at the content
		buf := make([]byte, 512)
		n, _ := body.Read(buf)
		bufStr := string(buf[:n])

		// Route based on content
		if strings.Contains(bufStr, "GetUserRequest") {
			// Reset body for the handler
			r.Body = newReadCloser(bufStr)
			handler.GetUser(w, r)
		} else if strings.Contains(bufStr, "UploadFileMTOMRequest") {
			// Reset body for the handler
			r.Body = newReadCloser(bufStr)
			handler.UploadFileMTOM(uploadDir)(w, r)
		} else if strings.Contains(bufStr, "UploadFileRequest") {
			// Reset body for the handler
			r.Body = newReadCloser(bufStr)
			handler.UploadFile(uploadDir)(w, r)
		} else {
			sendSOAPError(w, "Client", "Unknown operation", "Could not determine SOAP operation from request")
		}
	})

	// Health check endpoint
	soapMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy","service":"SOAP Server"}`))
	})

	// WSDL endpoint
	soapMux.HandleFunc("/wsdl", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		http.ServeFile(w, r, "wsdl/user.wsdl")
	})

	// Start server
	port := ":8080"
	fmt.Printf("===========================================\n")
	fmt.Printf("SOAP Server Starting\n")
	fmt.Printf("===========================================\n")
	fmt.Printf("Server running on: http://localhost%s\n", port)
	fmt.Printf("SOAP endpoint:    http://localhost%s/soap\n", port)
	fmt.Printf("WSDL endpoint:    http://localhost%s/wsdl\n", port)
	fmt.Printf("Health endpoint:  http://localhost%s/health\n", port)
	fmt.Printf("Upload directory: %s\n", uploadDir)
	fmt.Printf("===========================================\n")
	fmt.Printf("Available Operations:\n")
	fmt.Printf("  - GetUser:        Retrieve user information by ID\n")
	fmt.Printf("  - UploadFile:     Upload base64 encoded file\n")
	fmt.Printf("  - UploadFileMTOM: Upload file using MTOM (optimized binary transfer)\n")
	fmt.Printf("===========================================\n\n")

	if err := http.ListenAndServe(port, soapMux); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func getCurrentTime() string {
	return fmt.Sprint(time.Now().Format("2006-01-02 15:04:05"))
}

func stripQuotes(s string) string {
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"') {
		return s[1 : len(s)-1]
	}
	return s
}

func sendSOAPError(w http.ResponseWriter, faultCode, faultString, detail string) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")

	fault := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
    <soap:Body>
        <soap:Fault>
            <faultcode>%s</faultcode>
            <faultstring>%s</faultstring>
            <detail>%s</detail>
        </soap:Fault>
    </soap:Body>
</soap:Envelope>`, faultCode, faultString, detail)

	w.Write([]byte(fault))
}

// readCloser wraps a string to implement io.ReadCloser
type readCloser struct {
	*strings.Reader
}

func newReadCloser(s string) *readCloser {
	return &readCloser{strings.NewReader(s)}
}

func (rc *readCloser) Close() error {
	return nil
}
