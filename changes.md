# Changelog

---

## 2026-04-22

### 10:00 — Added AGENTS.md
Comprehensive project documentation for AI coding agents.
- Covers project purpose, directory structure, all HTTP routes, environment variables, DB schema, code conventions, and common pitfalls.

**Files:**
- `AGENTS.md` — created

---

### 11:00 — Auth: Cookie-based login
Single-user session auth using credentials stored in `.env`. A random 32-byte session token is generated at server startup. Login sets a `session` cookie; logout clears it. All routes except `/login`, `/healthz`, and `/static/*` require a valid session.

**Files:**
- `internal/config/config.go` — added `AuthUser`, `AuthPassword` fields
- `.env.example` — added `AUTH_USER`, `AUTH_PASSWORD`
- `internal/web/app.go` — added `sessionToken` on `App`, `requireAuth` middleware, `handleLogin`, `handleLogout`
- `web/templates/layout.gohtml` — added nav bar with logout button (shown only when `IsLoggedIn`)
- `web/templates/login.gohtml` — new login form template
- `web/static/css/app.css` — added `.header-inner`, `.nav`, `.nav-link`, `.login-wrap`, `.login-card` styles

---

### 11:30 — DB Management UI: create table, drop table, import CSV
New `/tables` section for managing database tables directly from the browser.

**Features:**
- **List tables** — shows all tables in the connected database
- **Create table** — dynamic column builder (+ Add Column / ✕ remove via vanilla JS); auto-adds `id AUTO_INCREMENT`; column names/types validated server-side against a whitelist to prevent SQL injection
- **Drop table** — requires browser confirm dialog
- **Import CSV** — multipart file upload; first row = column headers; all rows inserted in a single transaction (rollback on any error); 10 MB file limit, 30 s timeout

**Supported column types:** `VARCHAR(255)`, `INT`, `TEXT`, `DECIMAL(10,2)`, `BOOLEAN`, `TIMESTAMP`

**Files:**
- `internal/store/table_store.go` — new: `TableStore` with `ListTables`, `CreateTable`, `DropTable`, `ListColumns`, `ImportCSV`
- `cmd/web/main.go` — wire up `TableStore`, pass to `web.NewApp`
- `internal/web/app.go` — new routes + handlers for `/tables`, `/tables/new`, `/tables/{name}/delete`, `/tables/{name}/import`
- `web/templates/tables_list.gohtml` — new: table list page
- `web/templates/table_form.gohtml` — new: dynamic create table form
- `web/templates/table_import.gohtml` — new: CSV import form
- `web/static/css/app.css` — added `.column-row`, `.form-hint` styles

---

---

### 14:00 — Generic row CRUD, ER Diagram, Foreign Keys, Extended Column Types

**Generic row CRUD for any table**
Browse, add, edit, and delete rows in any table — not just `users`. Dynamic column detection via `SHOW COLUMNS`. All values treated as strings (MySQL handles coercion). Row list capped at 200 rows. `id` column is excluded from insert/update forms (auto-generated).

**ER Diagram**
New `/diagram` page. Server queries `INFORMATION_SCHEMA.COLUMNS` and `KEY_COLUMN_USAGE` to build a Mermaid `erDiagram` definition, which the browser renders via Mermaid.js CDN (dark theme matching the app palette). PK/FK markers shown on columns.

**Foreign keys**
- *At create time:* each column row has an optional "FK (optional)" dropdown listing all existing tables → `id`. Generates `FOREIGN KEY` constraint in `CREATE TABLE`.
- *On existing tables:* new "Add FK" button per table in the list → form with column + reference table/column dropdowns → executes `ALTER TABLE ADD CONSTRAINT FOREIGN KEY`.

**Extended column types**
New dynamic type picker in the create-table form. Base type dropdown; for `VARCHAR`/`CHAR` a length input appears (default 255/10); for `DECIMAL` precision + scale inputs appear. Full type string (`VARCHAR(120)`, `DECIMAL(10,2)`) assembled client-side before submit. Server validates via regex instead of a fixed set.

Types: `VARCHAR(n)`, `CHAR(n)`, `INT`, `BIGINT`, `FLOAT`, `DECIMAL(p,s)`, `TEXT`, `DATE`, `DATETIME`, `TIMESTAMP`, `BOOLEAN`, `JSON`

**Navigation**
Header nav updated to: Users · Daftar Tabel · ER Diagram

**Files:**
- `internal/store/table_store.go` — added `ListRows`, `GetRowByID`, `InsertRow`, `UpdateRow`, `DeleteRow`, `AddForeignKey`, `GetSchema`; updated `Column` struct with `References` field; replaced fixed type map with regex-based `isValidColumnType`
- `internal/web/app.go` — all new handlers, updated `TemplateData`, `buildMermaidDiagram` helper, `filterID`/`filterIDWithVals` helpers
- `web/templates/layout.gohtml` — added ER Diagram nav link, renamed "Tables" to "Daftar Tabel"
- `web/templates/table_form.gohtml` — extended type picker with JS param inputs, FK reference dropdown
- `web/templates/tables_list.gohtml` — added "Baris" and "+ FK" buttons per table
- `web/templates/table_rows.gohtml` — new: generic row list
- `web/templates/row_form.gohtml` — new: generic add/edit row form
- `web/templates/table_fk.gohtml` — new: add FK to existing table
- `web/templates/diagram.gohtml` — new: Mermaid ER diagram page
