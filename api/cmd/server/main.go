package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/benking/recipeasy/api/internal/auth"
	"github.com/benking/recipeasy/api/internal/middleware"
	"github.com/benking/recipeasy/api/internal/recipes"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	allowedEmails := os.Getenv("ALLOWED_EMAILS")

	authHandler := auth.NewHandler(pool, jwtSecret, googleClientID, googleClientSecret, googleRedirectURL, allowedEmails)
	recipesHandler := recipes.NewHandler(pool)
	jwtMiddleware := middleware.NewJWTMiddleware(jwtSecret)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{os.Getenv("FRONTEND_URL"), "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/auth/google", authHandler.GoogleLogin)
		r.Get("/auth/google/callback", authHandler.GoogleCallback)

		r.Group(func(r chi.Router) {
			r.Use(jwtMiddleware.Authenticate)

			r.Get("/recipes", recipesHandler.List)
			r.Post("/recipes", recipesHandler.Create)
			r.Get("/recipes/{id}", recipesHandler.Get)
			r.Put("/recipes/{id}", recipesHandler.Update)
			r.Delete("/recipes/{id}", recipesHandler.Delete)
			r.Post("/recipes/{id}/image", recipesHandler.UploadImage)

			r.Post("/import/url", recipesHandler.ImportURL)
			r.Post("/import/photo", recipesHandler.ImportPhoto)

			r.Get("/tags", recipesHandler.ListTags)
		})
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
