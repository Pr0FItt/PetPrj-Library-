package services

import (
	"fmt"
	"library-app/internal/models"
	"sync"
	"time"
)

type ReservationService struct {
	ReservationQueue chan *models.Reservation
	RWorkers         int
	RWG              sync.WaitGroup
}

func NewReservationService(rWorkers int) *ReservationService {
	rs := &ReservationService{
		ReservationQueue: make(chan *models.Reservation, 100),
		RWorkers:         rWorkers,
	}

	for i := 0; i < rs.RWorkers; i++ {
		rs.RWG.Add(1)
		go rs.reservationWorker(i)
	}

	return rs
}

func (rs *ReservationService) reservationWorker(id int) {
	defer rs.RWG.Done()

	for reserv := range rs.ReservationQueue {
		fmt.Printf("Работник %d в процессе резервации ID: %d\n", id, reserv.ID)
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Книга зарезервирована работником %d, ID очереди %d\n", id, reserv.ID)
	}
	fmt.Printf("Работник резервации #%d остановлен\n", id)
}

func (lib *Library) ReserveBook(bookID int, userEmail string, days int) error {
	lib.mu.Lock()

	book := lib.findBookUnsafe(bookID)
	if book == nil {
		lib.mu.Unlock()
		return fmt.Errorf("книга не найдена")
	}

	if !book.IsAvailable {
		lib.mu.Unlock()
		return fmt.Errorf("книга не доступна")
	}

	if lib.getUserActiveReservationsUnsafe(userEmail) >= 3 {
		lib.mu.Unlock()
		return fmt.Errorf("пользователь достиг лимита резервации (3 активные резервации)")
	}

	reservation := &models.Reservation{
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
	lib.mu.Lock()
	defer lib.mu.Unlock()

	for i, reservation := range lib.ReservationsSlice {
		if reservation.ID == reservationID {
			bookID := reservation.BookID

			lib.ReservationsSlice = append(lib.ReservationsSlice[:i], lib.ReservationsSlice[i+1:]...)

			if book := lib.findBookUnsafe(bookID); book != nil {
				book.IsAvailable = true
			}

			fmt.Printf("Бронь #%d отменена\n", reservationID)
			return nil
		}
	}
	return fmt.Errorf("бронь не найдена")
}

func (lib *Library) GetUserReservation(userEmail string) []*models.Reservation {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	var resers []*models.Reservation
	for _, reservation := range lib.ReservationsSlice {
		if reservation.UserEmail == userEmail {
			resers = append(resers, reservation)
		}
	}

	if len(resers) == 0 {
		return nil
	}
	return resers
}

func (lib *Library) ProcessExpiredReservations() {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for i := 0; i < len(lib.ReservationsSlice); i++ {
		reservation := lib.ReservationsSlice[i]

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
				lib.ProcessExpiredReservations()
			}
		}
	}()
}

func (lib *Library) SendExpirationNotification(reservationID int, userEmail string) {
	notification := &models.EmailNotification{
		To:      userEmail,
		Subject: "Бронь просрочена",
		Message: fmt.Sprintf("Ваша бронь #%d автоматически отменена due to expiration", reservationID),
	}

	select {
	case lib.Notifications.EmailQueue <- notification:
		fmt.Printf("Уведомление о просрочке отправлено\n")
	default:
		fmt.Printf("Очередь переполнена\n")
	}
}
