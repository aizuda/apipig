package service

import "encoding/json"

func countRequestImages(body []byte) int {
	var payload any
	if json.Unmarshal(body, &payload) != nil {
		return 0
	}
	return countImageValues(payload)
}

func countResponseImages(body []byte) int {
	var payload struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &payload) == nil && len(payload.Data) > 0 {
		total := 0
		for _, image := range payload.Data {
			if image.URL != "" || image.B64JSON != "" {
				total++
			}
		}
		if total > 0 {
			return total
		}
	}
	return countRequestImages(body)
}

func countImageValues(value any) int {
	switch item := value.(type) {
	case []any:
		total := 0
		for _, child := range item {
			total += countImageValues(child)
		}
		return total
	case map[string]any:
		total := 0
		if kind, _ := item["type"].(string); kind == "image" || kind == "image_url" || kind == "input_image" || kind == "output_image" {
			total++
		}
		for key, child := range item {
			if key != "type" {
				total += countImageValues(child)
			}
		}
		return total
	default:
		return 0
	}
}
