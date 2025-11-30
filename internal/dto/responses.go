package dto

type BookResponse struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	AuthorID    int    `json:"author_id"`
	AuthorName  string `json:"author_name"`
	Year        int    `json:"year"`
	IsAvailable bool   `json:"is_available"`
}

type AuthorResponse struct {
	AuthorID  int    `json:"author_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Biography string `json:"biography"`
}
