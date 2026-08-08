package response

import reviewModel "apipig/app/apps/code-review/model"

type ProjectSaveResult struct {
	Project       reviewModel.Project `json:"project"`
	WebhookSecret string              `json:"webhookSecret,omitempty"`
	WebhookURL    string              `json:"webhookUrl"`
}

type PushChannelsResult struct {
	Channels []reviewModel.PushChannelView `json:"channels"`
}

type WebhookAccepted struct {
	TaskID    string `json:"taskId,omitempty"`
	Status    string `json:"status"`
	Duplicate bool   `json:"duplicate"`
}

type Finding struct {
	Severity    string `json:"severity"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

type AIReviewResult struct {
	RiskLevel string    `json:"riskLevel"`
	Summary   string    `json:"summary"`
	Findings  []Finding `json:"findings"`
	Report    string    `json:"report"`
}
