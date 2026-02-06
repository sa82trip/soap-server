# SOAP Test System - Flow Chart Documentation

## 1. System Overview

This SOAP test system demonstrates a complete SOAP 1.1 web service implementation with two components:

- **SOAP Server (Go)**: HTTP server on port 8080 implementing three SOAP operations
- **SOAP Client (Spring Boot)**: REST API on port 8081 that bridges REST requests to SOAP calls

### Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Server | Go 1.21 | SOAP endpoint implementation |
| Client | Spring Boot 3.2.2 | REST-to-SOAP gateway |
| SOAP Version | SOAP 1.1 | Message protocol |
| MTOM | SAAJ + JAXB | Binary optimization |
| Documentation | SpringDoc OpenAPI | Swagger UI |

### SOAP Operations

| Operation | SOAPAction | Description |
|-----------|------------|-------------|
| GetUser | `http://example.com/soap/user/GetUser` | Retrieve user by ID |
| UploadFile | `http://example.com/soap/user/UploadFile` | Upload file (Base64) |
| UploadFileMTOM | `http://example.com/soap/user/UploadFileMTOM` | Upload file (MTOM) |

### REST Endpoints

| Method | Path | SOAP Operation |
|--------|------|----------------|
| GET | `/api/user/{id}` | GetUser |
| POST | `/api/upload` | UploadFile (Base64) |
| POST | `/api/upload-mtom` | UploadFileMTOM |
| GET | `/api/health` | Health check |
| GET | `/wsdl` | WSDL download |

---

## 2. High-Level Architecture

```mermaid
graph TB
    subgraph Client["Client Side - Port 8081 (Spring Boot)"]
        REST[REST Client<br/>cURL/Swagger]
        TC[TestController<br/>@RestController]
        SC[SoapConfig<br/>Standard SOAP]
        SMC[SoapMtomConfig<br/>MTOM Enabled]
        USC[UserSoapClient<br/>Base64]
        USMC[UserSoapMtomClient<br/>MTOM]
        WST[WebServiceTemplate<br/>Spring WS]

        REST -->|"GET /api/user/1"| TC
        REST -->|"POST /api/upload"| TC
        REST -->|"POST /api/upload-mtom"| TC

        TC -->|"getUser"| USC
        TC -->|"uploadFile"| USC
        TC -->|"uploadFileMTOM"| USMC

        SC -->|"Creates"| USC
        SMC -->|"Creates"| USMC

        USC -->|"marshalSendAndReceive"| WST
        USMC -->|"marshalSendAndReceive"| WST
    end

    subgraph Server["Server Side - Port 8080 (Go)"]
        MUX[HTTP ServeMux<br/>/soap Router]
        H[Health Handler<br/>/health]
        WSDL[WSDL Handler<br/>/wsdl]
        GU[GetUser Handler<br/>user.go]
        UF[UploadFile Handler<br/>file.go<br/>Base64]
        UFM[UploadFileMTOM Handler<br/>file_mtom.go<br/>MIME Parsing]
        DB[(Mock User DB<br/>map[string]User)]
        FS[(File System<br/>./uploads)]

        MUX -->|"SOAPAction"| GU
        MUX -->|"SOAPAction"| UF
        MUX -->|"SOAPAction"| UFM
        MUX --> H
        MUX --> WSDL

        GU --> DB
        UF --> FS
        UFM --> FS
    end

    WST ==>|"SOAP/HTTP<br/>text/xml"| MUX
    WST ==>|"SOAP/HTTP<br/>multipart/related"| MUX

    style REST fill:#e1f5fe
    style TC fill:#fff9c4
    style WST fill:#c8e6c9
    style MUX fill:#ffccbc
    style DB fill:#d1c4e9
    style FS fill:#d1c4e9
```

### Architecture Notes

- **Dual Client Strategy**: Separate client beans for Base64 (`UserSoapClient`) and MTOM (`UserSoapMtomClient`) operations
- **SOAPAction Routing**: Server uses SOAPAction header as primary routing mechanism
- **Fallback Routing**: Server parses request body when SOAPAction is missing
- **MTOM Detection**: Server detects MTOM requests via `Content-Type: multipart/related`

---

## 3. Sequence Diagrams

### 3.1 GetUser Operation Flow

```mermaid
sequenceDiagram
    autonumber
    participant RC as REST Client<br/>(cURL/Swagger)
    participant TC as TestController
    participant USC as UserSoapClient
    participant WST as WebServiceTemplate
    participant SVR as SOAP Server<br/>(Go main.go)
    participant GU as GetUser Handler
    participant DB as Mock User DB

    RC->>TC: GET /api/user/1
    TC->>TC: log.info("REST Request")
    TC->>USC: getUser("1")

    USC->>USC: Create GetUserRequest("1")
    USC->>WST: marshalSendAndReceive(request)<br/>+ SoapActionCallback

    Note over WST: JAXB marshals to XML
    WST->>SVR: POST /soap<br/>SOAPAction: GetUser<br/>Content-Type: text/xml

    SVR->>SVR: Read SOAPAction header
    SVR->>GU: Route to GetUser handler

    GU->>GU: XML Decode SOAP Envelope
    GU->>GU: Extract ID from Body

    GU->>DB: userDB["1"]
    DB-->>GU: User{id:"1", name:"홍길동"}

    GU->>GU: Build SOAP Response
    GU-->>SVR: SOAP XML Response

    SVR-->>WST: HTTP 200<br/>Content-Type: text/xml

    Note over WST: JAXB unmarshals XML
    WST-->>USC: GetUserResponse

    USC-->>TC: GetUserResponse
    TC->>TC: Build Map result
    TC-->>RC: HTTP 200 JSON<br/>{"id":"1","name":"홍길동",...}
```

### 3.2 UploadFile (Base64) Operation Flow

```mermaid
sequenceDiagram
    autonumber
    participant RC as REST Client<br/>(cURL/Swagger)
    participant TC as TestController
    participant USC as UserSoapClient
    participant WST as WebServiceTemplate
    participant SVR as SOAP Server<br/>(Go main.go)
    participant UF as UploadFile Handler
    participant FS as File System

    RC->>TC: POST /api/upload<br/>multipart/form-data

    TC->>TC: file.getBytes()
    TC->>TC: Base64.getEncoder()<br/>Encode to base64

    Note over TC: 1MB file → ~1.33MB base64

    TC->>USC: uploadFile(fileName, base64Data)

    USC->>USC: Create UploadFileRequest
    USC->>WST: marshalSendAndReceive<br/>+ SoapActionCallback(UploadFile)

    Note over WST: JAXB marshals XML<br/>base64 embedded in XML
    WST->>SVR: POST /soap<br/>SOAPAction: UploadFile<br/>Content-Type: text/xml

    SVR->>UF: Route to UploadFile handler

    UF->>UF: XML Decode SOAP Envelope
    UF->>UF: Extract fileName, fileData

    UF->>UF: base64.StdEncoding.DecodeString(fileData)

    Note over UF: Decode: ~1.33MB → 1MB

    UF->>UF: Generate UUID for fileId
    UF->>UF: sanitizeFileName()
    UF->>FS: os.WriteFile(path, data)

    UF->>UF: Build SOAP Response
    UF-->>SVR: SOAP XML Response

    SVR-->>WST: HTTP 200<br/>UploadFileResponse

    WST-->>USC: Unmarshal to Java object
    USC-->>TC: UploadFileResponse

    TC-->>RC: HTTP 200 JSON<br/>{"fileId":"uuid",...}

    Note over RC,FS: Total Transfer: 1.33MB (+ headers)
```

### 3.3 UploadFileMTOM Operation Flow

```mermaid
sequenceDiagram
    autonumber
    participant RC as REST Client<br/>(cURL/Swagger)
    participant TC as TestController
    participant USMC as UserSoapMtomClient
    participant MT as MTOM Marshaller<br/>(JAXB + SAAJ)
    participant SVR as SOAP Server<br/>(Go main.go)
    participant UFM as UploadFileMTOM Handler
    participant MIME as MIME Parser
    participant FS as File System

    RC->>TC: POST /api/upload-mtom<br/>multipart/form-data

    TC->>TC: file.getBytes()

    Note over TC: NO Base64 encoding!<br/>Raw bytes preserved

    TC->>USMC: uploadFileMTOM(fileName, fileBytes)

    USMC->>MT: marshalSendAndReceive<br/>+ SoapActionCallback

    Note over MT: MTOM detects byte[] field<br/>Creates XOP Include reference

    MT->>SVR: POST /soap<br/>SOAPAction: UploadFileMTOM<br/>Content-Type: multipart/related<br/>boundary=...<br/>type="application/xop+xml"

    Note over SVR: MIME Multipart Message:<br/>--boundary<br/>Content-Type: application/xop+xml<br/><SOAP Envelope><br/>  <fileData><xop:Include href="cid:..."/></><br/></SOAP Envelope><br/>--boundary<br/>Content-ID: <...><br/>Content-Type: application/octet-stream<br/>(RAW BINARY DATA)<br/>--boundary--

    SVR->>UFM: Route to UploadFileMTOM handler

    UFM->>MIME: parseMTOMRequest()

    MIME->>MIME: Parse boundary from Content-Type
    MIME->>MIME: multipart.NewReader()

    MIME->>MIME: Read Part 1: SOAP XML
    MIME->>MIME: Read Part 2: Binary attachment

    Note over MIME: Extract Content-ID from headers<br/>Remove angle brackets

    MIME->>UFM: Return fileName + binaryData

    UFM->>UFM: Parse XOP href from XML<br/>Match Content-ID to attachment

    Note over UFM: Resolve XOP Include:<br/>href="cid:abc123" → Content-ID: abc123

    UFM->>UFM: Generate UUID
    UFM->>FS: os.WriteFile(path, binaryData)

    UFM->>UFM: Build SOAP Response
    UFM-->>SVR: SOAP XML Response

    SVR-->>MT: HTTP 200<br/>UploadFileMTOMResponse

    MT-->>USMC: Unmarshal response

    USMC-->>TC: UploadFileMTOMResponse
    TC-->>RC: HTTP 200 JSON<br/>{"uploadMethod":"MTOM",...}

    Note over RC,FS: Total Transfer: ~1.01MB (+ MIME headers)
```

---

## 4. SOAP Message Processing Flowchart

```mermaid
flowchart TD
    START([HTTP Request Received<br/>POST /soap])

    CHECK_METHOD{Method = POST?}
    METHOD_ERROR[HTTP 405<br/>Method Not Allowed]

    GET_SOAP_ACTION[Read SOAPAction Header]
    CHECK_ACTION{SOAPAction Present?}
    GET_CONTENT_TYPE[Read Content-Type Header]

    ROUTE_ACTION[Strip quotes from SOAPAction]
    SWITCH_ACTION{Which Operation?}

    GET_USER[http://example.com/.../GetUser]
    UPLOAD_FILE[http://example.com/.../UploadFile]
    UPLOAD_MTOM[http://example.com/.../UploadFileMTOM]

    CALL_GET_USER[Call handler.GetUser]
    CALL_UPLOAD_FILE[Call handler.UploadFile]
    CALL_UPLOAD_MTOM[Call handler.UploadFileMTOM]

    FALLBACK[Read first 512 bytes of body]
    PARSE_BODY{Parse body content}

    BODY_GET_USER[Contains 'GetUserRequest']
    BODY_UPLOAD_MTOM[Contains 'UploadFileMTOMRequest']
    BODY_UPLOAD_FILE[Contains 'UploadFileRequest']

    UNKNOWN[SOAP Fault<br/>Unknown operation]

    HANDLE_GET_USER[Process GetUser<br/>Query Mock DB]
    HANDLE_UPLOAD[Process UploadFile<br/>Base64 Decode → Save]
    HANDLE_MTOM{Content-Type<br/>multipart/related?}

    PARSE_MTOM[Parse MIME multipart<br/>Extract binary attachment<br/>Resolve XOP Include]
    PARSE_BASE64[Parse SOAP XML<br/>Base64 decode fileData]

    SAVE_FILE[Generate UUID<br/>Sanitize filename<br/>Save to ./uploads]

    ERROR([SOAP Fault Response])
    SUCCESS([SOAP Response])

    START --> CHECK_METHOD
    CHECK_METHOD -->|No| METHOD_ERROR
    CHECK_METHOD -->|Yes| GET_SOAP_ACTION

    GET_SOAP_ACTION --> CHECK_ACTION
    CHECK_ACTION -->|Yes| ROUTE_ACTION
    CHECK_ACTION -->|No| GET_CONTENT_TYPE

    ROUTE_ACTION --> SWITCH_ACTION
    SWITCH_ACTION -->|GetUser| CALL_GET_USER
    SWITCH_ACTION -->|UploadFile| CALL_UPLOAD_FILE
    SWITCH_ACTION -->|UploadFileMTOM| CALL_UPLOAD_MTOM

    CALL_GET_USER --> HANDLE_GET_USER
    CALL_UPLOAD_FILE --> HANDLE_UPLOAD
    CALL_UPLOAD_MTOM --> HANDLE_MTOM

    GET_CONTENT_TYPE --> FALLBACK
    FALLBACK --> PARSE_BODY

    PARSE_BODY --> BODY_GET_USER
    PARSE_BODY --> BODY_UPLOAD_MTOM
    PARSE_BODY --> BODY_UPLOAD_FILE
    PARSE_BODY -->|Unknown| UNKNOWN

    BODY_GET_USER --> HANDLE_GET_USER
    BODY_UPLOAD_FILE --> HANDLE_UPLOAD
    BODY_UPLOAD_MTOM --> HANDLE_MTOM

    HANDLE_GET_USER --> SUCCESS
    HANDLE_UPLOAD --> SAVE_FILE
    SAVE_FILE --> SUCCESS

    HANDLE_MTOM -->|Yes| PARSE_MTOM
    HANDLE_MTOM -->|No| PARSE_BASE64

    PARSE_MTOM --> SAVE_FILE
    PARSE_BASE64 --> SAVE_FILE

    UNKNOWN --> ERROR
    METHOD_ERROR --> ERROR

    style START fill:#c8e6c9
    style SUCCESS fill:#c8e6c9
    style ERROR fill:#ffcdd2
    style HANDLE_MTOM fill:#fff9c4
    style PARSE_MTOM fill:#e1bee7
```

### Key Processing Points

1. **SOAPAction Priority**: Server checks SOAPAction header first (primary routing)
2. **Content-Based Fallback**: Parses request body for operation name when SOAPAction is missing
3. **MTOM Detection**: Checks `Content-Type: multipart/related` to detect MTOM
4. **XOP Resolution**: Matches `href="cid:..."` to MIME part `Content-ID`

---

## 5. Spring Component Interaction

```mermaid
graph TB
    subgraph SpringContext["Spring ApplicationContext"]
        subgraph Config["@Configuration Classes"]
            SC[SoapConfig]
            SMC[SoapMtomConfig]
            OAC[OpenApiConfig]
        end

        subgraph Beans["@Beans Created"]
            JM[Jaxb2Marshaller<br/>Standard]
            JMM[Jaxb2Marshaller<br/>mtomEnabled=true]
            SMF[SaajSoapMessageFactory<br/>SOAP 1.1]
            WST[WebServiceTemplate<br/>Standard]
            MWST[WebServiceTemplate<br/>MTOM]
            USC[UserSoapClient]
            USMC[UserSoapMtomClient]
        end

        subgraph Controllers["@RestController"]
            TC[TestController<br/>@Autowired clients]
        end

        subgraph Models["JAXB Models"]
            GUR[GetUserRequest/Response]
            UFR[UploadFileRequest/Response]
            UFMR[UploadFileMTOMRequest/Response]
        end

        SC -->|"Creates"| JM
        SC -->|"Creates"| SMF
        SC -->|"Creates"| WST
        SC -->|"Creates"| USC

        SMC -->|"Creates"| JMM
        SMC -->|"Creates"| MWST
        SMC -->|"Creates"| USMC

        OAC -.->|"Configures"| TC

        TC -->|"Injects"| USC
        TC -->|"Injects"| USMC

        USC -->|"Uses"| WST
        USMC -->|"Uses"| MWST

        WST -->|"Uses"| JM
        WST -->|"Uses"| SMF

        MWST -->|"Uses"| JMM
        MWST -->|"Uses"| SMF

        JM -.->|"Marshals/Unmarshals"| GUR
        JM -.->|"Marshals/Unmarshals"| UFR

        JMM -.->|"Marshals/Unmarshals"| UFMR
    end

    style SC fill:#e3f2fd
    style SMC fill:#f3e5f5
    style TC fill:#fff9c4
    style WST fill:#c8e6c9
    style MWST fill:#c8e6c9
```

### Dependency Injection Flow

1. **Configuration Classes**: `@Configuration` classes define bean creation methods
2. **Marshaller Beans**: Separate marshallers for standard and MTOM operations
3. **Template Beans**: `WebServiceTemplate` beans use appropriate marshaller/message factory
4. **Client Beans**: Client classes receive template via constructor injection
5. **Controller Injection**: `TestController` receives both clients via `@Autowired`

---

## 6. Data Flow Comparison: Base64 vs MTOM

```mermaid
flowchart LR
    subgraph Upload["File Upload Process - 1MB File"]
        direction TB
        CLIENT[REST Client<br/>1MB file.jpg]

        subgraph Base64Flow["Base64 Path"]
            direction TB
            B64_ENC[Base64 Encode<br/>1MB → 1.33MB]
            B64_XML[Embed in XML<br/><fileData>base64...</>]
            B64_HTTP[HTTP Transfer<br/>~1.33MB + headers]
            B64_DEC[Base64 Decode<br/>1.33MB → 1MB]
            B64_SAVE[Save to Disk<br/>1MB]
        end

        subgraph MTOMFlow["MTOM Path"]
            direction TB
            MTOM_XOP[Create XOP Ref<br/><xop:Include href="cid:.."/>]
            MTOM_MIME[MIME Multipart<br/>Part 1: SOAP XML ~1KB<br/>Part 2: Binary 1MB]
            MTOM_HTTP[HTTP Transfer<br/>~1.01MB + MIME headers]
            MTOM_RESOLVE[Resolve XOP<br/>Match Content-ID]
            MTOM_SAVE[Save to Disk<br/>1MB]
        end

        CLIENT --> B64_ENC
        CLIENT --> MTOM_XOP

        B64_ENC --> B64_XML
        MTOM_XOP --> MTOM_MIME

        B64_XML --> B64_HTTP
        MTOM_MIME --> MTOM_HTTP

        B64_HTTP --> B64_DEC
        MTOM_HTTP --> MTOM_RESOLVE

        B64_DEC --> B64_SAVE
        MTOM_RESOLVE --> MTOM_SAVE
    end

    style B64_HTTP fill:#ffcdd2
    style MTOM_HTTP fill:#c8e6c9
    style B64_ENC fill:#ffecb3
    style MTOM_MIME fill:#e1bee7
```

### Size Comparison Table

| Stage | Base64 | MTOM | Savings |
|-------|--------|------|---------|
| Original File | 1 MB | 1 MB | - |
| Encoding | +33% | +1% | ~32% |
| Transfer Size | ~1.33 MB | ~1.01 MB | ~320 KB |
| Decode | Required | Not needed | CPU savings |
| Memory | Higher (copy) | Lower (stream) | - |

---

## 7. MTOM vs Base64 Technical Comparison

```mermaid
graph LR
    subgraph Base64["Base64 Encoding"]
        B1[Binary Data]
        B2[Base64 Encode<br/>A-Za-z0-9+/ characters]
        B3[Embed in XML<br/><fileData>SGVsbG8...8K</fileData>]
        B4[Transfer as text/xml]
        B5[Base64 Decode]
        B6[Binary Data]
    end

    subgraph MTOM["MTOM Encoding"]
        M1[Binary Data]
        M2[Create XOP Reference<br/><fileData><xop:Include<br/>href="cid:uuid"/></fileData>]
        M3[MIME Multipart<br/>Part 1: XML with XOP<br/>Part 2: Binary attachment<br/>Content-ID: uuid]
        M4[Transfer as multipart/related]
        M5[Extract by Content-ID<br/>No decoding needed]
        M6[Binary Data]
    end

    style B3 fill:#ffcdd2
    style M3 fill:#c8e6c9
    style B5 fill:#ffecb3
    style M5 fill:#c8e6c9
```

### Technical Differences

| Aspect | Base64 | MTOM |
|--------|--------|------|
| **Content-Type** | `text/xml` | `multipart/related; boundary=...` |
| **Data Format** | Text in XML body | Binary MIME attachment |
| **XML Structure** | `<fileData>base64string</fileData>` | `<fileData><xop:Include href="cid:.."/></fileData>` |
| **Encoding** | 64-character alphabet | Raw binary (8-bit) |
| **Size Overhead** | ~33% | ~1% (MIME headers only) |
| **Processing** | Encode/Decode required | Direct byte copy |
| **CPU Usage** | Higher | Lower |
| **Memory** | Requires full copy | Can stream |
| **Standards** | XML Schema | MTOM 1.0 + XOP |

---

## 8. Error Handling Flowchart

```mermaid
flowchart TD
    START([Request Received])

    TRY_PROCESS{Try Processing}
    ERR_TYPE[Error Type Classification]

    CLIENT_ERR[Client Errors<br/>HTTP 4xx]
    SERVER_ERR[Server Errors<br/>HTTP 5xx]

    C_INVALID_XML{Invalid XML?<br/>Parse Error}
    C_MISSING_FIELD{Missing Required Field?<br/>fileName, fileData}
    C_INVALID_DATA{Invalid Data?<br/>Bad base64}
    C_NOT_FOUND{User Not Found?<br/>ID invalid}

    S_INTERNAL{Internal Error?<br/>I/O, System}
    S_DB_ERROR{Database Error?<br/>Mock DB lookup}

    S_DIR_CREATE[Failed to create<br/>upload directory]
    S_FILE_WRITE[Failed to write<br/>file to disk]

    BUILD_FAULT[Build SOAP Fault<br/><soap:Fault><br/>  <faultcode>code</><br/>  <faultstring>msg</><br/>  <detail>details</><br/></soap:Fault>]

    SET_HEADERS[Content-Type: text/xml<br/>charset=utf-8]

    SEND_FAULT([Send SOAP Fault Response])

    START --> TRY_PROCESS
    TRY_PROCESS -->|Error| ERR_TYPE

    ERR_TYPE -->|"Client"*| CLIENT_ERR
    ERR_TYPE -->|"Server"*| SERVER_ERR

    CLIENT_ERR --> C_INVALID_XML
    CLIENT_ERR --> C_MISSING_FIELD
    CLIENT_ERR --> C_INVALID_DATA
    CLIENT_ERR --> C_NOT_FOUND

    SERVER_ERR --> S_INTERNAL
    SERVER_ERR --> S_DB_ERROR
    SERVER_ERR --> S_DIR_CREATE
    SERVER_ERR --> S_FILE_WRITE

    C_INVALID_XML --> BUILD_FAULT
    C_MISSING_FIELD --> BUILD_FAULT
    C_INVALID_DATA --> BUILD_FAULT
    C_NOT_FOUND --> BUILD_FAULT

    S_INTERNAL --> BUILD_FAULT
    S_DB_ERROR --> BUILD_FAULT
    S_DIR_CREATE --> BUILD_FAULT
    S_FILE_WRITE --> BUILD_FAULT

    BUILD_FAULT --> SET_HEADERS
    SET_HEADERS --> SEND_FAULT

    style START fill:#c8e6c9
    style SEND_FAULT fill:#ffcdd2
    style BUILD_FAULT fill:#fff9c4
    style CLIENT_ERR fill:#ffecb3
    style SERVER_ERR fill:#ffab91
```

### SOAP Fault Structure

```xml
<soap:Fault>
    <faultcode>Client | Server</faultcode>
    <faultstring>Human-readable error message</faultstring>
    <detail>Detailed error information</detail>
</soap:Fault>
```

### Error Classification

| Category | Fault Code | Examples |
|----------|------------|----------|
| **Client** | `Client` | Invalid XML, missing fields, bad base64, user not found |
| **Server** | `Server` | I/O errors, directory creation failures, file write errors |

---

## 9. Complete End-to-End Request Flow

```mermaid
sequenceDiagram
    autonumber
    participant User as End User
    participant REST as REST Client<br/>(cURL/Swagger)
    participant SB as Spring Boot<br/>(Port 8081)
    participant GO as Go Server<br/>(Port 8080)
    participant DB as Mock Database
    participant FS as File System

    User->>REST: Upload file.txt (1MB)

    rect rgb(200, 230, 200)
        Note over REST,SB: Base64 Path (/api/upload)
        REST->>SB: POST /api/upload
        SB->>SB: Base64 encode (+33%)
        SB->>GO: SOAP request (1.33MB)
        GO->>GO: Base64 decode
        GO->>FS: Save file (1MB)
        GO-->>SB: SOAP response
        SB-->>REST: JSON response
    end

    User->>REST: Upload file.txt (1MB)

    rect rgb(230, 200, 255)
        Note over REST,SB: MTOM Path (/api/upload-mtom)
        REST->>SB: POST /api/upload-mtom
        SB->>SB: No encoding (raw bytes)
        SB->>GO: MTOM request (1.01MB)
        Note over GO: MIME multipart<br/>SOAP XML + Binary attachment
        GO->>GO: Extract from MIME
        GO->>FS: Save file (1MB)
        GO-->>SB: SOAP response
        SB-->>REST: JSON response
    end

    User->>REST: GET user info

    rect rgb(255, 255, 200)
        Note over REST,SB: Query Path (/api/user/1)
        REST->>SB: GET /api/user/1
        SB->>GO: SOAP GetUser request
        GO->>DB: Lookup user ID=1
        DB-->>GO: User data
        GO-->>SB: SOAP response
        SB-->>REST: JSON response
    end
```

---

## 10. Legend and Symbols

### Diagram Conventions

| Symbol | Meaning |
|--------|---------|
| `→` | Synchronous call |
| `-->` | Response/Return |
| `==>` | HTTP/SOAP over network |
| `-.->` | Configuration reference |
| `[...]` | Storage/Database |
| `(...)` | Grouped component |

### Color Coding

| Color | Usage |
|-------|-------|
| 🟢 Green | Success/Healthy/Client |
| 🔴 Red | Error/Failure |
| 🟡 Yellow | Processing/Warning |
| 🟣 Purple | MTOM/MIME processing |
| 🔵 Blue | Configuration/Setup |

---

## Appendix: Key Code References

### Server Routing Logic (`server/main.go`)

```go
// Line 26-49: SOAPAction-based routing
soapAction := r.Header.Get("SOAPAction")
soapAction = stripQuotes(soapAction)
switch soapAction {
case "http://example.com/soap/user/GetUser":
    handler.GetUser(w, r)
case "http://example.com/soap/user/UploadFile":
    handler.UploadFile(uploadDir)(w, r)
case "http://example.com/soap/user/UploadFileMTOM":
    handler.UploadFileMTOM(uploadDir)(w, r)
}
```

### MTOM XOP Resolution (`server/handler/file_mtom.go`)

```go
// Line 248-256: Extract Content-ID from XOP Include
re := regexp.MustCompile(`href=["']cid:([^"']+)["']`)
matches := re.FindStringSubmatch(fileDataElement)
if len(matches) > 1 {
    xopRefs = append(xopRefs, matches[1])
}

// Line 202-214: Resolve XOP to binary data
for _, xopRef := range xopRefs {
    for _, part := range parts {
        if part.ContentID == xopRef {
            fileData = part.Data
            break
        }
    }
}
```

### Spring MTOM Configuration (`client/.../SoapMtomConfig.java`)

```java
// Line 36-42: Enable MTOM on marshaller
@Bean
public Jaxb2Marshaller mtomMarshaller() {
    Jaxb2Marshaller marshaller = new Jaxb2Marshaller();
    marshaller.setPackagesToScan("com.example.soap.model");
    marshaller.setMtomEnabled(true);  // ← Enables MTOM
    marshaller.setSupportDtd(false);  // Security
    return marshaller;
}
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2024-02-01 | Initial documentation with all flow charts |

---

*This documentation was generated as part of the SOAP Test System project. For the latest source code, refer to the project repository.*
