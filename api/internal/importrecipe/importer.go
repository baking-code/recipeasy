package importrecipe

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gocolly/colly/v2"
)

// DraftRecipe is an unsaved recipe returned to the client for review before saving.
type DraftRecipe struct {
	Title        string             `json:"title"`
	Description  string             `json:"description"`
	Servings     *int               `json:"servings,omitempty"`
	PrepTimeMins *int               `json:"prep_time_mins,omitempty"`
	CookTimeMins *int               `json:"cook_time_mins,omitempty"`
	SourceURL    string             `json:"source_url,omitempty"`
	Tags         []string           `json:"tags"`
	Ingredients  []DraftIngredient  `json:"ingredients"`
	Steps        []DraftStep        `json:"steps"`
}

type DraftIngredient struct {
	Quantity string `json:"quantity,omitempty"`
	Unit     string `json:"unit,omitempty"`
	Name     string `json:"name"`
}

type DraftStep struct {
	Instruction  string `json:"instruction"`
	TimerMinutes *int   `json:"timer_minutes,omitempty"`
	TimerLabel   string `json:"timer_label,omitempty"`
}

type Importer struct {
	geminiAPIKey string
}

func NewImporter(geminiAPIKey string) *Importer {
	return &Importer{geminiAPIKey: geminiAPIKey}
}

// FromURL scrapes a URL. Tries schema.org JSON-LD first; falls back to Gemini.
func (im *Importer) FromURL(ctx context.Context, pageURL string) (*DraftRecipe, error) {
	var jsonLD string

	c := colly.NewCollector()
	c.OnHTML(`script[type="application/ld+json"]`, func(e *colly.HTMLElement) {
		text := e.Text
		if strings.Contains(text, `"Recipe"`) || strings.Contains(text, `"@type":"Recipe"`) {
			jsonLD = text
		}
	})

	var bodyText string
	c.OnHTML(`main, article, .recipe, #recipe`, func(e *colly.HTMLElement) {
		if bodyText == "" {
			bodyText = e.Text
		}
	})

	if err := c.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}

	if jsonLD != "" {
		draft, err := parseSchemaOrg(jsonLD)
		if err == nil {
			draft.SourceURL = pageURL
			return draft, nil
		}
	}

	// Fallback: Gemini text extraction
	if im.geminiAPIKey == "" {
		return nil, fmt.Errorf("no schema.org data found and GEMINI_API_KEY is not set")
	}
	return im.extractWithGeminiText(ctx, bodyText, pageURL)
}

// FromPhoto sends an image to Gemini Vision for OCR extraction.
func (im *Importer) FromPhoto(ctx context.Context, imageData []byte) (*DraftRecipe, error) {
	if im.geminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}
	return im.extractWithGeminiVision(ctx, imageData)
}

// parseSchemaOrg parses a schema.org Recipe JSON-LD blob.
func parseSchemaOrg(raw string) (*DraftRecipe, error) {
	var data map[string]interface{}

	// Handle @graph wrapper
	var wrapper struct {
		Graph []map[string]interface{} `json:"@graph"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapper); err == nil && len(wrapper.Graph) > 0 {
		for _, node := range wrapper.Graph {
			if t, _ := node["@type"].(string); t == "Recipe" {
				data = node
				break
			}
		}
	}
	if data == nil {
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			return nil, fmt.Errorf("unmarshal schema.org: %w", err)
		}
	}

	draft := &DraftRecipe{
		Tags:        []string{},
		Ingredients: []DraftIngredient{},
		Steps:       []DraftStep{},
	}

	if v, ok := data["name"].(string); ok {
		draft.Title = v
	}
	if v, ok := data["description"].(string); ok {
		draft.Description = v
	}

	if ingredients, ok := data["recipeIngredient"].([]interface{}); ok {
		for _, ing := range ingredients {
			if s, ok := ing.(string); ok {
				draft.Ingredients = append(draft.Ingredients, DraftIngredient{Name: s})
			}
		}
	}

	if instructions, ok := data["recipeInstructions"].([]interface{}); ok {
		for _, inst := range instructions {
			switch v := inst.(type) {
			case string:
				draft.Steps = append(draft.Steps, DraftStep{Instruction: v})
			case map[string]interface{}:
				text, _ := v["text"].(string)
				if text != "" {
					draft.Steps = append(draft.Steps, DraftStep{Instruction: text})
				}
			}
		}
	}

	if keywords, ok := data["keywords"].(string); ok {
		for _, k := range strings.Split(keywords, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				draft.Tags = append(draft.Tags, k)
			}
		}
	}

	if draft.Title == "" {
		return nil, fmt.Errorf("no recipe title found in schema.org data")
	}
	return draft, nil
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string          `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inline_data,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

const extractionPrompt = `Extract this recipe and return ONLY valid JSON with this exact structure:
{
  "title": "Recipe Name",
  "description": "Brief description",
  "servings": 4,
  "prep_time_mins": 15,
  "cook_time_mins": 30,
  "tags": ["tag1", "tag2"],
  "ingredients": [
    {"quantity": "200", "unit": "g", "name": "plain flour"}
  ],
  "steps": [
    {"instruction": "Step text", "timer_minutes": 20, "timer_label": "simmer the sauce"}
  ]
}
Use null for servings/times if not mentioned. Omit timer_minutes/timer_label on steps that have no timing. Return ONLY the JSON, no markdown.`

func (im *Importer) extractWithGeminiText(ctx context.Context, text, sourceURL string) (*DraftRecipe, error) {
	prompt := extractionPrompt + "\n\nRecipe text:\n" + text
	return im.callGemini(ctx, []geminiPart{{Text: prompt}}, sourceURL)
}

func (im *Importer) extractWithGeminiVision(ctx context.Context, imageData []byte) (*DraftRecipe, error) {
	mimeType := http.DetectContentType(imageData)
	encoded := base64.StdEncoding.EncodeToString(imageData)
	parts := []geminiPart{
		{Text: extractionPrompt},
		{InlineData: &geminiInlineData{MimeType: mimeType, Data: encoded}},
	}
	return im.callGemini(ctx, parts, "")
}

func (im *Importer) callGemini(ctx context.Context, parts []geminiPart, sourceURL string) (*DraftRecipe, error) {
	reqBody := geminiRequest{
		Contents: []geminiContent{{Parts: parts}},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s",
		im.geminiAPIKey,
	)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini returned %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode gemini response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from gemini")
	}

	rawJSON := result.Candidates[0].Content.Parts[0].Text
	rawJSON = strings.TrimPrefix(rawJSON, "```json")
	rawJSON = strings.TrimPrefix(rawJSON, "```")
	rawJSON = strings.TrimSuffix(rawJSON, "```")
	rawJSON = strings.TrimSpace(rawJSON)

	var draft DraftRecipe
	if err := json.Unmarshal([]byte(rawJSON), &draft); err != nil {
		return nil, fmt.Errorf("parse gemini json: %w", err)
	}

	draft.SourceURL = sourceURL
	if draft.Tags == nil {
		draft.Tags = []string{}
	}
	if draft.Ingredients == nil {
		draft.Ingredients = []DraftIngredient{}
	}
	if draft.Steps == nil {
		draft.Steps = []DraftStep{}
	}

	return &draft, nil
}
