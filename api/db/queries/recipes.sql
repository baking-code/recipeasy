-- name: ListRecipes :many
SELECT r.*,
       array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL) AS tags
FROM recipes r
LEFT JOIN recipe_tags rt ON rt.recipe_id = r.id
LEFT JOIN tags t ON t.id = rt.tag_id
WHERE (
    r.owner_id = @user_id
    OR r.is_shared = true
)
AND (
    @search::TEXT = ''
    OR r.search_vector @@ plainto_tsquery('english', @search)
)
AND (
    @tag_filter::TEXT = ''
    OR t.name = @tag_filter
)
AND (
    @max_time::INT = 0
    OR (COALESCE(r.prep_time_mins, 0) + COALESCE(r.cook_time_mins, 0)) <= @max_time
)
GROUP BY r.id
ORDER BY r.updated_at DESC
LIMIT @lim OFFSET @off;

-- name: GetRecipe :one
SELECT * FROM recipes WHERE id = $1;

-- name: CreateRecipe :one
INSERT INTO recipes (owner_id, title, description, servings, prep_time_mins, cook_time_mins, image_path, source_url, is_shared)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateRecipe :one
UPDATE recipes
SET title = $2,
    description = $3,
    servings = $4,
    prep_time_mins = $5,
    cook_time_mins = $6,
    image_path = $7,
    source_url = $8,
    is_shared = $9
WHERE id = $1
RETURNING *;

-- name: DeleteRecipe :exec
DELETE FROM recipes WHERE id = $1;

-- name: UpdateRecipeImagePath :one
UPDATE recipes SET image_path = $2 WHERE id = $1 RETURNING *;

-- name: GetIngredients :many
SELECT * FROM ingredients WHERE recipe_id = $1 ORDER BY position;

-- name: DeleteIngredients :exec
DELETE FROM ingredients WHERE recipe_id = $1;

-- name: InsertIngredient :one
INSERT INTO ingredients (recipe_id, position, quantity, unit, name)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSteps :many
SELECT * FROM steps WHERE recipe_id = $1 ORDER BY position;

-- name: DeleteSteps :exec
DELETE FROM steps WHERE recipe_id = $1;

-- name: InsertStep :one
INSERT INTO steps (recipe_id, position, instruction, timer_minutes, timer_label)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetOrCreateTag :one
INSERT INTO tags (name) VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: DeleteRecipeTags :exec
DELETE FROM recipe_tags WHERE recipe_id = $1;

-- name: AddRecipeTag :exec
INSERT INTO recipe_tags (recipe_id, tag_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetRecipeTags :many
SELECT t.name FROM tags t
JOIN recipe_tags rt ON rt.tag_id = t.id
WHERE rt.recipe_id = $1;

-- name: ListAllTags :many
SELECT DISTINCT t.name FROM tags t
JOIN recipe_tags rt ON rt.tag_id = t.id
ORDER BY t.name;
