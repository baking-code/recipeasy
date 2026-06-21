package recipes

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func listRecipes(ctx context.Context, db *pgxpool.Pool, userID string, q ListQuery) ([]Recipe, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Query(ctx, `
		SELECT r.id, r.owner_id, r.title, r.description, r.servings,
		       r.prep_time_mins, r.cook_time_mins, r.image_path, r.source_url,
		       r.is_shared, r.created_at, r.updated_at,
		       array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL) AS tags
		FROM recipes r
		LEFT JOIN recipe_tags rt ON rt.recipe_id = r.id
		LEFT JOIN tags t ON t.id = rt.tag_id
		WHERE (r.owner_id = $1 OR r.is_shared = true)
		AND ($2 = '' OR r.search_vector @@ plainto_tsquery('english', $2))
		AND ($3 = '' OR t.name = $3)
		AND ($4 = 0 OR (COALESCE(r.prep_time_mins, 0) + COALESCE(r.cook_time_mins, 0)) <= $4)
		GROUP BY r.id
		ORDER BY r.updated_at DESC
		LIMIT $5 OFFSET $6`,
		uid, q.Search, q.Tag, q.MaxTime, limit, q.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []Recipe
	for rows.Next() {
		var r Recipe
		if err := rows.Scan(
			&r.ID, &r.OwnerID, &r.Title, &r.Description, &r.Servings,
			&r.PrepTimeMins, &r.CookTimeMins, &r.ImagePath, &r.SourceURL,
			&r.IsShared, &r.CreatedAt, &r.UpdatedAt, &r.Tags,
		); err != nil {
			return nil, err
		}
		if r.Tags == nil {
			r.Tags = []string{}
		}
		recipes = append(recipes, r)
	}
	if recipes == nil {
		recipes = []Recipe{}
	}
	return recipes, nil
}

func getFullRecipe(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*Recipe, error) {
	r := &Recipe{}
	err := db.QueryRow(ctx, `
		SELECT id, owner_id, title, description, servings,
		       prep_time_mins, cook_time_mins, image_path, source_url,
		       is_shared, created_at, updated_at
		FROM recipes WHERE id = $1`, id,
	).Scan(
		&r.ID, &r.OwnerID, &r.Title, &r.Description, &r.Servings,
		&r.PrepTimeMins, &r.CookTimeMins, &r.ImagePath, &r.SourceURL,
		&r.IsShared, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Load ingredients
	iRows, err := db.Query(ctx,
		`SELECT id, position, quantity, unit, name FROM ingredients WHERE recipe_id = $1 ORDER BY position`, id)
	if err != nil {
		return nil, err
	}
	defer iRows.Close()
	for iRows.Next() {
		var ing Ingredient
		if err := iRows.Scan(&ing.ID, &ing.Position, &ing.Quantity, &ing.Unit, &ing.Name); err != nil {
			return nil, err
		}
		r.Ingredients = append(r.Ingredients, ing)
	}
	if r.Ingredients == nil {
		r.Ingredients = []Ingredient{}
	}

	// Load steps
	sRows, err := db.Query(ctx,
		`SELECT id, position, instruction, timer_minutes, timer_label FROM steps WHERE recipe_id = $1 ORDER BY position`, id)
	if err != nil {
		return nil, err
	}
	defer sRows.Close()
	for sRows.Next() {
		var s Step
		if err := sRows.Scan(&s.ID, &s.Position, &s.Instruction, &s.TimerMinutes, &s.TimerLabel); err != nil {
			return nil, err
		}
		r.Steps = append(r.Steps, s)
	}
	if r.Steps == nil {
		r.Steps = []Step{}
	}

	// Load tags
	tRows, err := db.Query(ctx,
		`SELECT t.name FROM tags t JOIN recipe_tags rt ON rt.tag_id = t.id WHERE rt.recipe_id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer tRows.Close()
	for tRows.Next() {
		var name string
		if err := tRows.Scan(&name); err != nil {
			return nil, err
		}
		r.Tags = append(r.Tags, name)
	}
	if r.Tags == nil {
		r.Tags = []string{}
	}

	return r, nil
}

func createRecipe(ctx context.Context, db *pgxpool.Pool, userIDStr string, req CreateRecipeRequest) (*Recipe, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var r Recipe
	err = tx.QueryRow(ctx, `
		INSERT INTO recipes (owner_id, title, description, servings, prep_time_mins, cook_time_mins, source_url, is_shared)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, owner_id, title, description, servings, prep_time_mins, cook_time_mins, image_path, source_url, is_shared, created_at, updated_at`,
		userID, req.Title, req.Description, req.Servings, req.PrepTimeMins, req.CookTimeMins, req.SourceURL, req.IsShared,
	).Scan(
		&r.ID, &r.OwnerID, &r.Title, &r.Description, &r.Servings,
		&r.PrepTimeMins, &r.CookTimeMins, &r.ImagePath, &r.SourceURL,
		&r.IsShared, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := upsertRelated(ctx, tx, r.ID, req); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	r.Ingredients = ingredientsFromInput(req.Ingredients)
	r.Steps = stepsFromInput(req.Steps)
	r.Tags = req.Tags
	return &r, nil
}

func updateRecipe(ctx context.Context, db *pgxpool.Pool, userIDStr string, id uuid.UUID, req CreateRecipeRequest) (*Recipe, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var r Recipe
	err = tx.QueryRow(ctx, `
		UPDATE recipes
		SET title=$2, description=$3, servings=$4, prep_time_mins=$5, cook_time_mins=$6, source_url=$7, is_shared=$8
		WHERE id=$1
		RETURNING id, owner_id, title, description, servings, prep_time_mins, cook_time_mins, image_path, source_url, is_shared, created_at, updated_at`,
		id, req.Title, req.Description, req.Servings, req.PrepTimeMins, req.CookTimeMins, req.SourceURL, req.IsShared,
	).Scan(
		&r.ID, &r.OwnerID, &r.Title, &r.Description, &r.Servings,
		&r.PrepTimeMins, &r.CookTimeMins, &r.ImagePath, &r.SourceURL,
		&r.IsShared, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Clear and re-insert related rows
	tx.Exec(ctx, `DELETE FROM ingredients WHERE recipe_id = $1`, id)
	tx.Exec(ctx, `DELETE FROM steps WHERE recipe_id = $1`, id)
	tx.Exec(ctx, `DELETE FROM recipe_tags WHERE recipe_id = $1`, id)

	if err := upsertRelated(ctx, tx, r.ID, req); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	r.Ingredients = ingredientsFromInput(req.Ingredients)
	r.Steps = stepsFromInput(req.Steps)
	r.Tags = req.Tags
	return &r, nil
}

func deleteRecipe(ctx context.Context, db *pgxpool.Pool, userIDStr string, id uuid.UUID) error {
	_, err := db.Exec(ctx, `DELETE FROM recipes WHERE id = $1`, id)
	return err
}

func upsertRelated(ctx context.Context, tx pgx.Tx, recipeID uuid.UUID, req CreateRecipeRequest) error {
	for i, ing := range req.Ingredients {
		pos := ing.Position
		if pos == 0 {
			pos = i + 1
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO ingredients (recipe_id, position, quantity, unit, name) VALUES ($1,$2,$3,$4,$5)`,
			recipeID, pos, ing.Quantity, ing.Unit, ing.Name,
		); err != nil {
			return fmt.Errorf("insert ingredient: %w", err)
		}
	}

	for i, step := range req.Steps {
		pos := step.Position
		if pos == 0 {
			pos = i + 1
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO steps (recipe_id, position, instruction, timer_minutes, timer_label) VALUES ($1,$2,$3,$4,$5)`,
			recipeID, pos, step.Instruction, step.TimerMinutes, step.TimerLabel,
		); err != nil {
			return fmt.Errorf("insert step: %w", err)
		}
	}

	for _, tagName := range req.Tags {
		var tagID uuid.UUID
		if err := tx.QueryRow(ctx,
			`INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING id`,
			tagName,
		).Scan(&tagID); err != nil {
			return fmt.Errorf("upsert tag: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO recipe_tags (recipe_id, tag_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
			recipeID, tagID,
		); err != nil {
			return fmt.Errorf("insert recipe_tag: %w", err)
		}
	}

	return nil
}

func ingredientsFromInput(inputs []IngredientInput) []Ingredient {
	out := make([]Ingredient, len(inputs))
	for i, ing := range inputs {
		out[i] = Ingredient{
			ID:       uuid.New(),
			Position: ing.Position,
			Quantity: ing.Quantity,
			Unit:     ing.Unit,
			Name:     ing.Name,
		}
	}
	return out
}

func stepsFromInput(inputs []StepInput) []Step {
	out := make([]Step, len(inputs))
	for i, s := range inputs {
		out[i] = Step{
			ID:           uuid.New(),
			Position:     s.Position,
			Instruction:  s.Instruction,
			TimerMinutes: s.TimerMinutes,
			TimerLabel:   s.TimerLabel,
		}
	}
	return out
}
