package models

import (
	"fmt"
)

type Book struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	AuthorID    int    `json:"author_id"`
	Year        int    `json:"year"`
	IsAvailable bool   `json:"is_available"`
}

func (b *Book) Borrow() error {
	if !b.IsAvailable {
		return fmt.Errorf("книга недоступна")
	}
	b.IsAvailable = false
	return nil
}

func (b *Book) Return() {
	b.IsAvailable = true
}

func (b Book) String() string {
	status := "доступна"
	if !b.IsAvailable {
		status = "выдана"
	}
	return fmt.Sprintf("\"%s\" (%d) [%s]", b.Title, b.Year, status)
}

func (b Book) Update(newBook Book) {
	b.Year = newBook.Year
	b.IsAvailable = newBook.IsAvailable
	b.ID = newBook.ID
	b.Title = newBook.Title
	b.AuthorID = newBook.AuthorID
}
