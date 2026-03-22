# Wemake - Factory & Entrepreneur Connection Platform

Wemake เป็นแอปพลิเคชันกลางในการเชื่อมต่อระหว่างโรงงานกับนักธุรกิจรุ่นใหม่ที่สนใจสร้างธุรกิจของตัวเอง ทำให้การติดต่อเชื่อมโยงกันเป็นไปได้อย่างราบรื่นและมีประสิทธิภาพ

## 🎯 ข้อมูลโครงการ

```
wemake/
├── cmd/app/              # Entry point ของแอปพลิเคชัน
│   └── main.go           # Main function
├── internal/
│   ├── config/           # Configuration management
│   ├── domain/           # Domain models (entities)
│   ├── handler/          # HTTP request handlers
│   ├── repository/       # Database access layer
│   ├── service/          # Business logic layer
│   └── middleware/       # HTTP middleware
├── api/                  # API routes definition
├── migration/            # Database migrations
├── go.mod & go.sum       # Go dependencies
├── .env.example          # Environment variables template
├── Makefile              # Build commands
├── Dockerfile            # Container configuration
├── docker-compose.yml    # Multi-container setup
└── README.md            # This file
```

## 📋 Architecture Pattern

โปรเจคนี้ใช้ **Clean Architecture** ซึ่งเป็นรูปแบบที่ได้รับความนิยมในชุมชน Go:

1. **Handler Layer** - รับ request จาก HTTP client
2. **Service Layer** - ประมวลผลและตรวจสอบ business logic
3. **Repository Layer** - จัดการการเข้าถึงฐานข้อมูล
4. **Domain Layer** - กำหนด core models และ entities

## 🚀 วิธีการรัน

### สาหรับพัฒนา (Development)

#### 1. ติดตั้ง Go (หากยังไม่มี)
```bash
# บน macOS ใช้ Homebrew
brew install go
```

#### 2. Clone หรือสร้างโปรเจค
```bash
cd /Users/poon/Desktop/wemake
```

#### 3. ดาวน์โหลด Dependencies
```bash
go mod download
go mod tidy
```

#### 4. ตั้งค่า Environment Variables
```bash
cp .env.example .env
```

#### 5. เริ่ม Database ด้วย Docker
```bash
docker-compose up -d
```

#### 6. รันแอปพลิเคชัน
```bash
# วิธีที่ 1: ใช้ Makefile
make dev

# วิธีที่ 2: ใช้ go run โดยตรง
go run ./cmd/app/main.go
```

### เพื่อสร้าง Build Production

```bash
# Build binary
make build

# Run binary
./bin/wemake
```

## 📡 API Endpoints

### Health Check
```bash
GET /health
```

### Factory (โรงงาน) Routes
```bash
# สร้างโรงงานใหม่
POST /api/v1/factories
Content-Type: application/json
{
  "name": "Factory Name",
  "email": "factory@example.com",
  "phone": "0881234567",
  "address": "123 Factory Road",
  "description": "Factory description"
}

# ดึงข้อมูลโรงงานทั้งหมด
GET /api/v1/factories

# ดึงข้อมูลโรงงานตามรหัส
GET /api/v1/factories/{id}

# อัปเดตข้อมูลโรงงาน
PUT /api/v1/factories/{id}

# ลบโรงงาน
DELETE /api/v1/factories/{id}
```

### Auth Routes (Customer/Factory)
```bash
# Register (role: CT = customer, FT = factory)
POST /api/v1/auth/register
Content-Type: application/json
{
  "role": "CT",
  "email": "customer01@example.com",
  "phone": "0812345678",
  "password": "P@ssw0rd123",
  "first_name": "สมชาย",
  "last_name": "ใจดี"
}

# Register สำหรับโรงงาน
POST /api/v1/auth/register
Content-Type: application/json
{
  "role": "FT",
  "email": "factory01@example.com",
  "phone": "0899999999",
  "password": "P@ssw0rd123",
  "factory_name": "สมชาย แพคเกจจิ้ง",
  "factory_type_id": 1,
  "tax_id": "0105555xxxxxx"
}

# Login
POST /api/v1/auth/login
Content-Type: application/json
{
  "email": "customer01@example.com",
  "password": "P@ssw0rd123"
}

# Forgot Password (response includes reset_token for Postman testing)
POST /api/v1/auth/forgot-password
Content-Type: application/json
{
  "email": "customer01@example.com"
}

# Reset Password
POST /api/v1/auth/reset-password
Content-Type: application/json
{
  "token": "reset-token-from-forgot-password",
  "new_password": "N3wP@ssword123"
}
```

## 🗄️ Database Schema

### factories table
```sql
- id (UUID) - Primary Key
- name (VARCHAR) - ชื่อโรงงาน
- email (VARCHAR) - อีเมล
- phone (VARCHAR) - เบอร์โทรศัพท์
- address (TEXT) - ที่อยู่
- description (TEXT) - รายละเอียด
- created_at (TIMESTAMP)
- updated_at (TIMESTAMP)
```

### entrepreneurs table
```sql
- id (UUID) - Primary Key
- name (VARCHAR) - ชื่อนักธุรกิจ
- email (VARCHAR) - อีเมล
- phone (VARCHAR) - เบอร์โทรศัพท์
- company (VARCHAR) - ชื่อบริษัท
- created_at (TIMESTAMP)
- updated_at (TIMESTAMP)
```

### connections table
```sql
- id (UUID) - Primary Key
- factory_id (UUID) - Foreign Key to factories
- entrepreneur_id (UUID) - Foreign Key to entrepreneurs
- status (VARCHAR) - pending, approved, rejected
- message (TEXT) - เนื้อหาข้อความ
- created_at (TIMESTAMP)
- updated_at (TIMESTAMP)
```

## 🛠️ Commands ที่เป็นประโยชน์

```bash
# ดูคำสั่งทั้งหมด
make help

# ดาวน์โหลด dependencies
make deps

# รันแอปพลิเคชัน
make run

# รันทดสอบ
make test

# สร้าง Docker images
docker-compose up -d

# หยุด Docker containers
docker-compose down

# Clean build artifacts
make clean
```

## 📚 ไลบรารีหลักที่ใช้

- **Fiber** - Web framework เร็ว และเบา (เหมือน Express.js ของ Go)
- **sqlx** - Database library ที่มีประสิทธิภาพ
- **PostgreSQL** - ฐานข้อมูล
- **Docker** - Containerization

## 🔗 Dependencies

```
github.com/gofiber/fiber/v2      - Web framework
github.com/joho/godotenv         - Environment variables loader
github.com/jmoiron/sqlx          - Database library
github.com/lib/pq               - PostgreSQL driver
github.com/google/uuid           - UUID generation
```

## 🐳 Docker Setup

### เริ่มต้น Services
```bash
docker-compose up -d
```

### เข้าถึง Database
- **PostgreSQL**: localhost:5432
- **PgAdmin**: http://localhost:5050
  - Email: admin@example.com
  - Password: admin

### หยุด Services
```bash
docker-compose down
```

## 📝 ขั้นตอนการพัฒนาต่อไป

1. **Authentication** - เพิ่ม JWT authentication
2. **Validation** - เพิ่ม input validation middleware
3. **Logging** - เพิ่ม structured logging
4. **Tests** - เพิ่ม unit tests และ integration tests
5. **API Documentation** - เพิ่ม Swagger/OpenAPI
6. **Error Handling** - ปรับปรุง error handling
7. **Rate Limiting** - เพิ่ม rate limiting
8. **Caching** - เพิ่ม Redis caching

## 📞 หมายเหตุ

- อย่าลืมอัปเดต `go.mod` module path จากดีฟอลต์
- เปลี่ยน JWT_SECRET หากใช้ production
- ตรวจสอบให้แน่ใจว่า PostgreSQL เพอร์ต 5432 ว่างอยู่

## 📄 License

MIT License

---

สำหรับคำถามหรือข้อเสนอแนะ สามารถติดต่อได้ยินดี!
