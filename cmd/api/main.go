package main

import (
	"fmt"
	"library-app/internal/handlers"
	"library-app/internal/services"
	"log"
)

func main() {
	library := services.NewLibrary()

	author1ID := library.AddAuthor("Лев Толстой", "tolstoy@mail.ru", "Русский писатель")
	author2ID := library.AddAuthor("Фёдор Достоевский", "dostoevsky@mail.ru", "Русский писатель")
	author3ID := library.AddAuthor("Антон Чехов", "chekhov@mail.ru", "Русский писатель и драматург")

	library.AddBook("Война и мир", author1ID, 1869)
	library.AddBook("Анна Каренина", author1ID, 1877)
	library.AddBook("Преступление и наказание", author2ID, 1866)
	library.AddBook("Братья Карамазовы", author2ID, 1880)
	library.AddBook("Вишневый сад", author3ID, 1904)
	library.AddBook("Чайка", author3ID, 1896)

	library.StartExpirationChecker()

	router := handlers.SetupRouter(library)

	fmt.Println("🚀 Сервер библиотеки запущен на http://localhost:8080")
	fmt.Println("📚 Доступные endpoints:")
	fmt.Println("   GET  /health          - Проверка здоровья API")
	fmt.Println("   GET  /books           - Все книги")
	fmt.Println("   GET  /books/:id       - Конкретная книга")
	fmt.Println("   POST /books           - Добавить книгу")
	fmt.Println("   POST /books/:id/reserve - Забронировать книгу")
	fmt.Println("   GET  /authors         - Все авторы")
	fmt.Println("   POST /authors         - Добавить автора")
	fmt.Println("   GET  /reservations    - Брони пользователя (user_email параметр)")
	fmt.Println("   POST /reservations/:id/cancel - Отменить бронь")
	fmt.Println("   GET  /search/books    - Поиск книг")

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
