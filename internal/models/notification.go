package models

type EmailNotification struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}
