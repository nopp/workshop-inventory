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
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"` // This will store the hashed password
}

type Config struct {
	Title            string `json:"title"`
	ItemsPerPage     int    `json:"items_per_page"`
	PhotoThumbSize   int    `json:"photo_thumbnail_size"`
	PhotoPreviewSize int    `json:"photo_preview_size"`
	SecretKey        string `json:"secret_key"`
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

var (
	dados  Inventario
	config Config
	store  *sessions.CookieStore
	users  []User
)

func carregarConfig() {
	file, err := os.ReadFile("config.json")
	if err != nil {
		log.Printf("Warning: config.json not found, using default values")
		config = Config{
			Title:            "Workshop Inventory",
			ItemsPerPage:     10,
			PhotoThumbSize:   200,
			PhotoPreviewSize: 600,
			SecretKey:        "your-secret-key-here", // Change this in production
		}
		return
	}
	if err := json.Unmarshal(file, &config); err != nil {
		log.Printf("Error loading config: %v", err)
		config = Config{
			Title:            "Workshop Inventory",
			ItemsPerPage:     10,
			PhotoThumbSize:   200,
			PhotoPreviewSize: 600,
			SecretKey:        "your-secret-key-here", // Change this in production
		}
	}
	store = sessions.NewCookieStore([]byte(config.SecretKey))
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
	file, err := os.ReadFile("users.json")
	if err == nil {
		json.Unmarshal(file, &users)
	} else {
		// Create default admin user if no users file exists
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		users = []User{
			{
				Username: "admin",
				Password: string(hashedPassword),
			},
		}
		salvarUsuarios()
	}
}

func salvarUsuarios() {
	data, _ := json.MarshalIndent(users, "", "  ")
	ioutil.WriteFile("users.json", data, 0644)
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

		for _, user := range users {
			if user.Username == username {
				err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
				if err == nil {
					session, _ := store.Get(r, "session")
					session.Values["authenticated"] = true
					session.Values["username"] = username
					session.Save(r, w)
					http.Redirect(w, r, "/", http.StatusSeeOther)
					return
				}
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
	http.HandleFunc("/novo", requireAuth(novoItem))
	http.HandleFunc("/editar", requireAuth(editarItem))
	http.HandleFunc("/deletar", requireAuth(deletarItem))
	http.HandleFunc("/estantes", requireAuth(listarEstantes))
	http.HandleFunc("/estantes/novo", requireAuth(novaEstante))
	http.HandleFunc("/estantes/editar", requireAuth(editarEstante))
	http.HandleFunc("/estantes/deletar", requireAuth(deletarEstante))

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
	if page > totalPages {
		page = totalPages
	}

	// Get items for current page
	startIndex := (page - 1) * config.ItemsPerPage
	endIndex := startIndex + config.ItemsPerPage
	if endIndex > totalItems {
		endIndex = totalItems
	}
	pageItems := itensFiltrados[startIndex:endIndex]

	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)

	tmpl.ExecuteTemplate(w, "index.html", struct {
		Itens      []Item
		Estantes   []Estante
		Query      string
		Pagination PaginationData
		Config     Config
		Username   string
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
