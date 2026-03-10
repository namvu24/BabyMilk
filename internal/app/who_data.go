package app

import "math"

// WHO Child Growth Standards (simplified) — Weight-for-age and Length-for-age
// Data for 0–24 months, boys and girls. Values are monthly (0–24).
// Source: WHO Child Growth Standards (2006)

// WHOPercentileData holds percentile values at each month of age.
type WHOPercentileData struct {
	Months []int     `json:"months"`
	P3     []float64 `json:"p3"`
	P15    []float64 `json:"p15"`
	P50    []float64 `json:"p50"`
	P85    []float64 `json:"p85"`
	P97    []float64 `json:"p97"`
}

// WHOPercentileResult holds the result of a percentile calculation.
type WHOPercentileResult struct {
	Percentile int     `json:"percentile"`
	ZScore     float64 `json:"z_score"`
}

// ── Weight-for-age data (kg) ──

var boysWeightForAge = WHOPercentileData{
	Months: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
	P3:     []float64{2.5, 3.4, 4.3, 5.0, 5.6, 6.0, 6.4, 6.7, 6.9, 7.1, 7.4, 7.6, 7.7, 7.9, 8.1, 8.3, 8.4, 8.6, 8.8, 8.9, 9.1, 9.2, 9.4, 9.5, 9.7},
	P15:    []float64{2.9, 3.9, 4.9, 5.7, 6.2, 6.7, 7.1, 7.4, 7.7, 7.9, 8.2, 8.4, 8.6, 8.8, 9.0, 9.2, 9.4, 9.6, 9.8, 9.9, 10.1, 10.3, 10.5, 10.6, 10.8},
	P50:    []float64{3.3, 4.5, 5.6, 6.4, 7.0, 7.5, 7.9, 8.3, 8.6, 8.9, 9.2, 9.4, 9.6, 9.9, 10.1, 10.3, 10.5, 10.7, 10.9, 11.1, 11.3, 11.5, 11.8, 12.0, 12.2},
	P85:    []float64{3.9, 5.1, 6.3, 7.2, 7.8, 8.4, 8.8, 9.2, 9.6, 9.9, 10.2, 10.5, 10.8, 11.0, 11.3, 11.5, 11.7, 12.0, 12.2, 12.5, 12.7, 12.9, 13.2, 13.4, 13.6},
	P97:    []float64{4.4, 5.8, 7.1, 8.0, 8.7, 9.3, 9.8, 10.3, 10.7, 11.0, 11.4, 11.7, 12.0, 12.3, 12.6, 12.8, 13.1, 13.4, 13.7, 13.9, 14.2, 14.5, 14.7, 15.0, 15.3},
}

var girlsWeightForAge = WHOPercentileData{
	Months: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
	P3:     []float64{2.4, 3.2, 3.9, 4.5, 5.0, 5.4, 5.7, 6.0, 6.3, 6.5, 6.7, 6.9, 7.0, 7.2, 7.4, 7.6, 7.7, 7.9, 8.1, 8.2, 8.4, 8.6, 8.7, 8.9, 9.0},
	P15:    []float64{2.8, 3.6, 4.5, 5.2, 5.7, 6.1, 6.5, 6.8, 7.0, 7.3, 7.5, 7.7, 7.9, 8.1, 8.3, 8.5, 8.7, 8.9, 9.1, 9.2, 9.4, 9.6, 9.8, 10.0, 10.2},
	P50:    []float64{3.2, 4.2, 5.1, 5.8, 6.4, 6.9, 7.3, 7.6, 7.9, 8.2, 8.5, 8.7, 8.9, 9.2, 9.4, 9.6, 9.8, 10.0, 10.2, 10.4, 10.6, 10.9, 11.1, 11.3, 11.5},
	P85:    []float64{3.7, 4.8, 5.8, 6.6, 7.3, 7.8, 8.2, 8.6, 9.0, 9.3, 9.6, 9.9, 10.1, 10.4, 10.6, 10.9, 11.1, 11.4, 11.6, 11.8, 12.1, 12.3, 12.5, 12.8, 13.0},
	P97:    []float64{4.2, 5.5, 6.6, 7.5, 8.2, 8.8, 9.3, 9.8, 10.2, 10.5, 10.9, 11.2, 11.5, 11.8, 12.1, 12.4, 12.6, 12.9, 13.2, 13.5, 13.7, 14.0, 14.3, 14.6, 14.8},
}

// ── Length-for-age data (cm) ──

var boysLengthForAge = WHOPercentileData{
	Months: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
	P3:     []float64{46.1, 50.8, 54.4, 57.3, 59.7, 61.7, 63.3, 64.8, 66.2, 67.5, 68.7, 69.9, 71.0, 72.1, 73.1, 74.1, 75.0, 76.0, 76.9, 77.7, 78.6, 79.4, 80.2, 81.0, 81.7},
	P15:    []float64{47.9, 52.8, 56.4, 59.4, 61.8, 63.8, 65.5, 67.0, 68.4, 69.7, 71.0, 72.2, 73.4, 74.5, 75.6, 76.6, 77.6, 78.6, 79.6, 80.5, 81.4, 82.3, 83.1, 83.9, 84.8},
	P50:    []float64{49.9, 54.7, 58.4, 61.4, 63.9, 65.9, 67.6, 69.2, 70.6, 72.0, 73.3, 74.5, 75.7, 76.9, 78.0, 79.1, 80.2, 81.2, 82.3, 83.2, 84.2, 85.1, 86.0, 86.9, 87.8},
	P85:    []float64{51.8, 56.7, 60.4, 63.5, 66.0, 68.0, 69.8, 71.3, 72.8, 74.2, 75.6, 76.9, 78.1, 79.3, 80.5, 81.7, 82.8, 83.9, 84.9, 86.0, 87.0, 88.0, 88.9, 89.9, 90.8},
	P97:    []float64{53.7, 58.6, 62.4, 65.5, 68.0, 70.1, 71.9, 73.5, 75.0, 76.5, 77.9, 79.2, 80.5, 81.8, 83.0, 84.2, 85.4, 86.5, 87.7, 88.8, 89.8, 90.9, 91.9, 92.9, 93.9},
}

var girlsLengthForAge = WHOPercentileData{
	Months: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
	P3:     []float64{45.4, 49.8, 53.0, 55.6, 57.8, 59.6, 61.2, 62.7, 64.0, 65.3, 66.5, 67.7, 68.9, 70.0, 71.0, 72.0, 73.0, 74.0, 74.9, 75.8, 76.7, 77.5, 78.4, 79.2, 80.0},
	P15:    []float64{47.2, 51.7, 55.0, 57.7, 59.9, 61.8, 63.5, 65.0, 66.4, 67.7, 69.0, 70.3, 71.4, 72.6, 73.7, 74.8, 75.8, 76.8, 77.8, 78.8, 79.7, 80.6, 81.5, 82.3, 83.2},
	P50:    []float64{49.1, 53.7, 57.1, 59.8, 62.1, 64.0, 65.7, 67.3, 68.7, 70.1, 71.5, 72.8, 74.0, 75.2, 76.4, 77.5, 78.6, 79.7, 80.7, 81.7, 82.7, 83.7, 84.6, 85.5, 86.4},
	P85:    []float64{51.0, 55.6, 59.1, 62.0, 64.3, 66.2, 68.0, 69.6, 71.1, 72.6, 74.0, 75.3, 76.6, 77.8, 79.1, 80.2, 81.4, 82.5, 83.6, 84.7, 85.7, 86.7, 87.7, 88.7, 89.6},
	P97:    []float64{52.9, 57.6, 61.1, 64.0, 66.4, 68.5, 70.3, 71.9, 73.5, 75.0, 76.4, 77.8, 79.2, 80.5, 81.7, 83.0, 84.2, 85.4, 86.5, 87.6, 88.7, 89.8, 90.8, 91.9, 92.9},
}

// GetWHOData returns the appropriate percentile data for the given gender and metric.
func GetWHOData(gender, metric string) *WHOPercentileData {
	switch metric {
	case "weight":
		if gender == "female" {
			return &girlsWeightForAge
		}
		return &boysWeightForAge
	case "length":
		if gender == "female" {
			return &girlsLengthForAge
		}
		return &boysLengthForAge
	default:
		return nil
	}
}

// CalculateWeightPercentile returns the estimated percentile and z-score for weight-for-age.
func CalculateWeightPercentile(gender string, ageWeeks int, weightKg float64) WHOPercentileResult {
	data := GetWHOData(gender, "weight")
	if data == nil {
		return WHOPercentileResult{}
	}
	return calculatePercentile(data, ageWeeks, weightKg)
}

// CalculateLengthPercentile returns the estimated percentile and z-score for length-for-age.
func CalculateLengthPercentile(gender string, ageWeeks int, lengthCm float64) WHOPercentileResult {
	data := GetWHOData(gender, "length")
	if data == nil {
		return WHOPercentileResult{}
	}
	return calculatePercentile(data, ageWeeks, lengthCm)
}

// calculatePercentile estimates the percentile using linear interpolation between WHO bands.
func calculatePercentile(data *WHOPercentileData, ageWeeks int, value float64) WHOPercentileResult {
	month := ageWeeks / 4 // approximate weeks to months
	if month < 0 {
		month = 0
	}
	if month >= len(data.P50) {
		month = len(data.P50) - 1
	}

	p3 := data.P3[month]
	p15 := data.P15[month]
	p50 := data.P50[month]
	p85 := data.P85[month]
	p97 := data.P97[month]

	// Estimate percentile based on which band the value falls in
	var percentile int
	var zScore float64

	// Use approximate SD: (P97 - P3) / (2 * 1.88) ≈ SD
	sd := (p97 - p3) / 3.76
	if sd > 0 {
		zScore = (value - p50) / sd
	}

	switch {
	case value <= p3:
		percentile = 3
	case value <= p15:
		percentile = 3 + int(float64(15-3)*(value-p3)/(p15-p3))
	case value <= p50:
		percentile = 15 + int(float64(50-15)*(value-p15)/(p50-p15))
	case value <= p85:
		percentile = 50 + int(float64(85-50)*(value-p50)/(p85-p50))
	case value <= p97:
		percentile = 85 + int(float64(97-85)*(value-p85)/(p97-p85))
	default:
		percentile = 97
	}

	return WHOPercentileResult{
		Percentile: percentile,
		ZScore:     math.Round(zScore*100) / 100,
	}
}

// GetWHOCurveData returns percentile curve arrays for chart overlay.
// months parameter limits to 0..months (inclusive).
func GetWHOCurveData(gender, metric string, months int) *WHOPercentileData {
	data := GetWHOData(gender, metric)
	if data == nil {
		return nil
	}
	if months <= 0 || months >= len(data.P50) {
		return data
	}
	// Return a subset
	return &WHOPercentileData{
		Months: data.Months[:months+1],
		P3:     data.P3[:months+1],
		P15:    data.P15[:months+1],
		P50:    data.P50[:months+1],
		P85:    data.P85[:months+1],
		P97:    data.P97[:months+1],
	}
}
