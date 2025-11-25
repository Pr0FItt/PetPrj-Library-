package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type ReservationSystem struct {
	ReservationQueue chan Reservation
	RWorkers int
	RWG sync.WaitGroup
}

func NewReservationSystem(rWorkers int) *ReservationSystem {
	rs := &ReservationSystem{
		ReservationQueue: make(chan Reservation),
		RWorkers: rWorkers,
	}

	for i := 0; i < rs.RWorkers; i++ {
		rs.RWG.Add(1)
		go rs.reservationWorker(1)
	}

	return rs
}

func (rs *ReservationSystem) reservationWorker(id int) {
	defer rs.RWG.Done()

	for reserv := range rs.ReservationQueue {
		fmt.Printf("Book is reserved for worker %d, ID in queue", reserv.ID)
	}
	fmt.Printf("Reservation Worker #%d stopped\n", id)
}

type Reservation struct {
	ID        int
	BookID    int
	UserEmail string
	StartDate time.Time
	EndDate   time.Time
	Status    string //"active", "completed", "cancelled"
}

func (lib *Library) ReserveBook(bookID int, userEmail string, days int) error {
	book := lib.FindBook(bookID)
	if book == nil {
		return errors.New("Book not found\n")
	}

	if !book.IsAvailable {
		return errors.New("Book is not available\n")
	}

	reservation := Reservation{
		ID:        lib.ReservationID,
		BookID:    bookID,
		UserEmail: userEmail,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(0, 0, days),
		Status:    "active",
	}

	select {
	case lib.Reservations.ReservationQueue <- reservation:
		fmt.Printf("Book is reserved for worker, ID -> %d in queue\n", reservation.ID)
	default:
		fmt.Printf("Reserbation queue is full\n")
	}

	return nil

}

type NotificationSystem struct {
	EmailQueue chan EmailNotification
	Workers    int
	WG         sync.WaitGroup
}

type EmailNotification struct {
	To      string
	Subject string
	Message string
}

func NewNotificationSystem(workers int) *NotificationSystem {
	ns := &NotificationSystem{
		EmailQueue: make(chan EmailNotification, 100),
		Workers:    workers,
	}

	for i := 0; i < workers; i++ {
		ns.WG.Add(1)
		go ns.emailWorker(i)
	}

	return ns
}

func (ns *NotificationSystem) emailWorker(id int) {
	defer ns.WG.Done()

	for email := range ns.EmailQueue {
		fmt.Printf("Email Worker #%d sends email: %s\n", id, email.Subject)

		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Email Worker #%d sent email: %s\n", id, email.To)
	}
	fmt.Printf("Email Worker #%d stopped\n", id)
}

func (ns *NotificationSystem) EmailShutdown() {
	close(ns.EmailQueue)
	ns.WG.Wait()
	fmt.Printf("NotificationSystem Shutdown Complete\n")
}

func (lib *Library) SendEmail(bookID int, userEmail string) {
	book := lib.FindBook(bookID)
	if book == nil {
		return
	}

	author := lib.FindAuthor(book.AuthorID)
	authorName := "unknown"
	if author != nil {
		authorName = author.Name
	}

	notification := EmailNotification{
		To:      userEmail,
		Subject: fmt.Sprintf("Book taken"),
		Message: fmt.Sprintf("Book \"%s\" (author: %s) was gived", book.Title, authorName),
	}

	select {
	case lib.Notifications.EmailQueue <- notification:
		fmt.Printf("Notification queued for email")
	default:
		fmt.Printf("Queue is full")
	}
}

func (lib *Library) SendReturnEmail(bookID int, userEmail string) {
	book := lib.FindBook(bookID)
	if book == nil {
		return
	}

	notification := EmailNotification{
		To:      userEmail,
		Subject: fmt.Sprintf("Book returned"),
		Message: fmt.Sprintf("Book %s returned in library", book.Title),
	}

	select {
	case lib.Notifications.EmailQueue <- notification:
		fmt.Printf("Notification of return queued for email")
	default:
		fmt.Printf("Queue is full")
	}

}

type Person struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (p Person) String() string {
	return fmt.Sprintf("%s (%s)", p.Name, p.Email)
}

type Author struct {
	Person
	Biography string
	AuthorID  int
}

func (aut Author) String() string {
	return fmt.Sprintf("Name: %s, Biography: %s, AuthorID: %d", aut.Name, aut.Biography, aut.AuthorID)
}

func (lib *Library) AddAuthor(name, email, biography string) int {
	author := Author{
		Person: Person{
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

func (lib Library) FindAuthor(id int) *Author {
	for i := range lib.Authors {
		if i == lib.Authors[i].AuthorID {
			return &lib.Authors[i]
		}
	}
	return nil
}

type Book struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	AuthorID    int    `json:"author_id"`
	Year        int    `json:"year"`
	IsAvailable bool   `json:"is_available"`
}

func (b *Book) Borrow(lib *Library, userEmail string) error {
	if !b.IsAvailable {
		return fmt.Errorf("Book is not available")
	}
	b.IsAvailable = false
	fmt.Println("Book %s gived", b.Title)

	go lib.SendEmail(b.ID, userEmail)

	return nil
}

func (b *Book) Return(lib *Library, userEmail string) {
	b.IsAvailable = true
	fmt.Println("Book %s returned", b.Title)

	go lib.SendReturnEmail(b.ID, userEmail)
}

func (b Book) String() string {
	status := "доступна"
	if !b.IsAvailable {
		status = "выдана"
	}
	return fmt.Sprintf("\"%s\" - %d (%d) [%s]", b.Title, b.AuthorID, b.Year, status)
}

type Library struct {
	Books         []Book
	Authors       []Author
	NextIDBook    int
	NextIDAuthor  int
	ReservationID int
	Notifications *NotificationSystem
	Reservations *ReservationSystem
}

func (lib *Library) AddBook(title string, authorID int, year int) bool {
	authorExists := false
	for _, author := range lib.Authors {
		if author.AuthorID == authorID {
			authorExists = true
			break
		}
	}
	if !authorExists {
		fmt.Printf("Author with ID %d not found\n", authorID)
		return false
	}
	book := Book{
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

func (lib Library) FindBook(id int) *Book {
	if len(lib.Books) == 0 {
		return nil
	}

	for i := range lib.Books {
		if lib.Books[i].ID == id {
			return &lib.Books[i]
		}
		println("FindBook")
	}
	return nil
}

func (lib Library) ListAllBooks() {
	fmt.Println("\nВсе книги в библиотеке:")
	if len(lib.Books) == 0 {
		fmt.Println("  Библиотека пуста")
		return
	}
	for _, book := range lib.Books {
		fmt.Printf("  %s\n", book)
	}
}

func (lib Library) ListAuthors() {
	fmt.Println("\nАвторы в библиотеке:")
	if len(lib.Authors) == 0 {
		fmt.Println("  Нет авторов")
		return
	}
	for _, author := range lib.Authors {
		fmt.Printf("  %s\n", author)
	}
}

func (lib Library) ListAvailableBooks() {
	for _, book := range lib.Books {
		if book.IsAvailable {
			fmt.Println(book)
		}
	}
}

func NewLibrary() *Library {
	return &Library{
		Books:         []Book{},
		Authors:       []Author{},
		NextIDBook:    1,
		NextIDAuthor:  1,
		ReservationID: 1,
		Notifications: NewNotificationSystem(3),
		Reservations: NewReservationSystem(3),

	}
}

func (lib *Library) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(lib, "", " ")
	if err != nil {
		return fmt.Errorf("Error marshalling library: %v", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("Error writing library: %v", err)
	}

	return nil
}

func LoadFromFile(filename string) (*Library, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("File reading error: %v", err)
	}

	var library Library
	err = json.Unmarshal(data, &library)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling library: %v", err)
	}

	return &library, nil
}

func main() {
	library := NewLibrary()

	// Добавляем авторов и запоминаем их ID
	author1ID := library.AddAuthor("Лев Толстой", "tolstoy@mail.ru", "Русский писатель")
	author2ID := library.AddAuthor("Фёдор Достоевский", "dostoevsky@mail.ru", "Русский писатель")

	// Добавляем книги
	library.AddBook("Война и мир", author1ID, 1869)
	library.AddBook("Анна Каренина", author1ID, 1877)
	library.AddBook("Преступление и наказание", author2ID, 1866)
	library.AddBook("Братья Карамазовы", author2ID, 1880)

	// Пытаемся добавить книгу несуществующему автору
	library.AddBook("Несуществующая книга", 999, 2000)

	// Показываем всех авторов и книги
	library.ListAuthors()
	library.ListAllBooks()

	// Работа с книгами
	book := library.FindBook(1)
	if book != nil {
		book.Borrow(library, "tols@mail.ru")
	}

	library.ListAvailableBooks()

	// Сохранение и загрузка
	err := library.SaveToFile("library.json")
	if err != nil {
		fmt.Printf("Ошибка сохранения: %v\n", err)
	} else {
		fmt.Println("Библиотека сохранена в library.json")
	}

	loadedLibrary, err := LoadFromFile("library.json")
	if err != nil {
		fmt.Printf("Ошибка загрузки: %v\n", err)
	} else {
		fmt.Println("\nЗагруженная библиотека:")
		loadedLibrary.ListAuthors()
		loadedLibrary.ListAllBooks()
	}
}
