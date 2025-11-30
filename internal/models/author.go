package models

import "fmt"

type Person struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Author struct {
	Person
	Biography string `json:"biography"`
	AuthorID  int    `json:"author_id"`
}

func (p Person) String() string {
	return fmt.Sprintf("%s (%s)", p.Name, p.Email)
}

func (aut Author) String() string {
	return fmt.Sprintf("Имя: %s, Биография: %s, ID автора: %d", aut.Name, aut.Biography, aut.AuthorID)
}
