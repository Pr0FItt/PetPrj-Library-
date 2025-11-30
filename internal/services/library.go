package services

import (
	"fmt"
	"library-app/internal/dto"
	"library-app/internal/models"
	"strings"
	"sync"
)

type Library struct {
	mu                sync.RWMutex
	Books             []*models.Book
	Authors           []*models.Author
	ReservationsSlice []*models.Reservation
	NextIDBook        int
	NextIDAuthor      int
	ReservationID     int
	Notifications     *NotificationService
	Reservations      *ReservationService
}

func NewLibrary() *Library {
	return &Library{
		Books:         []*models.Book{},
		Authors:       []*models.Author{},
		NextIDBook:    1,
		NextIDAuthor:  1,
		ReservationID: 1,
		Notifications: NewNotificationService(3),
		Reservations:  NewReservationService(3),
	}
}

func (lib *Library) AddAuthor(name, email, biography string) int {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	author := &models.Author{
		Person: models.Person{
			Name:  name,
			Email: email,
		},
		Biography: biography,
		AuthorID:  lib.NextIDAuthor,
	}
	lib.Authors = append(lib.Authors, author)
	lib.NextIDAuthor++

	fmt.Printf("Добавлен автор: %s\n", author)
	return author.AuthorID
}

func (lib *Library) FindAuthor(id int) *models.Author {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	for _, author := range lib.Authors {
		if author.AuthorID == id {
			return author
		}
	}
	return nil
}

func (lib *Library) AddBook(title string, authorID int, year int) bool {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	// Проверяем существование автора
	authorExists := false
	for _, author := range lib.Authors {
		if author.AuthorID == authorID {
			authorExists = true
			break
		}
	}

	if !authorExists {
		fmt.Printf("Автор с ID %d не найден\n", authorID)
		return false
	}

	book := &models.Book{
		ID:          lib.NextIDBook,
		Title:       title,
		AuthorID:    authorID,
		Year:        year,
		IsAvailable: true,
	}
	lib.Books = append(lib.Books, book)
	lib.NextIDBook++
	return true
}

func (lib *Library) FindBook(id int) *models.Book {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	for _, book := range lib.Books {
		if book.ID == id {
			return book
		}
	}
	return nil
}

func (lib *Library) AdvancedSearchBooks(title string, year int) []*models.Book {
	lib.mu.RLock()
	defer lib.mu.RUnlock()

	var results []*models.Book

	for _, book := range lib.Books {
		match := true

		// Поиск по названию (если передан)
		if title != "" && !strings.Contains(strings.ToLower(book.Title), strings.ToLower(title)) {
			match = false
		}

		// Поиск по году (если передан)
		if year != 0 && book.Year != year {
			match = false
		}

		if match {
			results = append(results, book)
		}
	}

	return results
}

func (lib *Library) UpdateBook(id int, req dto.UpdateBookRequest) error {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	if req.Title == nil && req.Year == nil && req.AuthorID == nil {
		return fmt.Errorf("Не указаны поля для обновления")
	}

	if req.Year != nil && *req.Year < 0 {
		return fmt.Errorf("Не верный формат ввода года")
	}

	book := lib.findBookUnsafe(id)
	if book == nil {
		return fmt.Errorf("Книга не найдена")
	}

	if req.Title != nil {
		book.Title = *req.Title
	}

	if req.AuthorID != nil {
		authorExists := false
		for _, author := range lib.Authors {
			if author.AuthorID == *req.AuthorID {
				authorExists = true
				break
			}
		}

		if !authorExists {
			return fmt.Errorf("Автор с ID %d не найден", *req.AuthorID)
		}
		book.AuthorID = *req.AuthorID
	}

	if req.Year != nil {
		book.Year = *req.Year
	}

	return nil
}

func (lib *Library) DeleteBook(id int) error {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	for i, book := range lib.Books {
		if book.ID == id {
			lib.Books = append(lib.Books[:i], lib.Books[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("Книга не найдена")
}

func (lib *Library) findBookUnsafe(id int) *models.Book {
	for _, book := range lib.Books {
		if book.ID == id {
			return book
		}
	}
	return nil
}

func (lib *Library) findAuthorUnsafe(id int) *models.Author {
	for _, author := range lib.Authors {
		if author.AuthorID == id {
			return author
		}
	}
	return nil
}

func (lib *Library) ListAllBooks() {
	fmt.Println("\nВсе книги в библиотеке:")
	if len(lib.Books) == 0 {
		fmt.Println("  Библиотека пуста")
		return
	}

	lib.mu.Lock()
	defer lib.mu.Unlock()

	for _, book := range lib.Books {
		fmt.Printf("  %s\n", book)
	}
}

func (lib *Library) ListAuthors() {
	fmt.Println("\nАвторы в библиотеке:")
	if len(lib.Authors) == 0 {
		fmt.Println("  Нет авторов")
		return
	}

	lib.mu.Lock()
	defer lib.mu.Unlock()

	for _, author := range lib.Authors {
		fmt.Printf("  %s\n", author)
	}
}

func (lib *Library) GetAllBooks() []models.Book {
	lib.mu.RLock()
	defer lib.mu.RUnlock()

	books := make([]models.Book, len(lib.Books))
	for i, book := range lib.Books {
		books[i] = *book
	}
	return books
}

func (lib *Library) GetAllAuthors() []models.Author {
	lib.mu.RLock()
	defer lib.mu.RUnlock()

	authors := make([]models.Author, len(lib.Authors))
	for i, author := range lib.Authors {
		authors[i] = *author
	}
	return authors
}

func (lib *Library) GetBooksByAuthor(authorID int) []models.Book {
	lib.mu.RLock()
	defer lib.mu.RUnlock()

	var authorBooks []models.Book
	for _, book := range lib.Books {
		if book.AuthorID == authorID {
			authorBooks = append(authorBooks, *book)
		}
	}
	return authorBooks
}

func (lib *Library) SearchBooks(query string) []models.Book {
	lib.mu.RLock()
	defer lib.mu.RUnlock()

	var results []models.Book
	for _, book := range lib.Books {
		if strings.Contains(book.Title, query) {
			results = append(results, *book)
		}
	}
	return results
}

func (lib *Library) ReturnBook(bookID int, userEmail string) error {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	book := lib.findBookUnsafe(bookID)
	if book == nil {
		return fmt.Errorf("книга не найдена")
	}

	if book.IsAvailable {
		return fmt.Errorf("книга уже доступна")
	}

	book.Return()

	go lib.SendReturnEmail(bookID, userEmail)

	return nil
}
