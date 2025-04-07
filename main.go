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

	"github.com/nfnt/resize"
)

type Config struct {
	Title            string `json:"title"`
	ItemsPerPage     int    `json:"items_per_page"`
	PhotoThumbSize   int    `json:"photo_thumbnail_size"`
	PhotoPreviewSize int    `json:"photo_preview_size"`
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
		}
	}
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		listarItens(w, r, tmpl)
	})
	http.HandleFunc("/novo", novoItem)
	http.HandleFunc("/editar", editarItem)
	http.HandleFunc("/deletar", deletarItem)

	http.HandleFunc("/estantes", listarEstantes)
	http.HandleFunc("/estantes/novo", novaEstante)
	http.HandleFunc("/estantes/editar", editarEstante)
	http.HandleFunc("/estantes/deletar", deletarEstante)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

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

	tmpl.ExecuteTemplate(w, "index.html", struct {
		Itens      []Item
		Estantes   []Estante
		Query      string
		Pagination PaginationData
		Config     Config
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
		Config: config,
	})
}

func novoItem(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseMultipartForm(10 << 20) // 10MB max memory

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
			Estante:       r.FormValue("estante"),
			Prateleira:    r.FormValue("prateleira"),
			Compartimento: r.FormValue("compartimento"),
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
			Estante:       r.FormValue("estante"),
			Prateleira:    r.FormValue("prateleira"),
			Compartimento: r.FormValue("compartimento"),
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
