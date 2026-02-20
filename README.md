<div align="center">

# 🚀 Gowoobro Backend API

**gowoobro.com 포트폴리오 사이트의 백엔드 REST API 서버**

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![Fiber](https://img.shields.io/badge/Fiber-v2-00ACD7?style=for-the-badge&logo=go&logoColor=white)](https://gofiber.io/)
[![MySQL](https://img.shields.io/badge/MySQL-8.0-4479A1?style=for-the-badge&logo=mysql&logoColor=white)](https://www.mysql.com/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://www.docker.com/)

</div>

---

## 📖 소개

**Gowoobro Backend**는 [gowoobro.com](https://gowoobro.com) 포트폴리오 웹사이트를 위한 고성능 REST API 서버입니다.  
Go의 빠른 실행 속도와 [Fiber](https://gofiber.io/) 프레임워크의 경량 아키텍처를 기반으로, 프로젝트 관리 · 방문자 추적 · Q&A 시스템 · 파일 업로드 기능을 제공합니다.

---

## ✨ 주요 기능

| 기능 | 설명 |
|:---|:---|
| 🗂️ **프로젝트 관리** | 포트폴리오 프로젝트 CRUD (웹/앱 구분, 스토어 링크, QR코드 등) |
| 🛡️ **IP 추적** | 방문자 IP 기록 및 관리 |
| 💬 **Q&A** | 방문자 질문 수집 및 관리 |
| 📤 **파일 업로드** | 프로젝트 아이콘 등 파일 업로드 처리 |
| 🔒 **TLS 지원** | 프로덕션 환경에서 HTTPS 지원 |
| 🐳 **Docker 지원** | Multi-stage 빌드를 통한 경량 컨테이너 배포 |

---

## 🏗️ 프로젝트 구조

```
gowoobrogo/
├── main.go                 # 앱 엔트리포인트
├── services/
│   └── http.go             # Fiber HTTP 서버 설정 (CORS, TLS, 압축)
├── router/
│   ├── router.go           # 라우터 초기화
│   └── routers/            # 도메인별 라우트 정의
│       ├── ipblock.go
│       ├── projects.go
│       ├── questions.go
│       └── upload.go
├── controllers/
│   ├── controllers.go      # 공통 컨트롤러 로직
│   ├── api/                # API 전용 컨트롤러 (파일 업로드)
│   └── rest/               # RESTful CRUD 컨트롤러
├── models/                 # 데이터 모델 및 DB 접근 계층
│   ├── db.go               # 데이터베이스 연결
│   ├── ipblock.go
│   ├── projects.go
│   └── questions.go
├── global/                 # 글로벌 유틸리티
│   ├── config/             # 환경 설정
│   ├── log/                # Zerolog 기반 로깅
│   ├── setting/            # 런타임 설정
│   └── time/               # 시간 유틸리티
├── dockerfile              # Multi-stage Docker 빌드
├── docker-compose.yml      # Docker Compose 설정
└── Makefile                # 빌드 자동화
```

---

## 🔌 API 엔드포인트

모든 엔드포인트는 `/api` 접두사를 사용합니다.

### 🗂️ Projects — `/api/projects`

| Method | Endpoint | 설명 |
|:---:|:---|:---|
| `GET` | `/api/projects` | 프로젝트 목록 조회 (`?page=&pagesize=`) |
| `GET` | `/api/projects/:id` | 특정 프로젝트 조회 |
| `POST` | `/api/projects` | 프로젝트 생성 |
| `POST` | `/api/projects/batch` | 프로젝트 일괄 생성 |
| `POST` | `/api/projects/count` | 프로젝트 수 조회 |
| `PUT` | `/api/projects` | 프로젝트 수정 |
| `DELETE` | `/api/projects` | 프로젝트 삭제 |
| `DELETE` | `/api/projects/batch` | 프로젝트 일괄 삭제 |

### 🛡️ IP Block — `/api/ipblock`

| Method | Endpoint | 설명 |
|:---:|:---|:---|
| `GET` | `/api/ipblock` | IP 목록 조회 |
| `GET` | `/api/ipblock/:id` | 특정 IP 조회 |
| `POST` | `/api/ipblock` | IP 기록 추가 |
| `PUT` | `/api/ipblock` | IP 정보 수정 |
| `DELETE` | `/api/ipblock` | IP 기록 삭제 |

### 💬 Questions — `/api/questions`

| Method | Endpoint | 설명 |
|:---:|:---|:---|
| `GET` | `/api/questions` | 질문 목록 조회 |
| `GET` | `/api/questions/:id` | 특정 질문 조회 |
| `POST` | `/api/questions` | 질문 등록 |
| `PUT` | `/api/questions` | 질문 수정 |
| `DELETE` | `/api/questions` | 질문 삭제 |

### 📤 Upload — `/api/upload`

| Method | Endpoint | 설명 |
|:---:|:---|:---|
| `POST` | `/api/upload/index` | 파일 업로드 |

---

## 🚀 시작하기

### 사전 요구사항

- **Go** 1.25+
- **MySQL** 8.0+ / MariaDB
- **Docker** & **Docker Compose** *(선택사항)*

### 로컬 실행

```bash
# 1. 의존성 설치
go mod download

# 2. 환경 설정 파일 생성
cp .env.yml.example .env.yml
# .env.yml에 DB 접속 정보 입력

# 3. 데이터베이스 테이블 생성
mysql -u <user> -p <dbname> < gowoobro.sql

# 4. 서버 실행
make run
```

서버가 `http://localhost:8007`에서 시작됩니다.

### 바이너리 빌드

```bash
# macOS / Windows
make server

# Linux 크로스 컴파일
make linux
```

빌드 결과물은 `bin/` 디렉토리에 생성됩니다.

---

## 🐳 Docker

### Docker Compose로 실행

```bash
docker compose up -d
```

### 수동 Docker 빌드 & 실행

```bash
# 이미지 빌드
make docker

# 컨테이너 실행
make dockerrun

# Docker Hub에 푸시
make push tag=v1.0.0
```

---

## ⚙️ 환경 설정

`.env.yml` 파일에서 환경별 설정을 관리합니다.

```yaml
develop:
  database:
    type: mysql
    host: localhost
    port: 3306
    name: gowoobro
    user: your_user
    password: your_password
  port: 8007
  cors: [http://localhost:9007]
  documentRoot: ./webdata
  path: ./webdata
```

| 항목 | 설명 |
|:---|:---|
| `database` | MySQL/MariaDB 접속 정보 |
| `port` | 서버 리스닝 포트 |
| `cors` | 허용할 CORS 오리진 목록 |
| `documentRoot` | 정적 파일 서빙 루트 |
| `path` | 파일 업로드 저장 경로 |

---

## 🛠️ 기술 스택

<div align="center">

| 분류 | 기술 |
|:---:|:---|
| **언어** | Go 1.25 |
| **프레임워크** | Fiber v2 |
| **데이터베이스** | MySQL / MariaDB |
| **로깅** | Zerolog |
| **이미지 처리** | disintegration/imaging |
| **컨테이너** | Docker, Docker Compose |
| **보안** | TLS/HTTPS, CORS |

</div>

---

## 📝 Makefile 명령어

```bash
make server       # Go 바이너리 빌드
make run          # 개발 서버 실행
make test         # 테스트 실행
make linux        # Linux용 크로스 컴파일
make docker       # Docker 이미지 빌드
make dockerrun    # Docker 컨테이너 실행
make push         # Docker Hub에 푸시
make clean        # 빌드 결과물 삭제
```

---

<div align="center">

**Made with ❤️ by [gowoobro](https://gowoobro.com)**

</div>
