package service

import "encoding/json"

func countRequestImages(body []byte) int {
	var payload any
	if json.Unmarshal(body, &payload) != nil {
		return 0
	}
	return countImageValues(payload)
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
