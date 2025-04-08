package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/nfnt/resize"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"` // "admin" or "viewer"
}

type Config struct {
	Title            string `json:"title"`
	ItemsPerPage     int    `json:"items_per_page"`
	PhotoThumbSize   int    `json:"photo_thumbnail_size"`
	PhotoPreviewSize int    `json:"photo_preview_size"`
	SessionTimeout   int    `json:"session_timeout"`
	MaxLoginAttempts int    `json:"max_login_attempts"`
	LockoutDuration  int    `json:"lockout_duration"`
}

type Item struct {
	ID            int    `json:"id"`
	Nome          string `json:"nome"`
	Descricao     string `json:"descricao"`
	Estante       string `json:"estante"`
	Prateleira    string `json:"prateleira"`
	Compartimento string `json:"compartimento"`
	Foto          string `json:"foto"`
}

type Estante struct {
	Nome string `json:"nome"`
}

type Inventario struct {
	Itens    []Item    `json:"itens"`
	Estantes []Estante `json:"estantes"`
}

type PaginationData struct {
	CurrentPage  int
	TotalPages   int
	ItemsPerPage int
	TotalItems   int
}

type Usuario struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Foto     string `json:"foto"`
}

type UsuariosData struct {
	Usuarios []Usuario `json:"usuarios"`
}

var (
	dados        Inventario
	config       Config
	store        *sessions.CookieStore
	users        []User
	usuariosData UsuariosData
)

func carregarConfig() {
	file, err := os.ReadFile("config.json")
	if err == nil {
		json.Unmarshal(file, &config)
	} else {
		config = Config{
			Title:            "Workshop Inventory",
			ItemsPerPage:     10,
			PhotoThumbSize:   100,
			PhotoPreviewSize: 600,
			SessionTimeout:   3600,
			MaxLoginAttempts: 5,
			LockoutDuration:  300,
		}
	}
	// Initialize session store with a fixed secret key
	store = sessions.NewCookieStore([]byte("your-secure-session-key-here"))
}

func carregarDados() {
	file, err := os.ReadFile("dados.json")
	if err == nil {
		json.Unmarshal(file, &dados)
	}
}

func salvarDados() {
	data, _ := json.MarshalIndent(dados, "", "  ")
	ioutil.WriteFile("dados.json", data, 0644)
}

func carregarUsuarios() {
	file, err := os.ReadFile("usuarios.json")
	if err == nil {
		json.Unmarshal(file, &usuariosData)
	} else {
		// Create default admin user if no users file exists
		usuariosData = UsuariosData{
			Usuarios: []Usuario{
				{
					ID:       1,
					Username: "admin",
					Password: "admin", // Default password, should be changed after first login
					Role:     "admin",
				},
			},
		}
		salvarUsuarios()
	}
}

func salvarUsuarios() {
	data, _ := json.MarshalIndent(usuariosData, "", "  ")
	os.WriteFile("usuarios.json", data, 0644)
}

func isAuthenticated(r *http.Request) bool {
	session, _ := store.Get(r, "session")
	auth, ok := session.Values["authenticated"].(bool)
	return ok && auth
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func getUserRole(r *http.Request) string {
	session, _ := store.Get(r, "session")
	role, _ := session.Values["role"].(string)
	return role
}

func requireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		userRole := getUserRole(r)
		if userRole != role {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl := template.Must(template.ParseFiles("templates/login.html"))
		tmpl.Execute(w, struct {
			Error  string
			Config Config
		}{
			Config: config,
		})
		return
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		for _, user := range usuariosData.Usuarios {
			if user.Username == username && user.Password == password {
				session, _ := store.Get(r, "session")
				session.Values["authenticated"] = true
				session.Values["username"] = username
				session.Values["role"] = user.Role
				session.Save(r, w)
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}

		tmpl := template.Must(template.ParseFiles("templates/login.html"))
		tmpl.Execute(w, struct {
			Error  string
			Config Config
		}{
			Error:  "Invalid username or password",
			Config: config,
		})
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Values["authenticated"] = false
	delete(session.Values, "username")
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func generateThumbnail(img image.Image) image.Image {
	return resize.Thumbnail(uint(config.PhotoThumbSize), uint(config.PhotoThumbSize), img, resize.Lanczos3)
}

func saveImage(file io.Reader, filename string) (string, error) {
	// Create directories if they don't exist
	os.MkdirAll("static/photos", 0755)
	os.MkdirAll("static/photos/thumbs", 0755)

	// Read the image
	img, format, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	// Generate thumbnail
	thumb := generateThumbnail(img)

	// Save original image
	originalPath := filepath.Join("static/photos", filename)
	f, err := os.Create(originalPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Save based on format
	switch format {
	case "jpeg", "jpg":
		jpeg.Encode(f, img, nil)
	case "png":
		png.Encode(f, img)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	// Save thumbnail
	thumbPath := filepath.Join("static/photos/thumbs", filename)
	f, err = os.Create(thumbPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	jpeg.Encode(f, thumb, nil)

	return filename, nil
}

func main() {
	carregarConfig()
	carregarDados()
	carregarUsuarios()

	// Create template functions
	funcMap := template.FuncMap{
		"add":      func(a, b int) int { return a + b },
		"subtract": func(a, b int) int { return a - b },
		"seq": func(start, end int) []int {
			var result []int
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
			return result
		},
		"js": func(s string) string {
			return template.JSEscapeString(s)
		},
	}

	// Parse templates with custom functions
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseFiles("templates/index.html", "templates/estantes.html", "templates/editar_item.html"))

	// Public routes
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Protected routes
	http.HandleFunc("/", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		listarItens(w, r, tmpl)
	}))
	http.HandleFunc("/novo", requireRole("admin", novoItem))
	http.HandleFunc("/editar", requireRole("admin", editarItem))
	http.HandleFunc("/deletar", requireRole("admin", deletarItem))
	http.HandleFunc("/estantes", requireRole("admin", listarEstantes))
	http.HandleFunc("/estantes/novo", requireRole("admin", novaEstante))
	http.HandleFunc("/estantes/editar", requireRole("admin", editarEstante))
	http.HandleFunc("/estantes/deletar", requireRole("admin", deletarEstante))

	// Add user management routes
	http.HandleFunc("/usuarios", listarUsuarios)
	http.HandleFunc("/usuarios/novo", novoUsuario)
	http.HandleFunc("/usuarios/editar", editarUsuario)
	http.HandleFunc("/usuarios/deletar", deletarUsuario)

	log.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}

func listarItens(w http.ResponseWriter, r *http.Request, tmpl *template.Template) {
	busca := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("q")))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	var itensFiltrados []Item
	if busca != "" {
		for _, item := range dados.Itens {
			if strings.Contains(strings.ToLower(item.Nome), busca) || strings.Contains(strings.ToLower(item.Descricao), busca) {
				itensFiltrados = append(itensFiltrados, item)
			}
		}
	} else {
		itensFiltrados = dados.Itens
	}

	// Calculate pagination
	totalItems := len(itensFiltrados)
	totalPages := (totalItems + config.ItemsPerPage - 1) / config.ItemsPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Get items for current page
	startIndex := (page - 1) * config.ItemsPerPage
	endIndex := startIndex + config.ItemsPerPage
	if endIndex > totalItems {
		endIndex = totalItems
	}

	var pageItems []Item
	if startIndex < totalItems {
		pageItems = itensFiltrados[startIndex:endIndex]
	}

	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	role, _ := session.Values["role"].(string)

	tmpl.ExecuteTemplate(w, "index.html", struct {
		Itens      []Item
		Estantes   []Estante
		Query      string
		Pagination PaginationData
		Config     Config
		Username   string
		Role       string
	}{
		Itens:    pageItems,
		Estantes: dados.Estantes,
		Query:    r.URL.Query().Get("q"),
		Pagination: PaginationData{
			CurrentPage:  page,
			TotalPages:   totalPages,
			ItemsPerPage: config.ItemsPerPage,
			TotalItems:   totalItems,
		},
		Config:   config,
		Username: username,
		Role:     role,
	})
}

func novoItem(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl := template.Must(template.ParseFiles("templates/novo_item.html"))
		tmpl.Execute(w, struct {
			Error    string
			Item     Item
			Estantes []Estante
			Config   Config
		}{
			Estantes: dados.Estantes,
			Config:   config,
		})
		return
	}

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(10 << 20) // 10MB max memory

		// Check for duplicate location
		estante := r.FormValue("estante")
		prateleira := r.FormValue("prateleira")
		compartimento := r.FormValue("compartimento")

		for _, item := range dados.Itens {
			if item.Estante == estante && item.Prateleira == prateleira && item.Compartimento == compartimento {
				// Return to the form with error message
				tmpl := template.Must(template.ParseFiles("templates/novo_item.html"))
				tmpl.Execute(w, struct {
					Error    string
					Item     Item
					Estantes []Estante
					Config   Config
				}{
					Error: fmt.Sprintf("An item already exists in this location (Shelf: %s, Rack: %s, Compartment: %s)", estante, prateleira, compartimento),
					Item: Item{
						Nome:          r.FormValue("nome"),
						Descricao:     r.FormValue("descricao"),
						Estante:       estante,
						Prateleira:    prateleira,
						Compartimento: compartimento,
					},
					Estantes: dados.Estantes,
					Config:   config,
				})
				return
			}
		}

		file, header, err := r.FormFile("foto")
		var filename string
		if err == nil {
			defer file.Close()
			ext := filepath.Ext(header.Filename)
			filename = fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
			filename, err = saveImage(file, filename)
			if err != nil {
				log.Printf("Error saving image: %v", err)
				filename = ""
			}
		}

		id := len(dados.Itens) + 1
		item := Item{
			ID:            id,
			Nome:          r.FormValue("nome"),
			Descricao:     r.FormValue("descricao"),
			Estante:       estante,
			Prateleira:    prateleira,
			Compartimento: compartimento,
			Foto:          filename,
		}
		dados.Itens = append(dados.Itens, item)
		salvarDados()
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func editarItem(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		id, _ := strconv.Atoi(r.URL.Query().Get("id"))
		var item Item
		for _, i := range dados.Itens {
			if i.ID == id {
				item = i
				break
			}
		}

		tmpl := template.Must(template.ParseFiles("templates/editar.html"))
		tmpl.Execute(w, struct {
			Item   Item
			Config Config
		}{
			Item:   item,
			Config: config,
		})
		return
	}

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(10 << 20) // 10MB max memory

		id, _ := strconv.Atoi(r.FormValue("id"))
		var itemIndex int
		var currentItem Item

		// Find the item and its index
		for i, item := range dados.Itens {
			if item.ID == id {
				itemIndex = i
				currentItem = item
				break
			}
		}

		// Check for duplicate location (excluding current item)
		estante := r.FormValue("estante")
		prateleira := r.FormValue("prateleira")
		compartimento := r.FormValue("compartimento")

		for i, item := range dados.Itens {
			if i != itemIndex && item.Estante == estante && item.Prateleira == prateleira && item.Compartimento == compartimento {
				http.Error(w, "An item already exists in this location (Shelf: "+estante+", Rack: "+prateleira+", Compartment: "+compartimento+")", http.StatusBadRequest)
				return
			}
		}

		// Handle photo upload
		file, header, err := r.FormFile("foto")
		filename := currentItem.Foto // Keep current photo by default
		if err == nil {
			// New photo uploaded
			defer file.Close()
			// Delete old photo if exists
			if currentItem.Foto != "" {
				os.Remove(filepath.Join("static/photos", currentItem.Foto))
				os.Remove(filepath.Join("static/photos/thumbs", currentItem.Foto))
			}
			// Save new photo
			ext := filepath.Ext(header.Filename)
			filename = fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
			filename, err = saveImage(file, filename)
			if err != nil {
				log.Printf("Error saving image: %v", err)
				filename = currentItem.Foto // Keep old photo on error
			}
		}

		// Update item
		dados.Itens[itemIndex] = Item{
			ID:            id,
			Nome:          r.FormValue("nome"),
			Descricao:     r.FormValue("descricao"),
			Estante:       estante,
			Prateleira:    prateleira,
			Compartimento: compartimento,
			Foto:          filename,
		}

		salvarDados()
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func deletarItem(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	for i, item := range dados.Itens {
		if item.ID == id {
			dados.Itens = append(dados.Itens[:i], dados.Itens[i+1:]...)
			break
		}
	}
	salvarDados()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func listarEstantes(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/estantes.html"))
	tmpl.Execute(w, dados)
}

func novaEstante(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		dados.Estantes = append(dados.Estantes, Estante{Nome: r.FormValue("nome")})
		salvarDados()
		http.Redirect(w, r, "/estantes", http.StatusSeeOther)
	}
}

func deletarEstante(w http.ResponseWriter, r *http.Request) {
	nome := r.URL.Query().Get("nome")
	for i, est := range dados.Estantes {
		if est.Nome == nome {
			dados.Estantes = append(dados.Estantes[:i], dados.Estantes[i+1:]...)
			break
		}
	}
	salvarDados()
	http.Redirect(w, r, "/estantes", http.StatusSeeOther)
}

func editarEstante(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		nomeAntigo := r.FormValue("nome_antigo")
		nomeNovo := r.FormValue("nome_novo")

		// Atualiza o nome da estante
		for i, est := range dados.Estantes {
			if est.Nome == nomeAntigo {
				dados.Estantes[i].Nome = nomeNovo
				break
			}
		}

		// Atualiza os itens que usam esta estante
		for i, item := range dados.Itens {
			if item.Estante == nomeAntigo {
				dados.Itens[i].Estante = nomeNovo
			}
		}

		salvarDados()
		http.Redirect(w, r, "/estantes", http.StatusSeeOther)
	}
}

func listarUsuarios(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	session, _ := store.Get(r, "session")
	role, _ := session.Values["role"].(string)
	if role != "admin" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tmpl := template.Must(template.ParseFiles("templates/usuarios.html"))
	tmpl.Execute(w, struct {
		Usuarios []Usuario
		Error    string
		Config   Config
		Username string
		Role     string
	}{
		Usuarios: usuariosData.Usuarios,
		Config:   config,
		Username: session.Values["username"].(string),
		Role:     role,
	})
}

func novoUsuario(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	session, _ := store.Get(r, "session")
	role, _ := session.Values["role"].(string)
	if role != "admin" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(10 << 20) // 10MB max memory

		username := r.FormValue("username")
		password := r.FormValue("password")
		role := r.FormValue("role")

		// Check if username already exists
		for _, user := range usuariosData.Usuarios {
			if user.Username == username {
				tmpl := template.Must(template.ParseFiles("templates/usuarios.html"))
				tmpl.Execute(w, struct {
					Usuarios []Usuario
					Error    string
					Config   Config
					Username string
					Role     string
				}{
					Usuarios: usuariosData.Usuarios,
					Error:    "Username already exists",
					Config:   config,
					Username: session.Values["username"].(string),
					Role:     role,
				})
				return
			}
		}

		// Handle photo upload
		file, header, err := r.FormFile("foto")
		var filename string
		if err == nil {
			defer file.Close()
			ext := filepath.Ext(header.Filename)
			filename = fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
			filename, err = saveImage(file, filename)
			if err != nil {
				log.Printf("Error saving image: %v", err)
				filename = ""
			}
		}

		id := len(usuariosData.Usuarios) + 1
		usuario := Usuario{
			ID:       id,
			Username: username,
			Password: password,
			Role:     role,
			Foto:     filename,
		}
		usuariosData.Usuarios = append(usuariosData.Usuarios, usuario)
		salvarUsuarios()
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	}
}

func editarUsuario(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	session, _ := store.Get(r, "session")
	role, _ := session.Values["role"].(string)
	if role != "admin" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(10 << 20) // 10MB max memory

		id, _ := strconv.Atoi(r.FormValue("id"))
		username := r.FormValue("username")
		password := r.FormValue("password")
		role := r.FormValue("role")

		// Find the user and update
		for i, user := range usuariosData.Usuarios {
			if user.ID == id {
				// Check if username is being changed to an existing one
				if user.Username != username {
					for _, otherUser := range usuariosData.Usuarios {
						if otherUser.Username == username {
							http.Error(w, "Username already exists", http.StatusBadRequest)
							return
						}
					}
				}

				// Handle photo upload
				file, header, err := r.FormFile("foto")
				filename := user.Foto // Keep current photo by default
				if err == nil {
					// New photo uploaded
					defer file.Close()
					// Delete old photo if exists
					if user.Foto != "" {
						os.Remove(filepath.Join("static/photos", user.Foto))
						os.Remove(filepath.Join("static/photos/thumbs", user.Foto))
					}
					// Save new photo
					ext := filepath.Ext(header.Filename)
					filename = fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
					filename, err = saveImage(file, filename)
					if err != nil {
						log.Printf("Error saving image: %v", err)
						filename = user.Foto // Keep old photo on error
					}
				}

				// Update user
				usuariosData.Usuarios[i] = Usuario{
					ID:       id,
					Username: username,
					Password: password,
					Role:     role,
					Foto:     filename,
				}
				salvarUsuarios()
				http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
				return
			}
		}
		http.Error(w, "User not found", http.StatusNotFound)
	}
}

func deletarUsuario(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	session, _ := store.Get(r, "session")
	role, _ := session.Values["role"].(string)
	if role != "admin" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	for i, user := range usuariosData.Usuarios {
		if user.ID == id {
			// Delete user's photo if exists
			if user.Foto != "" {
				os.Remove(filepath.Join("static/photos", user.Foto))
				os.Remove(filepath.Join("static/photos/thumbs", user.Foto))
			}
			usuariosData.Usuarios = append(usuariosData.Usuarios[:i], usuariosData.Usuarios[i+1:]...)
			salvarUsuarios()
			http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
			return
		}
	}
	http.Error(w, "User not found", http.StatusNotFound)
}
