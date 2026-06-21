// One-off script to migrate recipes from the legacy server/data/test.json into PostgreSQL.
// Usage: DATABASE_URL=... OWNER_EMAIL=you@gmail.com go run ./scripts/migrate-legacy
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type legacyRecipe struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Method      []string `json:"method"`
	Ingredients []string `json:"ingredients"`
	Time        int      `json:"time"`
	Categories  string   `json:"categories"`
	Tags        []struct {
		ID   int    `json:"id"`
		Text string `json:"text"`
	} `json:"tags"`
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	ownerEmail := os.Getenv("OWNER_EMAIL")

	if dbURL == "" || ownerEmail == "" {
		log.Fatal("DATABASE_URL and OWNER_EMAIL are required")
	}

	dataFile := os.Getenv("DATA_FILE")
	if dataFile == "" {
		dataFile = "server/data/test.json"
	}

	raw, err := os.ReadFile(dataFile)
	if err != nil {
		log.Fatalf("read data file: %v", err)
	}

	var rawMap map[string]legacyRecipe
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		log.Fatalf("parse data file: %v", err)
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	// Get owner user id
	var ownerID string
	if err := db.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, strings.ToLower(ownerEmail)).Scan(&ownerID); err != nil {
		log.Fatalf("owner user not found (run the app and login first to create user): %v", err)
	}

	migrated := 0
	for _, r := range rawMap {
		tx, err := db.Begin(ctx)
		if err != nil {
			log.Printf("begin tx: %v", err)
			continue
		}

		var recipeID string
		err = tx.QueryRow(ctx, `
			INSERT INTO recipes (owner_id, title, description, cook_time_mins, is_shared)
			VALUES ($1, $2, $3, $4, true)
			RETURNING id`,
			ownerID, r.Name, r.Description, r.Time,
		).Scan(&recipeID)
		if err != nil {
			tx.Rollback(ctx)
			log.Printf("insert recipe %s: %v", r.Name, err)
			continue
		}

		for i, ing := range r.Ingredients {
			tx.Exec(ctx,
				`INSERT INTO ingredients (recipe_id, position, name) VALUES ($1, $2, $3)`,
				recipeID, i+1, ing,
			)
		}

		for i, step := range r.Method {
			tx.Exec(ctx,
				`INSERT INTO steps (recipe_id, position, instruction) VALUES ($1, $2, $3)`,
				recipeID, i+1, step,
			)
		}

		for _, tag := range r.Tags {
			var tagID string
			tx.QueryRow(ctx,
				`INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING id`,
				tag.Text,
			).Scan(&tagID)
			if tagID != "" {
				tx.Exec(ctx, `INSERT INTO recipe_tags (recipe_id, tag_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, recipeID, tagID)
			}
		}

		if err := tx.Commit(ctx); err != nil {
			log.Printf("commit recipe %s: %v", r.Name, err)
			continue
		}
		migrated++
		fmt.Printf("  migrated: %s\n", r.Name)
	}

	fmt.Printf("\nDone. Migrated %d/%d recipes.\n", migrated, len(rawMap))
}
