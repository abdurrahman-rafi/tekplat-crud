package web

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/mail"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tekplat-crud/internal/config"
	"tekplat-crud/internal/store"
)

type App struct {
	config       config.Config
	store        *store.UserStore
	tableStore   *store.TableStore
	sessionToken string
}

type TemplateData struct {
	PageTitle   string
	Success     string
	Error       string
	Form        UserFormData
	Users       []store.User
	IsLoggedIn  bool
	Tables      []string
	TableName   string
	Columns     []string      // for import form column hint
	ColumnNames []string      // for row list / row form column headers
	Rows        [][]string    // for row list
	RowValues   []string      // for row edit form (parallel to ColumnNames)
	RowID       int           // for row edit form action URL
	AllTables   []string      // for FK dropdowns and create-table FK selector
	DiagramSrc  string        // Mermaid erDiagram source
}

type UserFormData struct {
	Mode   string
	Action string
	User   store.User
	Errors map[string]string
}

func NewApp(cfg config.Config, userStore *store.UserStore, tableStore *store.TableStore) *App {
	token, err := generateSessionToken()
	if err != nil {
		log.Fatal("gagal generate session token:", err)
	}
	return &App{
		config:       cfg,
		store:        userStore,
		tableStore:   tableStore,
		sessionToken: token,
	}
}

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/healthz", a.handleHealthz)
	mux.HandleFunc("/login", a.handleLogin)
	mux.HandleFunc("/logout", a.handleLogout)

	mux.Handle("/", a.requireAuth(http.HandlerFunc(a.handleHome)))
	mux.Handle("/users", a.requireAuth(http.HandlerFunc(a.handleUsers)))
	mux.Handle("/users/new", a.requireAuth(http.HandlerFunc(a.handleNewUserForm)))
	mux.Handle("/users/", a.requireAuth(http.HandlerFunc(a.handleUserActions)))
	mux.Handle("/tables", a.requireAuth(http.HandlerFunc(a.handleTables)))
	mux.Handle("/tables/new", a.requireAuth(http.HandlerFunc(a.handleNewTableForm)))
	mux.Handle("/tables/", a.requireAuth(http.HandlerFunc(a.handleTableActions)))
	mux.Handle("/diagram", a.requireAuth(http.HandlerFunc(a.handleDiagram)))

	return a.logRequest(mux)
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func (a *App) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil || cookie.Value != a.sessionToken {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if cookie, err := r.Cookie("session"); err == nil && cookie.Value == a.sessionToken {
			http.Redirect(w, r, "/users", http.StatusSeeOther)
			return
		}
		a.render(w, http.StatusOK, "login.gohtml", TemplateData{PageTitle: "Login"})
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			a.badRequest(w)
			return
		}
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")
		if username != a.config.AuthUser || password != a.config.AuthPassword {
			a.render(w, http.StatusUnauthorized, "login.gohtml", TemplateData{
				PageTitle: "Login",
				Error:     "Username atau password salah.",
			})
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    a.sessionToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, "/users", http.StatusSeeOther)
	default:
		a.methodNotAllowed(w)
	}
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.methodNotAllowed(w)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "session", Value: "", Path: "/", HttpOnly: true, MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ── Core ──────────────────────────────────────────────────────────────────────

func (a *App) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		a.notFound(w)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (a *App) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.methodNotAllowed(w)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// ── Users ─────────────────────────────────────────────────────────────────────

func (a *App) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ctx, cancel := a.requestContext(r)
		defer cancel()
		users, err := a.store.List(ctx)
		if err != nil {
			a.serverError(w, err)
			return
		}
		a.render(w, http.StatusOK, "list.gohtml", TemplateData{
			PageTitle:  "Daftar Users",
			Success:    userStatusMessage(r.URL.Query().Get("status")),
			Users:      users,
			IsLoggedIn: true,
		})
	case http.MethodPost:
		a.handleCreateUser(w, r)
	default:
		a.methodNotAllowed(w)
	}
}

func (a *App) handleNewUserForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.methodNotAllowed(w)
		return
	}
	a.render(w, http.StatusOK, "form.gohtml", TemplateData{
		PageTitle: "Tambah User",
		Form:      UserFormData{Mode: "create", Action: "/users", Errors: map[string]string{}},
		IsLoggedIn: true,
	})
}

func (a *App) handleUserActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 {
		a.notFound(w)
		return
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil || id <= 0 {
		a.notFound(w)
		return
	}
	switch parts[1] {
	case "edit":
		if r.Method != http.MethodGet {
			a.methodNotAllowed(w)
			return
		}
		a.handleEditUserForm(w, r, id)
	case "update":
		if r.Method != http.MethodPost {
			a.methodNotAllowed(w)
			return
		}
		a.handleUpdateUser(w, r, id)
	case "delete":
		if r.Method != http.MethodPost {
			a.methodNotAllowed(w)
			return
		}
		a.handleDeleteUser(w, r, id)
	default:
		a.notFound(w)
	}
}

func (a *App) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.badRequest(w)
		return
	}
	nama := strings.TrimSpace(r.FormValue("nama"))
	email := strings.TrimSpace(r.FormValue("email"))
	errs := validateUserInput(nama, email)
	if len(errs) > 0 {
		a.render(w, http.StatusUnprocessableEntity, "form.gohtml", TemplateData{
			PageTitle:  "Tambah User",
			Error:      "Form belum valid. Periksa input lalu simpan lagi.",
			Form:       UserFormData{Mode: "create", Action: "/users", User: store.User{Nama: nama, Email: email}, Errors: errs},
			IsLoggedIn: true,
		})
		return
	}
	ctx, cancel := a.requestContext(r)
	defer cancel()
	if err := a.store.Create(ctx, nama, email); err != nil {
		a.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/users?status=created", http.StatusSeeOther)
}

func (a *App) handleEditUserForm(w http.ResponseWriter, r *http.Request, id int) {
	ctx, cancel := a.requestContext(r)
	defer cancel()
	user, err := a.store.GetByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			a.notFound(w)
			return
		}
		a.serverError(w, err)
		return
	}
	a.render(w, http.StatusOK, "form.gohtml", TemplateData{
		PageTitle:  "Edit User",
		Form:       UserFormData{Mode: "edit", Action: "/users/" + strconv.Itoa(id) + "/update", User: user, Errors: map[string]string{}},
		IsLoggedIn: true,
	})
}

func (a *App) handleUpdateUser(w http.ResponseWriter, r *http.Request, id int) {
	if err := r.ParseForm(); err != nil {
		a.badRequest(w)
		return
	}
	nama := strings.TrimSpace(r.FormValue("nama"))
	email := strings.TrimSpace(r.FormValue("email"))
	errs := validateUserInput(nama, email)
	if len(errs) > 0 {
		a.render(w, http.StatusUnprocessableEntity, "form.gohtml", TemplateData{
			PageTitle:  "Edit User",
			Error:      "Form belum valid. Periksa input lalu simpan lagi.",
			Form:       UserFormData{Mode: "edit", Action: "/users/" + strconv.Itoa(id) + "/update", User: store.User{ID: id, Nama: nama, Email: email}, Errors: errs},
			IsLoggedIn: true,
		})
		return
	}
	ctx, cancel := a.requestContext(r)
	defer cancel()
	if err := a.store.Update(ctx, id, nama, email); err != nil {
		if err == sql.ErrNoRows {
			a.notFound(w)
			return
		}
		a.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/users?status=updated", http.StatusSeeOther)
}

func (a *App) handleDeleteUser(w http.ResponseWriter, r *http.Request, id int) {
	ctx, cancel := a.requestContext(r)
	defer cancel()
	if err := a.store.Delete(ctx, id); err != nil {
		if err == sql.ErrNoRows {
			a.notFound(w)
			return
		}
		a.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/users?status=deleted", http.StatusSeeOther)
}

// ── Tables (DDL + structure) ──────────────────────────────────────────────────

func (a *App) handleTables(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ctx, cancel := a.requestContext(r)
		defer cancel()
		tables, err := a.tableStore.ListTables(ctx)
		if err != nil {
			a.serverError(w, err)
			return
		}
		a.render(w, http.StatusOK, "tables_list.gohtml", TemplateData{
			PageTitle:  "Daftar Tabel",
			Success:    tableStatusMessage(r.URL.Query().Get("status"), r.URL.Query().Get("rows")),
			Tables:     tables,
			IsLoggedIn: true,
		})
	case http.MethodPost:
		a.handleCreateTable(w, r)
	default:
		a.methodNotAllowed(w)
	}
}

func (a *App) handleNewTableForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.methodNotAllowed(w)
		return
	}
	ctx, cancel := a.requestContext(r)
	defer cancel()
	tables, _ := a.tableStore.ListTables(ctx)
	a.render(w, http.StatusOK, "table_form.gohtml", TemplateData{
		PageTitle:  "Buat Tabel Baru",
		AllTables:  tables,
		IsLoggedIn: true,
	})
}

// handleTableActions routes everything under /tables/{name}/...
func (a *App) handleTableActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/tables/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		a.notFound(w)
		return
	}

	tableName := parts[0]
	action := parts[1]

	switch action {
	case "delete":
		if r.Method != http.MethodPost {
			a.methodNotAllowed(w)
			return
		}
		a.handleDropTable(w, r, tableName)

	case "import":
		switch r.Method {
		case http.MethodGet:
			a.handleImportCSVForm(w, r, tableName)
		case http.MethodPost:
			a.handleImportCSV(w, r, tableName)
		default:
			a.methodNotAllowed(w)
		}

	case "rows":
		// /tables/{name}/rows[/{id}/{action}]
		switch {
		case len(parts) == 2 && r.Method == http.MethodGet:
			a.handleListRows(w, r, tableName)
		case len(parts) == 2 && r.Method == http.MethodPost:
			a.handleInsertRow(w, r, tableName)
		case len(parts) == 3 && parts[2] == "new" && r.Method == http.MethodGet:
			a.handleNewRowForm(w, r, tableName)
		case len(parts) == 4:
			rowID, err := strconv.Atoi(parts[2])
			if err != nil || rowID <= 0 {
				a.notFound(w)
				return
			}
			switch parts[3] {
			case "edit":
				if r.Method != http.MethodGet {
					a.methodNotAllowed(w)
					return
				}
				a.handleEditRowForm(w, r, tableName, rowID)
			case "update":
				if r.Method != http.MethodPost {
					a.methodNotAllowed(w)
					return
				}
				a.handleUpdateRow(w, r, tableName, rowID)
			case "delete":
				if r.Method != http.MethodPost {
					a.methodNotAllowed(w)
					return
				}
				a.handleDeleteRow(w, r, tableName, rowID)
			default:
				a.notFound(w)
			}
		default:
			a.notFound(w)
		}

	case "fk":
		switch {
		case len(parts) == 3 && parts[2] == "new" && r.Method == http.MethodGet:
			a.handleFKForm(w, r, tableName)
		case len(parts) == 2 && r.Method == http.MethodPost:
			a.handleAddFK(w, r, tableName)
		default:
			a.notFound(w)
		}

	default:
		a.notFound(w)
	}
}

func (a *App) handleCreateTable(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.badRequest(w)
		return
	}

	tableName := strings.TrimSpace(r.FormValue("table_name"))
	names := r.Form["col_name[]"]
	types := r.Form["col_type[]"]
	refs := r.Form["col_ref[]"]

	var columns []store.Column
	for i, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		colType := "VARCHAR(255)"
		if i < len(types) && types[i] != "" {
			colType = types[i]
		}
		ref := ""
		if i < len(refs) {
			ref = refs[i]
		}
		columns = append(columns, store.Column{Name: n, Type: colType, References: ref})
	}

	ctx, cancel := a.requestContext(r)
	defer cancel()

	renderErr := func(msg string) {
		tables, _ := a.tableStore.ListTables(ctx)
		a.render(w, http.StatusUnprocessableEntity, "table_form.gohtml", TemplateData{
			PageTitle:  "Buat Tabel Baru",
			Error:      msg,
			AllTables:  tables,
			IsLoggedIn: true,
		})
	}

	if tableName == "" {
		renderErr("Nama tabel wajib diisi.")
		return
	}
	if len(columns) == 0 {
		renderErr("Minimal satu kolom diperlukan.")
		return
	}
	if err := a.tableStore.CreateTable(ctx, tableName, columns); err != nil {
		renderErr("Gagal membuat tabel: " + err.Error())
		return
	}

	http.Redirect(w, r, "/tables?status=created", http.StatusSeeOther)
}

func (a *App) handleDropTable(w http.ResponseWriter, r *http.Request, tableName string) {
	ctx, cancel := a.requestContext(r)
	defer cancel()
	if err := a.tableStore.DropTable(ctx, tableName); err != nil {
		a.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/tables?status=deleted", http.StatusSeeOther)
}

func (a *App) handleImportCSVForm(w http.ResponseWriter, r *http.Request, tableName string) {
	ctx, cancel := a.requestContext(r)
	defer cancel()
	columns, err := a.tableStore.ListColumns(ctx, tableName)
	if err != nil {
		a.serverError(w, err)
		return
	}
	a.render(w, http.StatusOK, "table_import.gohtml", TemplateData{
		PageTitle:  "Import CSV → " + tableName,
		TableName:  tableName,
		Columns:    columns,
		IsLoggedIn: true,
	})
}

func (a *App) handleImportCSV(w http.ResponseWriter, r *http.Request, tableName string) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		a.badRequest(w)
		return
	}
	file, _, err := r.FormFile("csv_file")
	if err != nil {
		a.badRequest(w)
		return
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		a.render(w, http.StatusUnprocessableEntity, "table_import.gohtml", TemplateData{
			PageTitle: "Import CSV → " + tableName, Error: "Gagal membaca file CSV: " + err.Error(),
			TableName: tableName, IsLoggedIn: true,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	count, err := a.tableStore.ImportCSV(ctx, tableName, records)
	if err != nil {
		a.render(w, http.StatusUnprocessableEntity, "table_import.gohtml", TemplateData{
			PageTitle: "Import CSV → " + tableName, Error: "Gagal import: " + err.Error(),
			TableName: tableName, IsLoggedIn: true,
		})
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/tables?status=imported&rows=%d", count), http.StatusSeeOther)
}

// ── Row CRUD ──────────────────────────────────────────────────────────────────

func (a *App) handleListRows(w http.ResponseWriter, r *http.Request, tableName string) {
	ctx, cancel := a.requestContext(r)
	defer cancel()
	cols, rows, err := a.tableStore.ListRows(ctx, tableName)
	if err != nil {
		a.serverError(w, err)
		return
	}
	a.render(w, http.StatusOK, "table_rows.gohtml", TemplateData{
		PageTitle:   "Baris: " + tableName,
		TableName:   tableName,
		ColumnNames: cols,
		Rows:        rows,
		Success:     rowStatusMessage(r.URL.Query().Get("status")),
		IsLoggedIn:  true,
	})
}

func (a *App) handleNewRowForm(w http.ResponseWriter, r *http.Request, tableName string) {
	ctx, cancel := a.requestContext(r)
	defer cancel()
	cols, err := a.tableStore.ListColumns(ctx, tableName)
	if err != nil {
		a.serverError(w, err)
		return
	}
	// exclude id — auto-generated
	editCols := filterID(cols)
	a.render(w, http.StatusOK, "row_form.gohtml", TemplateData{
		PageTitle:   "Tambah Baris: " + tableName,
		TableName:   tableName,
		ColumnNames: editCols,
		IsLoggedIn:  true,
	})
}

func (a *App) handleInsertRow(w http.ResponseWriter, r *http.Request, tableName string) {
	if err := r.ParseForm(); err != nil {
		a.badRequest(w)
		return
	}
	ctx, cancel := a.requestContext(r)
	defer cancel()

	cols, err := a.tableStore.ListColumns(ctx, tableName)
	if err != nil {
		a.serverError(w, err)
		return
	}
	editCols := filterID(cols)

	vals := make([]string, len(editCols))
	for i, c := range editCols {
		vals[i] = r.FormValue(c)
	}
	if err := a.tableStore.InsertRow(ctx, tableName, editCols, vals); err != nil {
		a.render(w, http.StatusUnprocessableEntity, "row_form.gohtml", TemplateData{
			PageTitle:   "Tambah Baris: " + tableName,
			TableName:   tableName,
			ColumnNames: editCols,
			RowValues:   vals,
			Error:       "Gagal menyimpan: " + err.Error(),
			IsLoggedIn:  true,
		})
		return
	}
	http.Redirect(w, r, "/tables/"+tableName+"/rows?status=created", http.StatusSeeOther)
}

func (a *App) handleEditRowForm(w http.ResponseWriter, r *http.Request, tableName string, id int) {
	ctx, cancel := a.requestContext(r)
	defer cancel()
	cols, vals, err := a.tableStore.GetRowByID(ctx, tableName, id)
	if err != nil {
		if err == sql.ErrNoRows {
			a.notFound(w)
			return
		}
		a.serverError(w, err)
		return
	}
	editCols, editVals := filterIDWithVals(cols, vals)
	a.render(w, http.StatusOK, "row_form.gohtml", TemplateData{
		PageTitle:   "Edit Baris: " + tableName,
		TableName:   tableName,
		ColumnNames: editCols,
		RowValues:   editVals,
		RowID:       id,
		IsLoggedIn:  true,
	})
}

func (a *App) handleUpdateRow(w http.ResponseWriter, r *http.Request, tableName string, id int) {
	if err := r.ParseForm(); err != nil {
		a.badRequest(w)
		return
	}
	ctx, cancel := a.requestContext(r)
	defer cancel()

	cols, err := a.tableStore.ListColumns(ctx, tableName)
	if err != nil {
		a.serverError(w, err)
		return
	}
	editCols := filterID(cols)
	vals := make([]string, len(editCols))
	for i, c := range editCols {
		vals[i] = r.FormValue(c)
	}
	if err := a.tableStore.UpdateRow(ctx, tableName, id, editCols, vals); err != nil {
		a.render(w, http.StatusUnprocessableEntity, "row_form.gohtml", TemplateData{
			PageTitle:   "Edit Baris: " + tableName,
			TableName:   tableName,
			ColumnNames: editCols,
			RowValues:   vals,
			RowID:       id,
			Error:       "Gagal update: " + err.Error(),
			IsLoggedIn:  true,
		})
		return
	}
	http.Redirect(w, r, "/tables/"+tableName+"/rows?status=updated", http.StatusSeeOther)
}

func (a *App) handleDeleteRow(w http.ResponseWriter, r *http.Request, tableName string, id int) {
	ctx, cancel := a.requestContext(r)
	defer cancel()
	if err := a.tableStore.DeleteRow(ctx, tableName, id); err != nil {
		a.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/tables/"+tableName+"/rows?status=deleted", http.StatusSeeOther)
}

// ── Foreign Keys ──────────────────────────────────────────────────────────────

func (a *App) handleFKForm(w http.ResponseWriter, r *http.Request, tableName string) {
	ctx, cancel := a.requestContext(r)
	defer cancel()
	cols, err := a.tableStore.ListColumns(ctx, tableName)
	if err != nil {
		a.serverError(w, err)
		return
	}
	tables, err := a.tableStore.ListTables(ctx)
	if err != nil {
		a.serverError(w, err)
		return
	}
	a.render(w, http.StatusOK, "table_fk.gohtml", TemplateData{
		PageTitle:   "Tambah Foreign Key: " + tableName,
		TableName:   tableName,
		ColumnNames: filterID(cols),
		AllTables:   tables,
		IsLoggedIn:  true,
	})
}

func (a *App) handleAddFK(w http.ResponseWriter, r *http.Request, tableName string) {
	if err := r.ParseForm(); err != nil {
		a.badRequest(w)
		return
	}
	col := strings.TrimSpace(r.FormValue("column"))
	refTable := strings.TrimSpace(r.FormValue("ref_table"))
	refCol := strings.TrimSpace(r.FormValue("ref_col"))
	if col == "" || refTable == "" || refCol == "" {
		a.badRequest(w)
		return
	}
	ctx, cancel := a.requestContext(r)
	defer cancel()
	if err := a.tableStore.AddForeignKey(ctx, tableName, col, refTable, refCol); err != nil {
		a.render(w, http.StatusUnprocessableEntity, "table_fk.gohtml", TemplateData{
			PageTitle: "Tambah Foreign Key: " + tableName,
			TableName: tableName,
			Error:     "Gagal menambah FK: " + err.Error(),
			IsLoggedIn: true,
		})
		return
	}
	http.Redirect(w, r, "/tables?status=fk_added", http.StatusSeeOther)
}

// ── ER Diagram ────────────────────────────────────────────────────────────────

func (a *App) handleDiagram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.methodNotAllowed(w)
		return
	}
	ctx, cancel := a.requestContext(r)
	defer cancel()

	schemas, err := a.tableStore.GetSchema(ctx)
	if err != nil {
		a.serverError(w, err)
		return
	}

	a.render(w, http.StatusOK, "diagram.gohtml", TemplateData{
		PageTitle:  "ER Diagram",
		DiagramSrc: buildMermaidDiagram(schemas),
		IsLoggedIn: true,
	})
}

func buildMermaidDiagram(schemas []store.TableSchema) string {
	var sb strings.Builder
	sb.WriteString("erDiagram\n")
	for _, t := range schemas {
		sb.WriteString("  " + t.Name + " {\n")
		for _, c := range t.Columns {
			key := ""
			switch c.Key {
			case "PRI":
				key = " PK"
			case "MUL":
				key = " FK"
			}
			sb.WriteString(fmt.Sprintf("    %s %s%s\n", c.DataType, c.Name, key))
		}
		sb.WriteString("  }\n")
	}
	for _, t := range schemas {
		for _, fk := range t.FKs {
			sb.WriteString(fmt.Sprintf("  %s ||--o{ %s : \"%s\"\n",
				fk.RefTable, t.Name, fk.Column))
		}
	}
	return sb.String()
}

// ── Rendering & helpers ───────────────────────────────────────────────────────

func (a *App) render(w http.ResponseWriter, status int, page string, data TemplateData) {
	files := []string{
		filepath.Join("web", "templates", "layout.gohtml"),
		filepath.Join("web", "templates", page),
	}
	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		a.serverError(w, err)
		return
	}
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("gagal render template %s: %v", page, err)
	}
}

func (a *App) notFound(w http.ResponseWriter) {
	http.Error(w, "halaman tidak ditemukan", http.StatusNotFound)
}

func (a *App) badRequest(w http.ResponseWriter) {
	http.Error(w, "request tidak valid", http.StatusBadRequest)
}

func (a *App) methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method tidak diizinkan", http.StatusMethodNotAllowed)
}

func (a *App) serverError(w http.ResponseWriter, err error) {
	log.Printf("server error: %v", err)
	http.Error(w, "terjadi kesalahan pada server", http.StatusInternalServerError)
}

func (a *App) requestContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 5*time.Second)
}

func (a *App) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func validateUserInput(nama, email string) map[string]string {
	errs := make(map[string]string)
	if nama == "" {
		errs["nama"] = "Nama wajib diisi."
	}
	if email == "" {
		errs["email"] = "Email wajib diisi."
	} else if _, err := mail.ParseAddress(email); err != nil {
		errs["email"] = "Format email tidak valid."
	}
	return errs
}

func filterID(cols []string) []string {
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		if c != "id" {
			out = append(out, c)
		}
	}
	return out
}

func filterIDWithVals(cols, vals []string) ([]string, []string) {
	var oc, ov []string
	for i, c := range cols {
		if c != "id" {
			oc = append(oc, c)
			if i < len(vals) {
				ov = append(ov, vals[i])
			}
		}
	}
	return oc, ov
}

func userStatusMessage(status string) string {
	switch status {
	case "created":
		return "Data user berhasil ditambahkan."
	case "updated":
		return "Data user berhasil diperbarui."
	case "deleted":
		return "Data user berhasil dihapus."
	default:
		return ""
	}
}

func rowStatusMessage(status string) string {
	switch status {
	case "created":
		return "Baris berhasil ditambahkan."
	case "updated":
		return "Baris berhasil diperbarui."
	case "deleted":
		return "Baris berhasil dihapus."
	default:
		return ""
	}
}

func tableStatusMessage(status, rows string) string {
	switch status {
	case "created":
		return "Tabel berhasil dibuat."
	case "deleted":
		return "Tabel berhasil dihapus."
	case "imported":
		return fmt.Sprintf("Berhasil mengimpor %s baris data.", rows)
	case "fk_added":
		return "Foreign key berhasil ditambahkan."
	default:
		return ""
	}
}
