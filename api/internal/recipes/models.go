package recipes

import (
	"time"

	"github.com/google/uuid"
)

type Recipe struct {
	ID            uuid.UUID   `json:"id"`
	OwnerID       uuid.UUID   `json:"owner_id"`
	Title         string      `json:"title"`
	Description   *string     `json:"description,omitempty"`
	Servings      *int        `json:"servings,omitempty"`
	PrepTimeMins  *int        `json:"prep_time_mins,omitempty"`
	CookTimeMins  *int        `json:"cook_time_mins,omitempty"`
	ImagePath     *string     `json:"image_path,omitempty"`
	SourceURL     *string     `json:"source_url,omitempty"`
	IsShared      bool        `json:"is_shared"`
	Tags          []string    `json:"tags"`
	Ingredients   []Ingredient `json:"ingredients"`
	Steps         []Step       `json:"steps"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type Ingredient struct {
	ID       uuid.UUID `json:"id"`
	Position int       `json:"position"`
	Quantity *string   `json:"quantity,omitempty"`
	Unit     *string   `json:"unit,omitempty"`
	Name     string    `json:"name"`
}

type Step struct {
	ID           uuid.UUID `json:"id"`
	Position     int       `json:"position"`
	Instruction  string    `json:"instruction"`
	TimerMinutes *int      `json:"timer_minutes,omitempty"`
	TimerLabel   *string   `json:"timer_label,omitempty"`
}

type CreateRecipeRequest struct {
	Title        string       `json:"title"`
	Description  *string      `json:"description"`
	Servings     *int         `json:"servings"`
	PrepTimeMins *int         `json:"prep_time_mins"`
	CookTimeMins *int         `json:"cook_time_mins"`
	SourceURL    *string      `json:"source_url"`
	IsShared     bool         `json:"is_shared"`
	Tags         []string     `json:"tags"`
	Ingredients  []IngredientInput `json:"ingredients"`
	Steps        []StepInput  `json:"steps"`
}

type IngredientInput struct {
	Position int     `json:"position"`
	Quantity *string `json:"quantity"`
	Unit     *string `json:"unit"`
	Name     string  `json:"name"`
}

type StepInput struct {
	Position     int     `json:"position"`
	Instruction  string  `json:"instruction"`
	TimerMinutes *int    `json:"timer_minutes"`
	TimerLabel   *string `json:"timer_label"`
}

type ListQuery struct {
	Search  string `json:"q"`
	Tag     string `json:"tag"`
	MaxTime int    `json:"max_time"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
}

type ImportURLRequest struct {
	URL string `json:"url"`
}
