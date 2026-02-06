# SOAP Server (Go)

Go로 구현된 SOAP 서버입니다. 사용자 조회와 파일 업로드(Base64, MTOM) 기능을 제공합니다.

## 기능

- **GetUser**: 사용자 ID로 정보 조회
- **UploadFile**: Base64 인코딩 파일 업로드
- **UploadFileMTOM**: MTOM 최적화 파일 업로드

## 실행

```bash
go run main.go
```

서버는 포트 8080에서 실행됩니다.

## 엔드포인트

| 경로 | 설명 |
|------|------|
| `/soap` | SOAP 엔드포인트 |
| `/wsdl` | WSDL 정의 |
| `/health` | 건강 상태 확인 |

## SOAPAction

- `http://example.com/soap/user/GetUser`
- `http://example.com/soap/user/UploadFile`
- `http://example.com/soap/user/UploadFileMTOM`

## 요구사항

- Go 1.21+

## 샘플 데이터

| ID | 이름 | 이메일 |
|----|------|--------|
| 1 | 홍길동 | hong@example.com |
| 2 | 김철수 | kim@example.com |
| 3 | 이영희 | lee@example.com |
