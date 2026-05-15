# T-Monitor: Master Refactoring & Unit Testing Plan

เอกสารฉบับนี้คือ **แผนแม่บท (Master Plan)** สำหรับการทำ Unit Testing และ Refactoring ระบบ Backend ของ T-Monitor ทั้งหมด โดยแบ่งย่อยตามเมนู/ฟีเจอร์หลักของระบบ เพื่อให้สามารถดำเนินการตรวจสอบและแก้ไขทีละส่วนได้อย่างปลอดภัย ป้องกันการเกิด Code Error กระทบเป็นวงกว้าง

---

## 🛠️ Phase 0: Testing Infrastructure Setup
ก่อนเริ่มปรับแก้โค้ดทางฝั่ง Business Logic เราจำเป็นต้องวางรากฐานสำหรับระบบ Test ก่อน
- [ ] ติดตั้ง `testify` สำหรับทำ Assertions
- [ ] สร้าง Mock Database (SQLite In-Memory หรือ Transaction Rollback) สำหรับรันเทสโดยไม่กระทบข้อมูลจริง
- [ ] สร้าง Test Utilities (เช่น การ Mock JWT Token, สร้าง Mock User/Project)

---

## 📊 Phase 1: Authentication & User Management (ระบบผู้ใช้งาน)
**ทำไมต้องเริ่มที่นี่?** เพราะระบบอื่นๆ ทั้งหมดต้องพึ่งพาข้อมูล User และสิทธิ์การเข้าถึง หากระบบนี้เสถียร จะทำให้การเทสระบบอื่นง่ายขึ้น

- **ไฟล์เป้าหมาย:** `auth_handlers.go`, `user_handlers.go`
- **Unit Tests ที่ต้องเขียน:**
  - `TestLogin`, `TestRegister`
  - `TestProfileUpdate` (รวมถึงการอัปโหลดรูปภาพ)
  - `TestRoleManagement` (สิทธิ์ Admin vs User)
- **เป้าหมายการ Refactor:** แยกโค้ดส่วนการสร้าง JWT Token และ Hashing รหัสผ่านออกจาก Handler ไปไว้ที่ `services/auth_service.go`

---

## 🏢 Phase 2: Company & Members (ระบบจัดการองค์กร)
- **ไฟล์เป้าหมาย:** `company_handlers.go`, `company_member_handlers.go`
- **Unit Tests ที่ต้องเขียน:**
  - `TestCreateCompany`, `TestDeleteCompany`
  - `TestInviteMember`, `TestAcceptInvitation`
- **เป้าหมายการ Refactor:** จัดระเบียบการเช็คสิทธิ์ (Authorization) ให้อ่านง่ายขึ้น และแยก Query ฐานข้อมูลที่ซับซ้อนออกไปที่ `repositories`

---

## 📁 Phase 3: Project Management (ระบบจัดการโปรเจกต์)
- **ไฟล์เป้าหมาย:** `project_handlers.go`
- **Unit Tests ที่ต้องเขียน:**
  - `TestCreateProject`, `TestProjectOwnership`
  - `TestProjectMemberAccess` (ทดสอบว่า Member มองเห็น Project แต่ User นอกโปรเจกต์มองไม่เห็น)
- **เป้าหมายการ Refactor:** ลดความซ้ำซ้อนของโค้ดในการตรวจสอบสิทธิ์ (Ownership Check) ที่กระจายอยู่ในหลายๆ ฟังก์ชัน

---

## ⚙️ Phase 4: API Endpoints (ระบบจัดการ API ที่ต้องการ Monitor)
- **ไฟล์เป้าหมาย:** `api_handlers.go`
- **Unit Tests ที่ต้องเขียน:**
  - `TestCreateAPI`, `TestUpdateAPI`
  - `TestPostmanImport` (จำลองการอัปโหลดไฟล์ JSON ของ Postman)
  - `TestPauseAPI` (ทดสอบการตั้งเวลา Pause ชั่วคราว)
- **เป้าหมายการ Refactor:** แยกโค้ดการ Parsing Postman Collection ที่ยาวมาก (หลายร้อยบรรทัด) ออกไปเป็น Utility Function ต่างหาก

---

## 🔔 Phase 5: Notifications Settings (ระบบตั้งค่าการแจ้งเตือน)
- **ไฟล์เป้าหมาย:** `notification_handlers.go`
- **Unit Tests ที่ต้องเขียน:**
  - `TestUpsertNotificationConfig`
  - `TestGetNotificationConfig`
- **เป้าหมายการ Refactor:** จัดระเบียบ JSON Binding และ Default Values ให้เป็นมาตรฐาน

---

## 📈 Phase 6: Analytics & AI Chat (ระบบวิเคราะห์ข้อมูลและ AI)
- **ไฟล์เป้าหมาย:** `analytics_handlers.go`, `ai_handlers.go`
- **Unit Tests ที่ต้องเขียน:**
  - `TestGetUptimeStats`, `TestGetLatencyTrend`
  - Mock การทำงานของ Ollama AI (ไม่ยิง API จริงตอนรันเทส)
- **เป้าหมายการ Refactor:** แยก Logic การดึงข้อมูล Analytics เชิงลึกออกจาก Handler ไปไว้ที่ `services/analytics_service.go`

---

## 🛠️ Phase 7: Repair Tasks & Incidents (ระบบจัดการปัญหา)
- **ไฟล์เป้าหมาย:** `repair_handlers.go`
- **Unit Tests ที่ต้องเขียน:**
  - `TestApproveRepairTask`, `TestCloseRepairTask`
- **เป้าหมายการ Refactor:** ปรับปรุง Flow การเปลี่ยนสถานะ (State Machine) ของ Repair Task ให้มี Logic ที่ชัดเจนขึ้น

---

## 🧠 Phase 8: Core Healthcheck Worker (หัวใจหลักของระบบ)
**ทำไมถึงเอาไว้ท้ายสุด?** เพราะเป็นระบบที่ซับซ้อนที่สุด ทำงานแบบ Background (Goroutines) และเชื่อมโยงกับทุกระบบ
- **ไฟล์เป้าหมาย:** `workers/healthcheck.go`
- **Unit Tests ที่ต้องเขียน:**
  - Mock HTTP Client เพื่อจำลองการ Ping API ที่สำเร็จ (200) และล้มเหลว (500)
  - ทดสอบ Logic การเกิด Incident ใหม่ vs Incident ซ้ำ (Auto-Repair Tasks)
  - ทดสอบ `isSafeURL` (SSRF Prevention)
- **เป้าหมายการ Refactor:** หั่นฟังก์ชัน `runPing` และ `handleResult` ที่ใหญ่มาก (หลายร้อยบรรทัด) ออกเป็นชิ้นเล็กๆ:
  - `RequestBuilder` (สร้าง HTTP Request)
  - `ResultEvaluator` (ประเมินว่า API ล่มหรือไม่)
  - `AlertDispatcher` (จัดการเรื่องแจ้งเตือน Telegram/Email)

---

### วิธีการดำเนินงาน (Workflow)
เราจะทำงานร่วมกันทีละ Phase โดยคุณสามารถสั่งว่า **"เริ่มทำ Phase 1"** ผมจะเริ่มดำเนินการสร้าง Test และ Refactor โค้ดในส่วนนั้น พร้อมรายงานผลให้คุณทราบก่อนขยับไป Phase ถัดไปครับ
