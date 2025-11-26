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
	RWorkers         int
	RWG              sync.WaitGroup
}

func NewReservationSystem(rWorkers int) *ReservationSystem {
	rs := &ReservationSystem{
		ReservationQueue: make(chan Reservation),
		RWorkers:         rWorkers,
	}

	for i := 0; i < rs.RWorkers; i++ {
		rs.RWG.Add(1)
		go rs.reservationWorker(i)
	}

	return rs
}

func (rs *ReservationSystem) reservationWorker(id int) {
	defer rs.RWG.Done()

	for reserv := range rs.ReservationQueue {
		fmt.Printf("Работник %d в процессе резервации ID: %d\n", id, reserv.ID)
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Книга зарезервирована работником %d, ID очереди %d\n", id, reserv.ID)
	}
	fmt.Printf("Работник резервации #%d остановлен\n", id)
}

type Reservation struct {
	ID        int
	BookID    int
	UserEmail string
	StartDate time.Time
	EndDate   time.Time
	Status    string //"active", "completed", "cancelled"
}

func (lib *Library) findBookUnsafe(id int) *Book {
	for i := range lib.Books {
		if lib.Books[i].ID == id {
			return &lib.Books[i]
		}
	}
	return nil
}

func (lib *Library) ReserveBook(bookID int, userEmail string, days int) error {
	lib.mu.Lock()

	book := lib.findBookUnsafe(bookID)
	if book == nil {
		lib.mu.Unlock()
		return errors.New("Книга не найдена\n")
	}

	if !book.IsAvailable {
		lib.mu.Unlock()
		return errors.New("Книга не доступна\n")
	}

	if lib.getUserActiveReservationsUnsafe(userEmail) >= 3 {
		lib.mu.Unlock()
		return errors.New("Пользователь достиг лимита резервации (3 активные резервации)\n")
	}

	reservation := Reservation{
		ID:        lib.ReservationID,
		BookID:    bookID,
		UserEmail: userEmail,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(0, 0, days),
		Status:    "active",
	}

	lib.ReservationsSlice = append(lib.ReservationsSlice, reservation)
	book.IsAvailable = false
	lib.ReservationID++

	lib.mu.Unlock()

	select {
	case lib.Reservations.ReservationQueue <- reservation:
		fmt.Printf("Книга зарезервирована работником, ID -> %d в очереди\n", reservation.ID)
	default:
		fmt.Printf("Очередь резервации переполнена\n")
	}

	return nil
}

func (lib *Library) getUserActiveReservationsUnsafe(userEmail string) int {
	count := 0
	for _, reservation := range lib.ReservationsSlice {
		if reservation.UserEmail == userEmail && reservation.Status == "active" {
			count++
		}
	}
	return count
}

func (lib *Library) CancelReservation(reservationID int) error {
	for i := range lib.ReservationsSlice {
		if lib.ReservationsSlice[i].ID == reservationID {
			bookID := lib.ReservationsSlice[i].BookID
			lib.ReservationsSlice = append(lib.ReservationsSlice[:i], lib.ReservationsSlice[i+1:]...)

			if book := lib.findBookUnsafe(bookID); book != nil {
				book.IsAvailable = true
			}

			fmt.Printf("Бронь #%d отменена\n", reservationID)
			return nil
		}
	}
	return errors.New("Бронь не найдена\n")
}

func (lib *Library) GetUserReservation(userEmail string) []Reservation {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	resers := []Reservation{}
	for _, reservation := range lib.ReservationsSlice {
		if reservation.UserEmail == userEmail {
			resers = append(resers, reservation)
		}
	}

	if len(resers) == 0 {
		return nil
	} else {
		return resers
	}
}

func (lib *Library) ProccessExpiredReservations() {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for i := 0; i < len(lib.ReservationsSlice); i++ {
		reservation := &lib.ReservationsSlice[i]

		if reservation.Status == "active" && reservation.EndDate.Before(now) {
			fmt.Printf("Бронь %d просрочена для пользователя %s\n", reservation.ID, reservation.UserEmail)

			reservation.Status = "expired"

			if book := lib.findBookUnsafe(reservation.BookID); book != nil {
				book.IsAvailable = true
				fmt.Printf("Книга %s снова доступна\n", book.Title)
			}

			expiredCount++

			go lib.SendExpirationNotification(reservation.ID, reservation.UserEmail)
		}

	}

	if expiredCount > 0 {
		fmt.Printf("Обработано просроченных броней: %d\n", expiredCount)
	}

}

func (lib *Library) StartExpirationChecker() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				lib.ProccessExpiredReservations()
			}
		}
	}()
}

func (lib *Library) SendExpirationNotification(reservationID int, userEmail string) {
	notification := EmailNotification{
		To:      userEmail,
		Subject: "Бронь просрочена",
		Message: fmt.Sprintf("Ваша бронь #%d автоматические отменена due to expiration", reservationID),
	}

	select {
	case lib.Notifications.EmailQueue <- notification:
		fmt.Printf("Уведомление о просрочке было отправлено\n")
	default:
		fmt.Printf("Очередь переполнена")
	}
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
		fmt.Printf("Почтовый работник #%d отправляет email: %s\n", id, email.Subject)

		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Почтовый работник #%d отправил email: %s\n", id, email.To)
	}
	fmt.Printf("Почтовый работник #%d остановлен\n", id)
}

func (ns *NotificationSystem) EmailShutdown() {
	close(ns.EmailQueue)
	ns.WG.Wait()
	fmt.Printf("Завершение системы уведомлений успешно завершено\n")
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
		Subject: fmt.Sprintf("Книга взята"),
		Message: fmt.Sprintf("Книга \"%s\" (автор: %s) была получена", book.Title, authorName),
	}

	select {
	case lib.Notifications.EmailQueue <- notification:
		fmt.Printf("Уведомление поставлена в очередь")
	default:
		fmt.Printf("Очередь уведомлений переполнена")
	}
}

func (lib *Library) SendReturnEmail(bookID int, userEmail string) {
	book := lib.FindBook(bookID)
	if book == nil {
		return
	}

	notification := EmailNotification{
		To:      userEmail,
		Subject: fmt.Sprintf("Книга возвращена"),
		Message: fmt.Sprintf("Книга %s возвращена в библиотеку", book.Title),
	}

	select {
	case lib.Notifications.EmailQueue <- notification:
		fmt.Printf("Уведомление о возвращении поставлено в очередь")
	default:
		fmt.Printf("Очередь уведомлений переполнена")
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
	return fmt.Sprintf("Имя: %s, Биография: %s, ID автора: %d", aut.Name, aut.Biography, aut.AuthorID)
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
		if lib.Authors[i].AuthorID == id {
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
		return fmt.Errorf("Книга недоступна\n")
	}
	b.IsAvailable = false
	fmt.Printf("Книга %s получена\n\n", b.Title)

	go lib.SendEmail(b.ID, userEmail)

	return nil
}

func (b *Book) Return(lib *Library, userEmail string) {
	b.IsAvailable = true
	fmt.Printf("Книга %s возвращена\n\n", b.Title)

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
	mu                sync.Mutex
	Books             []Book
	Authors           []Author
	ReservationsSlice []Reservation
	NextIDBook        int
	NextIDAuthor      int
	ReservationID     int
	Notifications     *NotificationSystem
	Reservations      *ReservationSystem
}

func (lib *Library) AddBook(title string, authorID int, year int) bool {
	lib.mu.Lock()
	defer lib.mu.Unlock()
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
	lib.mu.Lock()
	defer lib.mu.Unlock()
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
		Reservations:  NewReservationSystem(3),
	}
}

func (lib *Library) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(lib, "", " ")
	if err != nil {
		return fmt.Errorf("Ошибка маршалинга библиотеки: %v\n", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("Ошибка записи библиотеки: %v\n", err)
	}

	return nil
}

func LoadFromFile(filename string) (*Library, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения файла: %v\n", err)
	}

	var library Library
	err = json.Unmarshal(data, &library)
	if err != nil {
		return nil, fmt.Errorf("Ошибка анмаршалинга библиотеки: %v\n", err)
	}

	return &library, nil
}

func main() {
	fmt.Println("=== СИСТЕМА УПРАВЛЕНИЯ БИБЛИОТЕКОЙ ===")

	library := NewLibrary()

	fmt.Println("\n--- Добавляем авторов ---")
	author1ID := library.AddAuthor("Лев Толстой", "tolstoy@mail.ru", "Русский писатель")
	author2ID := library.AddAuthor("Фёдор Достоевский", "dostoevsky@mail.ru", "Русский писатель")
	author3ID := library.AddAuthor("Антон Чехов", "chekhov@mail.ru", "Русский писатель и драматург")

	fmt.Println("\n--- Добавляем книги ---")
	library.AddBook("Война и мир", author1ID, 1869)
	library.AddBook("Анна Каренина", author1ID, 1877)
	library.AddBook("Преступление и наказание", author2ID, 1866)
	library.AddBook("Братья Карамазовы", author2ID, 1880)
	library.AddBook("Вишневый сад", author3ID, 1904)
	library.AddBook("Чайка", author3ID, 1896)

	library.ListAuthors()
	library.ListAllBooks()

	library.StartExpirationChecker()
	fmt.Println("\n--- Запущен обработчик просроченных броней ---")

	fmt.Println("\n=== ДЕМО 1: Конкурентное бронирование ===")

	var wg sync.WaitGroup
	users := []string{"user1@mail.ru", "user2@mail.ru", "user3@mail.ru", "user4@mail.ru", "user5@mail.ru"}

	for i, user := range users {
		wg.Add(1)
		go func(userNum int, email string) {
			defer wg.Done()

			fmt.Printf("Пользователь %s пытается забронировать книгу #1...\n", email)
			err := library.ReserveBook(1, email, 7) // Бронируем на 7 дней
			if err != nil {
				fmt.Printf("❌ Пользователь %s: %v\n", email, err)
			} else {
				fmt.Printf("✅ Пользователь %s: успешное бронирование!\n", email)
			}
		}(i, user)
	}

	wg.Wait()
	fmt.Println("\n--- Результаты бронирования ---")
	library.ListAllBooks()

	fmt.Println("\n=== ДЕМО 2: Работа с бронями ===")

	userReservations := library.GetUserReservation("user1@mail.ru")
	if len(userReservations) > 0 {
		fmt.Printf("Брони пользователя user1@mail.ru:\n")
		for _, res := range userReservations {
			fmt.Printf("  - Бронь #%d, книга #%d, статус: %s\n",
				res.ID, res.BookID, res.Status)
		}

		if len(userReservations) > 0 {
			fmt.Printf("\nОтменяем бронь #%d...\n", userReservations[0].ID)
			err := library.CancelReservation(userReservations[0].ID)
			if err != nil {
				fmt.Printf("Ошибка отмены: %v\n", err)
			} else {
				fmt.Printf("Бронь отменена\n")
			}
		}
	}

	fmt.Println("\n=== ДЕМО 3: Бронирование других книг ===")

	fmt.Printf("user1@mail.ru пытается забронировать книгу #3...\n")
	err := library.ReserveBook(3, "user1@mail.ru", 5)
	if err != nil {
		fmt.Printf("Ошибка: %v\n", err)
	} else {
		fmt.Printf("Успешное бронирование!\n")
	}

	fmt.Println("\n=== ДЕМО 4: Проверка лимита бронирований ===")

	usersForLimit := []string{"user2@mail.ru", "user2@mail.ru", "user2@mail.ru", "user2@mail.ru"}
	booksForLimit := []int{2, 4, 5, 6}

	for i := 0; i < len(usersForLimit); i++ {
		fmt.Printf("user2@mail.ru пытается забронировать книгу #%d...\n", booksForLimit[i])
		err := library.ReserveBook(booksForLimit[i], usersForLimit[i], 3)
		if err != nil {
			fmt.Printf("Ошибка: %v\n", err)
		} else {
			fmt.Printf("Успешное бронирование!\n")
		}
	}

	fmt.Println("\n--- Финальное состояние библиотеки ---")
	library.ListAllBooks()

	fmt.Println("\n=== ДЕМО 5: Имитация просроченных броней ===")

	library.mu.Lock()
	if len(library.ReservationsSlice) > 0 {
		library.ReservationsSlice[0].EndDate = time.Now().Add(-24 * time.Hour)
		fmt.Printf("Изменили дату брони #%d на прошедшую\n", library.ReservationsSlice[0].ID)
	}
	library.mu.Unlock()

	fmt.Printf("Запускаем обработку просроченных броней...\n")
	library.ProccessExpiredReservations()

	fmt.Println("\n--- Состояние после обработки просроченных ---")
	library.ListAllBooks()

	fmt.Println("\n--- Завершение работы ---")
	time.Sleep(2 * time.Second)

	library.Notifications.EmailShutdown()
	fmt.Println("Программа завершена!")
}
