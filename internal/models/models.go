package models

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
type DeleteResponse struct {
	Data struct {
		ID      string `json:"id"`
		Deleted bool   `json:"deleted"`
	} `json:"data"`
}
