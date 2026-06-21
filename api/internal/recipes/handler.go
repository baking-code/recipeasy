package recipes

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/benking/recipeasy/api/internal/middleware"
	importrecipe "github.com/benking/recipeasy/api/internal/importrecipe"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	db        *pgxpool.Pool
	importer  *importrecipe.Importer
	imagePath string
}

func NewHandler(db *pgxpool.Pool) *Handler {
	imgPath := os.Getenv("IMAGE_DIR")
	if imgPath == "" {
		imgPath = "/var/recipeasy/images"
	}
	return &Handler{
		db:        db,
		importer:  importrecipe.NewImporter(os.Getenv("GEMINI_API_KEY")),
		imagePath: imgPath,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	q := ListQuery{
		Search: r.URL.Query().Get("q"),
		Tag:    r.URL.Query().Get("tag"),
		Limit:  50,
	}
	if mt := r.URL.Query().Get("max_time"); mt != "" {
		q.MaxTime, _ = strconv.Atoi(mt)
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		q.Limit, _ = strconv.Atoi(l)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		q.Offset, _ = strconv.Atoi(o)
	}

	recipeList, err := listRecipes(r.Context(), h.db, userID, q)
	if err != nil {
		log.Printf("list recipes: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list recipes")
		return
	}
	writeJSON(w, http.StatusOK, recipeList)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	recipe, err := getFullRecipe(r.Context(), h.db, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		log.Printf("get recipe: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get recipe")
		return
	}
	writeJSON(w, http.StatusOK, recipe)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var req CreateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	recipe, err := createRecipe(r.Context(), h.db, userID, req)
	if err != nil {
		log.Printf("create recipe: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create recipe")
		return
	}
	writeJSON(w, http.StatusCreated, recipe)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req CreateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	recipe, err := updateRecipe(r.Context(), h.db, userID, id, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		log.Printf("update recipe: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update recipe")
		return
	}
	writeJSON(w, http.StatusOK, recipe)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := deleteRecipe(r.Context(), h.db, userID, id); err != nil {
		log.Printf("delete recipe: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete recipe")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	r.ParseMultipartForm(10 << 20) // 10MB
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "image file required")
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s%s", id.String(), ext)
	destPath := filepath.Join(h.imagePath, filename)

	dest, err := os.Create(destPath)
	if err != nil {
		log.Printf("create image file: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to save image")
		return
	}
	defer dest.Close()
	io.Copy(dest, file)

	imagePath := "/images/" + filename
	if _, err := h.db.Exec(r.Context(),
		`UPDATE recipes SET image_path = $1 WHERE id = $2`, imagePath, id); err != nil {
		log.Printf("update image path: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update image path")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"image_path": imagePath})
}

func (h *Handler) ImportURL(w http.ResponseWriter, r *http.Request) {
	var req ImportURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	draft, err := h.importer.FromURL(r.Context(), req.URL)
	if err != nil {
		log.Printf("import url %s: %v", req.URL, err)
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("failed to import: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

func (h *Handler) ImportPhoto(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(20 << 20) // 20MB
	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "image file required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read image")
		return
	}

	draft, err := h.importer.FromPhoto(r.Context(), data)
	if err != nil {
		log.Printf("import photo: %v", err)
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("failed to import: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(),
		`SELECT DISTINCT t.name FROM tags t JOIN recipe_tags rt ON rt.tag_id = t.id ORDER BY t.name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tags")
		return
	}
	defer rows.Close()

	tags := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		tags = append(tags, name)
	}
	writeJSON(w, http.StatusOK, tags)
}
