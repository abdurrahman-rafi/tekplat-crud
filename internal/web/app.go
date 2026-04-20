package web

import (
	"context"
	"database/sql"
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
	config config.Config
	store  *store.UserStore
}

type TemplateData struct {
	PageTitle string
	Success   string
	Error     string
	Form      UserFormData
	Users     []store.User
}

type UserFormData struct {
	Mode   string
	Action string
	User   store.User
	Errors map[string]string
}

func NewApp(cfg config.Config, userStore *store.UserStore) *App {
	return &App{
		config: cfg,
		store:  userStore,
	}
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("web/static"))

	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	mux.HandleFunc("/healthz", a.handleHealthz)
	mux.HandleFunc("/", a.handleHome)
	mux.HandleFunc("/users", a.handleUsers)
	mux.HandleFunc("/users/new", a.handleNewUserForm)
	mux.HandleFunc("/users/", a.handleUserActions)

	return a.logRequest(mux)
}

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

		data := TemplateData{
			PageTitle: "Daftar Users",
			Success:   statusMessage(r.URL.Query().Get("status")),
			Users:     users,
		}
		a.render(w, http.StatusOK, "list.gohtml", data)
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

	data := TemplateData{
		PageTitle: "Tambah User",
		Form: UserFormData{
			Mode:   "create",
			Action: "/users",
			Errors: map[string]string{},
		},
	}

	a.render(w, http.StatusOK, "form.gohtml", data)
}

func (a *App) handleUserActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
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
	errors := validateUserInput(nama, email)

	if len(errors) > 0 {
		data := TemplateData{
			PageTitle: "Tambah User",
			Error:     "Form belum valid. Periksa input lalu simpan lagi.",
			Form: UserFormData{
				Mode:   "create",
				Action: "/users",
				User: store.User{
					Nama:  nama,
					Email: email,
				},
				Errors: errors,
			},
		}
		a.render(w, http.StatusUnprocessableEntity, "form.gohtml", data)
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

	data := TemplateData{
		PageTitle: "Edit User",
		Form: UserFormData{
			Mode:   "edit",
			Action: "/users/" + strconv.Itoa(id) + "/update",
			User:   user,
			Errors: map[string]string{},
		},
	}

	a.render(w, http.StatusOK, "form.gohtml", data)
}

func (a *App) handleUpdateUser(w http.ResponseWriter, r *http.Request, id int) {
	if err := r.ParseForm(); err != nil {
		a.badRequest(w)
		return
	}

	nama := strings.TrimSpace(r.FormValue("nama"))
	email := strings.TrimSpace(r.FormValue("email"))
	errors := validateUserInput(nama, email)

	if len(errors) > 0 {
		data := TemplateData{
			PageTitle: "Edit User",
			Error:     "Form belum valid. Periksa input lalu simpan lagi.",
			Form: UserFormData{
				Mode:   "edit",
				Action: "/users/" + strconv.Itoa(id) + "/update",
				User: store.User{
					ID:    id,
					Nama:  nama,
					Email: email,
				},
				Errors: errors,
			},
		}
		a.render(w, http.StatusUnprocessableEntity, "form.gohtml", data)
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
	errors := make(map[string]string)

	if nama == "" {
		errors["nama"] = "Nama wajib diisi."
	}
	if email == "" {
		errors["email"] = "Email wajib diisi."
	} else if _, err := mail.ParseAddress(email); err != nil {
		errors["email"] = "Format email tidak valid."
	}

	return errors
}

func statusMessage(status string) string {
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
