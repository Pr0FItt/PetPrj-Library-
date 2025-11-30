package handlers

import (
	"github.com/gin-gonic/gin"
	"library-app/internal/dto"
	"library-app/internal/models"
	"library-app/internal/services"
	"strconv"
)

func SetupRouter(library *services.Library) *gin.Engine {
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Books endpoints
	books := router.Group("/books")
	{
		books.GET("/", func(c *gin.Context) {
			books := library.GetAllBooks()
			c.JSON(200, gin.H{
				"success": true,
				"data":    books,
				"count":   len(books),
			})
		})

		books.GET("/:id", func(c *gin.Context) {
			idStr := c.Param("id")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "Неверный ID книги"})
				return
			}

			book := library.FindBook(id)
			if book == nil {
				c.JSON(404, gin.H{"error": "Книга не найдена"})
				return
			}

			c.JSON(200, gin.H{"data": book})
		})

		// ИСПРАВЛЕННЫЙ ПОИСК - убрал :search из пути
		books.GET("/search/advanced", func(c *gin.Context) {
			var req dto.SearchBookRequest
			if err := c.ShouldBindQuery(&req); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			if req.Title == "" && req.Year == 0 {
				c.JSON(400, gin.H{"error": "Необходим хотя бы один параметр для поиска"})
				return // ДОБАВИЛ return
			}

			books := library.AdvancedSearchBooks(req.Title, req.Year)
			if len(books) == 0 {
				c.JSON(404, gin.H{"error": "Книги по заданным критериям не найдены"})
				return
			}
			c.JSON(200, gin.H{
				"success": true,
				"data":    books,
				"count":   len(books),
			})
		})

		books.POST("/", func(c *gin.Context) {
			var req dto.CreateBookRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Неверные данные: " + err.Error()})
				return
			}

			success := library.AddBook(req.Title, req.AuthorID, req.Year)
			if !success {
				c.JSON(400, gin.H{"error": "Не удалось добавить книгу. Проверьте ID автора."})
				return
			}

			c.JSON(201, gin.H{"message": "Книга успешно добавлена"})
		})

		books.POST("/:id/reserve", func(c *gin.Context) {
			idStr := c.Param("id")
			bookID, err := strconv.Atoi(idStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "Неверный ID книги"})
				return
			}

			var req dto.ReserveBookRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Неверные данные: " + err.Error()})
				return
			}

			err = library.ReserveBook(bookID, req.UserEmail, req.Days)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"message": "Книга успешно забронирована"})
		})

		books.POST("/:id/return", func(c *gin.Context) {
			idStr := c.Param("id")
			bookID, err := strconv.Atoi(idStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "Неверный ID книги"})
				return
			}

			var req struct {
				UserEmail string `json:"user_email" binding:"required"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Неверные данные: " + err.Error()})
				return
			}

			err = library.ReturnBook(bookID, req.UserEmail)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"message": "Книга успешно возвращена"})
		})

		books.PUT("/:id/update", func(c *gin.Context) {
			idStr := c.Param("id")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "Неверный ID книги"})
				return
			}

			var req dto.UpdateBookRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Неверные данные: " + err.Error()})
				return
			}

			err = library.UpdateBook(id, req)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"message": "Книга успешно обновлена"})
		})

		books.DELETE("/:id/delete", func(c *gin.Context) {
			idStr := c.Param("id")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "Неверный ID книги"})
				return
			}

			book := library.FindBook(id)
			if book == nil {
				c.JSON(404, gin.H{"error": "Книга не найдена"})
				return
			}

			err = library.DeleteBook(id)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"message": "Книга успешно удалена"})
		})
	}

	// Authors endpoints (аналогично обновить)
	authors := router.Group("/authors")
	{
		authors.GET("/", func(c *gin.Context) {
			authors := library.GetAllAuthors()
			c.JSON(200, gin.H{
				"success": true,
				"data":    authors,
				"count":   len(authors),
			})
		})

		authors.POST("/", func(c *gin.Context) {
			var req dto.CreateAuthorRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Неверные данные: " + err.Error()})
				return
			}

			authorID := library.AddAuthor(req.Name, req.Email, req.Biography)
			c.JSON(201, gin.H{
				"message":   "Автор успешно добавлен",
				"author_id": authorID,
			})
		})

		authors.GET("/:id/books", func(c *gin.Context) {
			idStr := c.Param("id")
			authorID, err := strconv.Atoi(idStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "Неверный ID автора"})
				return
			}

			authorBooks := library.GetBooksByAuthor(authorID)
			c.JSON(200, gin.H{
				"success": true,
				"data":    authorBooks,
				"count":   len(authorBooks),
			})
		})
	}

	// Reservations endpoints
	reservations := router.Group("/reservations")
	{
		reservations.GET("/", func(c *gin.Context) {
			userEmail := c.Query("user_email")
			if userEmail == "" {
				c.JSON(400, gin.H{"error": "Необходим параметр user_email"})
				return
			}

			userReservations := library.GetUserReservation(userEmail)
			if userReservations == nil {
				userReservations = []*models.Reservation{}
			}

			c.JSON(200, gin.H{
				"success": true,
				"data":    userReservations,
				"count":   len(userReservations),
			})
		})

		reservations.POST("/:id/cancel", func(c *gin.Context) {
			idStr := c.Param("id")
			reservationID, err := strconv.Atoi(idStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "Неверный ID бронирования"})
				return
			}

			err = library.CancelReservation(reservationID)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"message": "Бронь успешно отменена"})
		})
	}

	// Search endpoint
	router.GET("/search/books", func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(400, gin.H{"error": "Необходим поисковый запрос (q)"})
			return
		}

		results := library.SearchBooks(query)
		c.JSON(200, gin.H{
			"success": true,
			"data":    results,
			"count":   len(results),
		})
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"success": true,
			"message": "Библиотека API работает",
			"version": "1.0.0",
		})
	})

	return router
}
