package models

import "time"

type Links struct {
	Admin string `json:"admin,omitempty"`
}

type Category struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FAQCount int    `json:"faq_count"`
	Links    Links  `json:"links,omitempty"`
}

type FAQ struct {
	ID           string            `json:"id"`
	Category     string            `json:"category"`
	CategoryName string            `json:"category_name,omitempty"`
	Question     map[string]string `json:"question"`
	Answer       map[string]string `json:"answer"`
	Links        Links             `json:"links,omitempty"`
}

type PageMeta struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
	LastPage    int `json:"last_page"`
}

type PageLinks struct {
	Self     string `json:"self,omitempty"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
}

type CategoryPage struct {
	Data  []Category `json:"data"`
	Meta  PageMeta   `json:"meta"`
	Links PageLinks  `json:"links"`
}
type FAQPage struct {
	Data  []FAQ     `json:"data"`
	Meta  PageMeta  `json:"meta"`
	Links PageLinks `json:"links"`
}
type CategoryResponse struct {
	Data Category `json:"data"`
}
type FAQResponse struct {
	Data FAQ `json:"data"`
}
type FormOption struct {
	Value  string            `json:"value"`
	Labels map[string]string `json:"labels"`
}

type FormField struct {
	Key       string            `json:"key"`
	Type      string            `json:"type"`
	Required  bool              `json:"required"`
	Skippable bool              `json:"skippable"`
	Active    bool              `json:"active"`
	Labels    map[string]string `json:"labels"`
	Options   []FormOption      `json:"options"`
}

type Form struct {
	Key        string      `json:"key"`
	Type       string      `json:"type"`
	Name       string      `json:"name"`
	Version    int         `json:"version"`
	Active     bool        `json:"active"`
	Fields     []FormField `json:"fields"`
	FieldCount int         `json:"field_count"`
	Links      Links       `json:"links,omitempty"`
}

type FormPage struct {
	Data  []Form    `json:"data"`
	Meta  PageMeta  `json:"meta"`
	Links PageLinks `json:"links"`
}

type FormResponse struct {
	Data Form `json:"data"`
}

type DeleteResponse struct {
	Data struct {
		ID      string `json:"id"`
		Deleted bool   `json:"deleted"`
	} `json:"data"`
}

type AuthUser struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type LoginData struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      AuthUser  `json:"user"`
}

type LoginResponse struct {
	Data LoginData `json:"data"`
}

type AuthStatus struct {
	Authenticated bool      `json:"authenticated"`
	ExpiresAt     time.Time `json:"expires_at"`
	User          AuthUser  `json:"user"`
}

type AuthStatusResponse struct {
	Data AuthStatus `json:"data"`
}
