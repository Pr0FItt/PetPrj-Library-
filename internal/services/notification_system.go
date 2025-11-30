package services

import (
	"fmt"
	"library-app/internal/models"
	"sync"
	"time"
)

type NotificationService struct {
	EmailQueue chan *models.EmailNotification
	Workers    int
	WG         sync.WaitGroup
}

func NewNotificationService(workers int) *NotificationService {
	ns := &NotificationService{
		EmailQueue: make(chan *models.EmailNotification, 100),
		Workers:    workers,
	}

	for i := 0; i < workers; i++ {
		ns.WG.Add(1)
		go ns.emailWorker(i)
	}

	return ns
}

func (ns *NotificationService) emailWorker(id int) {
	defer ns.WG.Done()

	for email := range ns.EmailQueue {
		fmt.Printf("Почтовый работник #%d отправляет email: %s\n", id, email.Subject)

		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Почтовый работник #%d отправил email: %s\n", id, email.To)
	}
	fmt.Printf("Почтовый работник #%d остановлен\n", id)
}

func (ns *NotificationService) EmailShutdown() {
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

	notification := &models.EmailNotification{
		To:      userEmail,
		Subject: "Книга взята",
		Message: fmt.Sprintf("Книга \"%s\" (автор: %s) была получена", book.Title, authorName),
	}

	select {
	case lib.Notifications.EmailQueue <- notification:
		fmt.Printf("Уведомление поставлено в очередь\n")
	default:
		fmt.Printf("Очередь уведомлений переполнена\n")
	}
}

func (lib *Library) SendReturnEmail(bookID int, userEmail string) {
	book := lib.FindBook(bookID)
	if book == nil {
		return
	}

	notification := &models.EmailNotification{
		To:      userEmail,
		Subject: "Книга возвращена",
		Message: fmt.Sprintf("Книга %s возвращена в библиотеку", book.Title),
	}

	select {
	case lib.Notifications.EmailQueue <- notification:
		fmt.Printf("Уведомление о возвращении поставлено в очередь\n")
	default:
		fmt.Printf("Очередь уведомлений переполнена\n")
	}
}
