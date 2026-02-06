package handler

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
)

// User represents a user in the system
type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
}

// Mock user database
var userDB = map[string]User{
	"1": {ID: "1", Name: "홍길동", Email: "hong@example.com", CreatedAt: "2024-01-01"},
	"2": {ID: "2", Name: "김철수", Email: "kim@example.com", CreatedAt: "2024-01-15"},
	"3": {ID: "3", Name: "이영희", Email: "lee@example.com", CreatedAt: "2024-02-01"},
}

// GetUserRequest represents the SOAP request for getting a user
type GetUserRequest struct {
	XMLName xml.Name `xml:"http://example.com/soap/user GetUserRequest"`
	ID      string   `xml:"id"`
}

// GetUserResponse represents the SOAP response for getting a user
type GetUserResponse struct {
	XMLName   xml.Name `xml:"http://example.com/soap/user GetUserResponse"`
	ID        string   `xml:"id"`
	Name      string   `xml:"name"`
	Email     string   `xml:"email"`
	CreatedAt string   `xml:"createdAt"`
}

// GetUser handles the GetUser SOAP operation
func GetUser(w http.ResponseWriter, r *http.Request) {
	// Read and parse the SOAP request body
	var soapEnvelope struct {
		XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
		Body    struct {
			XMLName xml.Name        `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
			Request GetUserRequest  `xml:"GetUserRequest"`
		}
	}

	if err := xml.NewDecoder(r.Body).Decode(&soapEnvelope); err != nil {
		sendSOAPError(w, "Client", "Invalid XML format", err.Error())
		return
	}

	userID := soapEnvelope.Body.Request.ID

	// Look up the user
	user, exists := userDB[userID]
	if !exists {
		sendSOAPError(w, "Client", "User not found", fmt.Sprintf("User with ID %s not found", userID))
		return
	}

	// Create SOAP response
	response := GetUserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}

	sendSOAPResponse(w, "GetUserResponse", response)
}

// sendSOAPResponse sends a SOAP response
func sendSOAPResponse(w http.ResponseWriter, elementName string, body interface{}) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")

	envelope := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
    <soap:Body>
        <%s xmlns="http://example.com/soap/user">
%s
        </%s>
    </soap:Body>
</soap:Envelope>`, elementName, marshalXML(body), elementName)

	w.Write([]byte(envelope))
}

// marshalXML converts a struct to XML elements
func marshalXML(v interface{}) string {
	var result strings.Builder

	// Manually build XML based on struct type
	switch t := v.(type) {
	case GetUserResponse:
		result.WriteString(fmt.Sprintf("<id>%s</id>\n        ", t.ID))
		result.WriteString(fmt.Sprintf("<name>%s</name>\n        ", t.Name))
		result.WriteString(fmt.Sprintf("<email>%s</email>\n        ", t.Email))
		result.WriteString(fmt.Sprintf("<createdAt>%s</createdAt>", t.CreatedAt))
	case UploadFileResponse:
		result.WriteString(fmt.Sprintf("<fileId>%s</fileId>\n        ", t.FileID))
		result.WriteString(fmt.Sprintf("<fileName>%s</fileName>\n        ", t.FileName))
		result.WriteString(fmt.Sprintf("<size>%d</size>\n        ", t.Size))
		result.WriteString(fmt.Sprintf("<path>%s</path>", t.Path))
	case UploadFileMTOMResponse:
		result.WriteString(fmt.Sprintf("<fileId>%s</fileId>\n        ", t.FileID))
		result.WriteString(fmt.Sprintf("<fileName>%s</fileName>\n        ", t.FileName))
		result.WriteString(fmt.Sprintf("<size>%d</size>\n        ", t.Size))
		result.WriteString(fmt.Sprintf("<path>%s</path>", t.Path))
	}

	return result.String()
}

// sendSOAPError sends a SOAP fault response
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
