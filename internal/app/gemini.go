package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GeminiClient calls the Google Gemini API to generate baby development content.
type GeminiClient struct {
	APIKey     string
	Model      string
	HTTPClient *http.Client
	MaxRetries int           // maximum number of retry attempts (default 3)
	BaseDelay  time.Duration // initial backoff delay (default 1s, doubles each retry)
}

// NewGeminiClient creates a new GeminiClient.
// Model defaults to "gemini-2.5-flash" if empty.
func NewGeminiClient(apiKey, model string) *GeminiClient {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &GeminiClient{
		APIKey: apiKey,
		Model:  model,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
	}
}

// geminiRequest is the request body for the Gemini generateContent API.
type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig map[string]interface{} `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

// geminiResponse is the response from the Gemini generateContent API.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

// GenerateDevelopmentContent generates baby development content for a given week.
func (g *GeminiClient) GenerateDevelopmentContent(weekNumber int, dob time.Time) (string, error) {
	if g.APIKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY is not configured")
	}

	now := time.Now()
	ageInDays := int(now.Sub(dob).Hours() / 24)
	ageMonths := ageInDays / 30
	ageWeeks := ageInDays / 7

	prompt := fmt.Sprintf(`You are a pediatric child development expert. Generate comprehensive development information for a baby who is %d weeks old (approximately %d months old, born on %s). The content should be for week %d of the baby's life.

Return ONLY valid JSON (no markdown, no code fences) with this exact structure:
{
  "week_number": %d,
  "age_description": "X weeks old (~Y months)",
  "behaviors": [
    {"title": "Behavior name", "description": "Detailed description of what to expect"}
  ],
  "milestones": [
    {"title": "Milestone name", "description": "What this milestone looks like", "typical_range": "X-Y weeks"}
  ],
  "wonder_week": {
    "is_active": true/false,
    "leap_number": null or number (1-10),
    "name": "Name of the wonder week/leap if active, or empty string",
    "description": "What happens during this wonder week",
    "signs": ["Sign 1", "Sign 2"],
    "handling_tips": ["Tip 1", "Tip 2"]
  },
  "upcoming_wonder_weeks": [
    {"week": 5, "leap_number": 1, "name": "The World of Changing Sensations", "weeks_away": 2}
  ],
  "exercises": [
    {
      "name": "Exercise name",
      "icon": "single relevant emoji",
      "instructions": "Step-by-step instructions for the exercise",
      "benefits": "What this exercise helps develop",
      "duration": "Recommended duration"
    }
  ]
}

Important guidelines:
- Include 3-5 behaviors typical for this age
- Include 3-5 developmental milestones with realistic typical ranges
- For wonder weeks, use the established "The Wonder Weeks" framework by van de Rijt and Plooij. The 10 leaps occur approximately at weeks 5, 8, 12, 19, 26, 37, 46, 55, 64, and 75
- If the baby is currently in or near a wonder week (within 1 week), set is_active to true
- Include 2-3 upcoming wonder weeks with weeks_away calculated from week %d
- Include 4-6 age-appropriate exercises with emoji icons
- All exercises must be safe for the baby's age
- Be specific and practical in descriptions
- Use encouraging, supportive language for parents`, ageWeeks, ageMonths, dob.Format("2006-01-02"), weekNumber, weekNumber, weekNumber)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: map[string]interface{}{
			"temperature":      0.7,
			"maxOutputTokens":  4096,
			"responseMimeType": "application/json",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.Model, g.APIKey)

	log.Printf("Calling Gemini API for week %d (model: %s)", weekNumber, g.Model)

	// Retry loop for invalid JSON content (separate from HTTP-level retries in callWithRetry)
	const maxJSONRetries = 2
	var lastJSONErr error

	for jsonAttempt := 0; jsonAttempt <= maxJSONRetries; jsonAttempt++ {
		if jsonAttempt > 0 {
			log.Printf("Retrying Gemini call for week %d due to invalid JSON (attempt %d/%d): %v", weekNumber, jsonAttempt, maxJSONRetries, lastJSONErr)
			time.Sleep(time.Duration(jsonAttempt) * time.Second)
		}

		body, err := g.callWithRetry(url, jsonBody)
		if err != nil {
			return "", err
		}

		var geminiResp geminiResponse
		if err := json.Unmarshal(body, &geminiResp); err != nil {
			lastJSONErr = fmt.Errorf("failed to parse Gemini response: %w", err)
			continue
		}

		if geminiResp.Error != nil {
			return "", fmt.Errorf("Gemini API error: %s", geminiResp.Error.Message)
		}

		if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
			lastJSONErr = fmt.Errorf("Gemini returned empty response")
			continue
		}

		text := geminiResp.Candidates[0].Content.Parts[0].Text

		// Strip markdown code fences if present
		text = strings.TrimSpace(text)
		if strings.HasPrefix(text, "```json") {
			text = strings.TrimPrefix(text, "```json")
			text = strings.TrimSuffix(text, "```")
			text = strings.TrimSpace(text)
		} else if strings.HasPrefix(text, "```") {
			text = strings.TrimPrefix(text, "```")
			text = strings.TrimSuffix(text, "```")
			text = strings.TrimSpace(text)
		}

		// Validate JSON
		var js json.RawMessage
		if err := json.Unmarshal([]byte(text), &js); err != nil {
			n := len(text)
			if n > 200 {
				n = 200
			}
			lastJSONErr = fmt.Errorf("Gemini returned invalid JSON: %w\nRaw: %s", err, text[:n])
			continue
		}

		return text, nil
	}

	return "", fmt.Errorf("Gemini returned invalid JSON after %d attempts: %w", maxJSONRetries+1, lastJSONErr)
}

// callWithRetry performs an HTTP POST with exponential backoff retry for transient errors.
// Retries on: network errors, 429 (rate limit), and 5xx (server errors).
// Respects Retry-After header from 429 responses.
func (g *GeminiClient) callWithRetry(url string, jsonBody []byte) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= g.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := g.backoffDelay(attempt, lastErr)
			log.Printf("Gemini API retry %d/%d after %v (previous error: %v)", attempt, g.MaxRetries, delay, lastErr)
			time.Sleep(delay)
		}

		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := g.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("Gemini API request failed: %w", err)
			continue // network error → retry
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue // read error → retry
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		lastErr = &apiError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
			RetryAfter: resp.Header.Get("Retry-After"),
		}

		if !isRetryableStatus(resp.StatusCode) {
			// Non-retryable error (400, 401, 403, 404, etc.) — fail immediately
			return nil, lastErr
		}
	}

	return nil, fmt.Errorf("Gemini API failed after %d retries: %w", g.MaxRetries+1, lastErr)
}

// backoffDelay calculates the delay for a retry attempt using exponential backoff.
// For 429 responses with Retry-After header, it uses that value instead.
func (g *GeminiClient) backoffDelay(attempt int, lastErr error) time.Duration {
	// Check if the last error was a 429 with Retry-After header
	if apiErr, ok := lastErr.(*apiError); ok && apiErr.RetryAfter != "" {
		if seconds, err := strconv.Atoi(apiErr.RetryAfter); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}

	// Exponential backoff: baseDelay * 2^(attempt-1), capped at 30s
	delay := time.Duration(float64(g.BaseDelay) * math.Pow(2, float64(attempt-1)))
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	return delay
}

// isRetryableStatus returns true for HTTP status codes that warrant a retry.
func isRetryableStatus(code int) bool {
	return code == 429 || code >= 500
}

// apiError represents a non-OK response from the Gemini API.
type apiError struct {
	StatusCode int
	Body       string
	RetryAfter string // from Retry-After header (429 responses)
}

func (e *apiError) Error() string {
	return fmt.Sprintf("Gemini API returned status %d: %s", e.StatusCode, e.Body)
}

// GeneratePersonalizedInsight generates AI-powered personalized insights for a baby.
func (g *GeminiClient) GeneratePersonalizedInsight(profile BabyProfile, latestGrowth *GrowthMeasurement, feedingAvgML, sleepAvgMin int) (string, error) {
	if g.APIKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY is not configured")
	}

	now := time.Now()
	ageInDays := int(now.Sub(profile.DateOfBirth).Hours() / 24)
	ageWeeks := ageInDays / 7
	ageMonths := ageInDays / 30

	gender := profile.Gender
	if gender == "" {
		gender = "unknown"
	}
	milkType := profile.MilkType
	if milkType == "" {
		milkType = "formula"
	}

	growthInfo := "No growth measurements available."
	if latestGrowth != nil {
		growthInfo = fmt.Sprintf("Latest measurement (date: %s): weight %.2f kg, length %.1f cm.",
			latestGrowth.Date.Format("2006-01-02"), latestGrowth.WeightKg, latestGrowth.LengthCm)
		if latestGrowth.HeadCircumferenceCm != nil {
			growthInfo += fmt.Sprintf(" Head circumference: %.1f cm.", *latestGrowth.HeadCircumferenceCm)
		}

		// Add WHO context
		wp := CalculateWeightPercentile(gender, ageWeeks, latestGrowth.WeightKg)
		lp := CalculateLengthPercentile(gender, ageWeeks, latestGrowth.LengthCm)
		growthInfo += fmt.Sprintf(" Estimated weight percentile: P%d (z-score: %.2f). Estimated length percentile: P%d (z-score: %.2f).",
			wp.Percentile, wp.ZScore, lp.Percentile, lp.ZScore)
	}

	prompt := fmt.Sprintf(`You are a pediatric health expert. Analyze the following baby data and provide personalized insights.

Baby Information:
- Age: %d weeks old (~%d months), born %s
- Gender: %s
- Milk type: %s
- %s
- Average daily feeding: %d ml (last 7 days)
- Average daily sleep: %d minutes (last 7 days)

Return ONLY valid JSON (no markdown, no code fences) with this exact structure:
{
  "growth_assessment": {
    "percentile": 50,
    "status": "on-track|concern|remarkable",
    "reasoning": "Explanation of growth assessment"
  },
  "feeding_analysis": {
    "daily_avg_ml": %d,
    "recommended_ml": 800,
    "milk_type_guidance": "Guidance based on milk type",
    "recommendations": ["Recommendation 1", "Recommendation 2"]
  },
  "sleep_analysis": {
    "daily_avg_minutes": %d,
    "recommended_minutes": 840,
    "pattern_observations": "Observations about sleep patterns"
  },
  "activities": [
    {
      "name": "Activity name",
      "icon": "single emoji",
      "instructions": "How to do this activity",
      "benefits": "What it develops",
      "duration": "5-10 minutes",
      "difficulty": "easy|medium|advanced"
    }
  ],
  "alerts": [
    {
      "severity": "info|warning|urgent",
      "message": "Alert message",
      "action": "Recommended action"
    }
  ],
  "summary": "Overall assessment paragraph"
}

Important guidelines:
- Base growth assessment on WHO standards for the baby's age and gender
- Set percentile based on actual measurements if available
- Feeding recommendations should match %s milk type and age
- Sleep recommendations should be age-appropriate (newborns need 14-17h, 4-11mo need 12-15h, 1-2y need 11-14h)
- Include 3-5 personalized activities appropriate for the baby's developmental stage
- Only include alerts if there are genuine concerns (e.g., weight below P3, insufficient feeding)
- Keep the summary encouraging and supportive
- This is informational only, not medical advice
- status should be "on-track" for normal development, "concern" for potential issues, "remarkable" for above-average metrics`,
		ageWeeks, ageMonths, profile.DateOfBirth.Format("2006-01-02"),
		gender, milkType, growthInfo,
		feedingAvgML, sleepAvgMin,
		feedingAvgML, sleepAvgMin, milkType)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: map[string]interface{}{
			"temperature":      0.4,
			"maxOutputTokens":  4096,
			"responseMimeType": "application/json",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.Model, g.APIKey)

	log.Printf("Calling Gemini API for personalized insights (model: %s)", g.Model)

	const maxJSONRetries = 2
	var lastJSONErr error

	for jsonAttempt := 0; jsonAttempt <= maxJSONRetries; jsonAttempt++ {
		if jsonAttempt > 0 {
			log.Printf("Retrying Gemini insight call due to invalid JSON (attempt %d/%d): %v", jsonAttempt, maxJSONRetries, lastJSONErr)
			time.Sleep(time.Duration(jsonAttempt) * time.Second)
		}

		body, err := g.callWithRetry(url, jsonBody)
		if err != nil {
			return "", err
		}

		var geminiResp geminiResponse
		if err := json.Unmarshal(body, &geminiResp); err != nil {
			lastJSONErr = fmt.Errorf("failed to parse Gemini response: %w", err)
			continue
		}

		if geminiResp.Error != nil {
			return "", fmt.Errorf("Gemini API error: %s", geminiResp.Error.Message)
		}

		if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
			lastJSONErr = fmt.Errorf("Gemini returned empty response")
			continue
		}

		text := geminiResp.Candidates[0].Content.Parts[0].Text
		text = strings.TrimSpace(text)
		if strings.HasPrefix(text, "```json") {
			text = strings.TrimPrefix(text, "```json")
			text = strings.TrimSuffix(text, "```")
			text = strings.TrimSpace(text)
		} else if strings.HasPrefix(text, "```") {
			text = strings.TrimPrefix(text, "```")
			text = strings.TrimSuffix(text, "```")
			text = strings.TrimSpace(text)
		}

		var js json.RawMessage
		if err := json.Unmarshal([]byte(text), &js); err != nil {
			n := len(text)
			if n > 200 {
				n = 200
			}
			lastJSONErr = fmt.Errorf("Gemini returned invalid JSON: %w\nRaw: %s", err, text[:n])
			continue
		}

		return text, nil
	}

	return "", fmt.Errorf("Gemini returned invalid JSON after %d attempts: %w", maxJSONRetries+1, lastJSONErr)
}
