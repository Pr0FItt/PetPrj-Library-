package dto

type CreateBookRequest struct {
	Title    string `json:"title" binding:"required"`
	AuthorID int    `json:"author_id" binding:"required"`
	Year     int    `json:"year" binding:"required"`
}

type SearchBookRequest struct {
	Title string `form:"title"`
	Year  int    `form:"year"`
}

type UpdateBookRequest struct {
	Title    *string `json:"title"`
	AuthorID *int    `json:"author_id"`
	Year     *int    `json:"year"`
}

type CreateAuthorRequest struct {
	Name      string `json:"name" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Biography string `json:"biography"`
}

type ReserveBookRequest struct {
	UserEmail string `json:"user_email" binding:"required"`
	Days      int    `json:"days" binding:"required"`
}
