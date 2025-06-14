package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var regexSchema = map[string]string{
	"name":    `^[А-Яа-яЁёA-Za-z0-9\s]{1,100}$`,
	"authors": `^[А-Яа-яЁёA-Za-z\s,]{1,130}$`,
	"genres":  `^[А-Яа-яЁёA-Za-z\s,]{1,100}$`,
	"year":    `^\d{4}$`,
	"width":   `^\d+(\.\d+)?$`,
	"height":  `^\d+(\.\d+)?$`,
	"cover":   `^(мягкий|твердый)$`,
	"source":  `^(покупка|подарок|наследство)$`,
	"added":   `^\d{2}-\d{2}-\d{4}$`,
	"read":    `^\d{2}-\d{2}-\d{4}$`,
	"rating":  `^([1-9]|10)/10 - [А-Яа-яЁёA-Za-z0-9\s\,\.\!\?]{1,200}$`,
}
var (
	fileMutex sync.Mutex
)

const (
	FILENAME = "books"
)

func ValidateRegex(field, value string) error {
	pattern, ok := regexSchema[field]
	if !ok {
		return fmt.Errorf("некорректное поле: %s", field)
	}

	re := regexp.MustCompile(pattern)
	if !re.MatchString(value) {
		return fmt.Errorf("неверный формат для поля %s", pattern)
	}
	return nil
}

func ValidateYear(year string) error {
	if err := ValidateRegex("year", year); err != nil {
		return err
	}

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		return err
	}

	currentYear := time.Now().Year()
	if yearInt > currentYear {
		return errors.New("год не может быть больше текущего")
	}
	if yearInt < 1500 {
		return errors.New("год не может быть меньше 1500")
	}
	return nil
}

func ValidateHeightWidth(value, heightOrWidth string) error {
	field := "width"
	if heightOrWidth == "height" {
		field = "height"
	}

	if err := ValidateRegex(field, value); err != nil {
		return err
	}

	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}

	if val > 1000 {
		return fmt.Errorf("%s обложка не может быть больше метра", heightOrWidth)
	}
	if val <= 0 {
		return fmt.Errorf("%s обложка может быть только положительной", heightOrWidth)
	}
	return nil
}

func ValidateAdded(added, year string) error {
	if err := ValidateRegex("added", added); err != nil {
		return err
	}

	addedDate, err := time.Parse("02-01-2006", added)
	if err != nil {
		return err
	}

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		return err
	}

	if addedDate.After(time.Now()) {
		return errors.New("дата добавления не может быть в будущем")
	}
	if addedDate.Year() < yearInt {
		return errors.New("дата добавления не может быть раньше даты издания")
	}
	return nil
}

func ValidateRead(read, added string) error {
	if read == "" {
		return nil
	}

	if err := ValidateRegex("read", read); err != nil {
		return err
	}

	readDate, err := time.Parse("02-01-2006", read)
	if err != nil {
		return err
	}

	addedDate, err := time.Parse("02-01-2006", added)
	if err != nil {
		return err
	}

	if readDate.Before(addedDate) {
		return errors.New("дата чтения не может быть до даты добавления")
	}
	return nil
}

func ValidateRating(rating string) error {
	if rating == "" {
		return nil
	}
	if err := ValidateRegex("rating", rating); err != nil {
		return fmt.Errorf("рейтинг должен быть в формате 'X/10 - комментарий' (например: '8/10 - отличная книга')")
	}
	return nil
}

func ValidateCover(bookType string) error {
	if err := ValidateRegex("cover", bookType); err != nil {
		return fmt.Errorf("тип обложки должен быть 'мягкий' или 'твердый'")
	}
	return nil
}

func ValidateSource(source string) error {
	if err := ValidateRegex("source", source); err != nil {
		return fmt.Errorf("источник должен быть: 'покупка', 'подарок' или 'наследство'")
	}
	return nil
}

func ValidateName(name string) error {
	if strings.Contains(name, "  ") {
		return fmt.Errorf("название не должно содержать двойных пробелов")
	}

	if err := ValidateRegex("name", name); err != nil {
		return fmt.Errorf("название может содержать только буквы, цифры и пробелы")
	}

	if len(name) < 1 || len(name) > 100 {
		return fmt.Errorf("название должно быть от 1 до 100 символов")
	}

	return nil
}
func normalizeCommas(input string) string {
	// Заменяем " , " или любые пробелы вокруг запятых на ", "
	re := regexp.MustCompile(`\s*,\s*`)
	return re.ReplaceAllString(input, ",")
}

func ValidateAuthors(authors string) (string, error) {
	normalized := normalizeCommas(authors)
	if strings.Contains(normalized, "  ") || strings.Contains(normalized, ",,") {
		return "", fmt.Errorf("авторы не должны содержать двойных пробелов или запятых")
	}

	if err := ValidateRegex("authors", normalized); err != nil {
		return "", fmt.Errorf("авторы могут содержать только буквы, пробелы и запятые")
	}

	return normalized, nil
}

func ValidateGenres(genres string) (string, error) {
	normalized := normalizeCommas(genres)
	if strings.Contains(normalized, "  ") || strings.Contains(normalized, ",,") {
		return "", fmt.Errorf("жанры не должны содержать двойных пробелов или запятых")
	}

	if err := ValidateRegex("genres", normalized); err != nil {
		return "", fmt.Errorf("жанры могут содержать только буквы, пробелы и запятые")
	}

	return normalized, nil
}

func isUniqueBook(book Book) (bool, error) {
	if _, err := os.Stat(FILENAME); os.IsNotExist(err) {
		return true, nil // файла не существует
	}

	file, err := os.Open(FILENAME)
	if err != nil {
		return false, fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		existingBook, err := lineToDict(line)
		if err != nil {
			return false, fmt.Errorf("ошибка парсинга строки: %v", err)
		}

		if existingBook["name"] == book.Name && existingBook["authors"] == book.Authors {
			return false, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("ошибка чтения файла: %v", err)
	}

	return true, nil
}

func appendBookToFile(book Book) error {
	file, err := os.OpenFile(FILENAME, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	line := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\n",
		book.ID,
		book.Name,
		book.Year,
		book.Authors,
		book.Genres,
		book.Width,
		book.Height,
		book.Cover,
		book.Source,
		book.Added,
		book.Read,
		book.Rating,
	)

	if _, err := file.WriteString(line); err != nil {
		return fmt.Errorf("ошибка записи в файл: %v", err)
	}

	return nil
}

const port = ":5000"

type Book struct {
	ID      string
	Name    string
	Authors string
	Genres  string
	Year    string
	Width   string
	Height  string
	Cover   string
	Source  string
	Added   string
	Read    string
	Rating  string
}

func displayMenu() string {
	return `
 -------------------
| Выберите действие |
 -------------------
0 - Меню
1 - Create
2 - Read
3 - Search
4 - Delete
5 - Update
exit - Выйти
`
}

func displayCreateMenu() string {
	return `
 ------------------
| Добавление книги |
 ------------------
1/
|---- 0 - Меню
|---- 1 - Ввести книгу
|---- exit - Назад
`
}

func lineToDict(line string) (map[string]string, error) {
	parts := strings.Split(strings.TrimSpace(line), "|")

	if len(parts) < 12 {
		return nil, fmt.Errorf("недостаточно частей в строке (ожидается 12, получено %d)", len(parts))
	}

	return map[string]string{
		"id":         parts[0],
		"name":       parts[1],
		"year":       parts[2],
		"authors":    parts[3],
		"genres":     parts[4],
		"width":      parts[5],
		"height":     parts[6],
		"book_type":  parts[7],
		"source":     parts[8],
		"date_added": parts[9],
		"date_read":  parts[10],
		"rating":     parts[11],
	}, nil
}
func getNextID() (int, error) {
	// если файл существует
	if _, err := os.Stat(FILENAME); os.IsNotExist(err) {
		return 1, nil
	}

	// открыть
	file, err := os.Open(FILENAME)
	if err != nil {
		return 0, fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	// найти последнюю строку
	scanner := bufio.NewScanner(file)
	var lastLine string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lastLine = line
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("ошибка чтения файла: %v", err)
	}

	// если пустой
	if lastLine == "" {
		return 1, nil
	}

	// парсим последнюю
	book, err := lineToDict(lastLine)
	if err != nil {
		return 0, fmt.Errorf("ошибка парсинга строки: %v", err)
	}

	// if = int
	id, err := strconv.Atoi(book["id"])
	if err != nil {
		return 0, fmt.Errorf("неверный формат ID: %v", err)
	}

	return id + 1, nil
}

func Create(book Book) string {

	// следующий ID
	bookID, err := getNextID()
	if err != nil {
		return fmt.Sprintf("Ошибка при получении ID: %v", err)
	}
	book.ID = strconv.Itoa(bookID)

	// Проверка на уникальность
	if isUnique, err := isUniqueBook(book); err != nil {
		return fmt.Sprintf("Ошибка проверки уникальности: %v", err)
	} else if !isUnique {
		return fmt.Sprintf("Книга уже добавлена: %s, написанная %s", book.Name, book.Authors)
	}

	// Добавить в файл
	if err := appendBookToFile(book); err != nil {
		return fmt.Sprintf("Ошибка при записи в файл: %v", err)
	}

	return fmt.Sprintf("Добавлена книга: %s (ID: %d)", book.Name, bookID)

}
func Read() ([]Book, error) {
	if _, err := os.Stat(FILENAME); os.IsNotExist(err) {
		return nil, nil // Файла нет - возвращаем пустой список
	}

	file, err := os.Open(FILENAME)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	var books []Book
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		bookMap, err := lineToDict(line)
		if err != nil {
			return nil, fmt.Errorf("ошибка парсинга строки: %v", err)
		}

		book := Book{
			ID:      bookMap["id"],
			Name:    bookMap["name"],
			Year:    bookMap["year"],
			Authors: bookMap["authors"],
			Genres:  bookMap["genres"],
			Width:   bookMap["width"],
			Height:  bookMap["height"],
			Cover:   bookMap["book_type"],
			Source:  bookMap["source"],
			Added:   bookMap["date_added"],
			Read:    bookMap["date_read"],
			Rating:  bookMap["rating"],
		}
		books = append(books, book)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %v", err)
	}

	return books, nil
}

func formatBookList(books []Book) string {
	if len(books) == 0 {
		return "Список книг пуст"
	}

	var builder strings.Builder
	builder.WriteString("\nСписок книг:\n")
	builder.WriteString(strings.Repeat("-", 50) + "\n")

	for _, book := range books {
		builder.WriteString(fmt.Sprintf(
			"ID: %s\nНазвание: %s\nАвторы: %s\nГод: %s\nЖанры: %s\n"+
				"Размер: %sx%s мм\nТип обложки: %s\nИсточник: %s\n"+
				"Добавлена: %s\nПрочитана: %s\nРейтинг: %s\n"+
				strings.Repeat("-", 50)+"\n",
			book.ID, book.Name, book.Authors, book.Year, book.Genres,
			book.Width, book.Height, book.Cover, book.Source,
			book.Added, book.Read, book.Rating))
	}

	builder.WriteString(fmt.Sprintf("Всего книг: %d\n", len(books)))
	return builder.String()

}
func displayReadMenu() string {
	return `
 ---------------
| Просмотр книг |
 ---------------
2/
|---- 0 - Меню
|---- 1 - Вывести книги
|---- exit -  Назад
`
}
func displaySearchMenu() string {
	return `
 ---------------
| Просмотр книг |
 ---------------
3/
|---- 0 - Меню
|---- 1 - Найти книги
|---- exit -  Назад
`
}
func displayFilterMenu() string {
	return `
 -------------
| Найти книги |
 -------------
---- */
|---- ---- 0 - Меню
|---- ---- 1 - По полю 'id'
|---- ---- 2 - По полю 'name'
|---- ---- 3 - По полю 'year'
|---- ---- 4 - По полю 'authors'
|---- ---- 5 - По полю 'genres'
|---- ---- 6 - По полю 'width'
|---- ---- 7 - По полю 'height'
|---- ---- 8 - По полю 'cover'
|---- ---- 9 - По полю 'source'
|---- ---- 10 - По полю 'added'
|---- ---- 11 - По полю 'read'
|---- ---- 12 - По полю 'rating'
|---- ---- exit -  Назад
`
}

func (b *Book) getField(field string) string {
	switch field {
	case "id":
		return b.ID
	case "name":
		return b.Name
	case "year":
		return b.Year
	case "authors":
		return b.Authors
	case "genres":
		return b.Genres
	case "width":
		return b.Width
	case "height":
		return b.Height
	case "cover":
		return b.Cover
	case "source":
		return b.Source
	case "added":
		return b.Added
	case "read":
		return b.Read
	case "rating":
		return b.Rating
	default:
		return ""
	}
}

// обновить поле
func (b *Book) setField(field string, value string) error {
	switch field {
	case "id":
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("ID должен быть числом")
		}
		b.ID = value
	case "name":
		if err := ValidateName(value); err != nil {
			return err
		}
		b.Name = value
	case "year":
		if err := ValidateYear(value); err != nil {
			return err
		}
		b.Year = value
	case "authors":
		normalized, err := ValidateAuthors(value)
		if err != nil {
			return err
		}
		b.Authors = normalized
	case "genres":
		normalized, err := ValidateGenres(value)
		if err != nil {
			return err
		}
		b.Genres = normalized
	case "width":
		if err := ValidateHeightWidth(value, "width"); err != nil {
			return err
		}
		b.Width = value
	case "height":
		if err := ValidateHeightWidth(value, "height"); err != nil {
			return err
		}
		b.Height = value
	case "cover":
		if err := ValidateCover(value); err != nil {
			return err
		}
		b.Cover = value
	case "source":
		if err := ValidateSource(value); err != nil {
			return err
		}
		b.Source = value
	case "added":
		if err := ValidateAdded(value, b.Year); err != nil {
			return err
		}
		b.Added = value
	case "read":
		if value != "" {
			if err := ValidateRead(value, b.Added); err != nil {
				return err
			}
		}
		b.Read = value
	case "rating":
		if value != "" {
			if err := ValidateRating(value); err != nil {
				return err
			}
		}
		b.Rating = value
	default:
		return fmt.Errorf("неизвестное поле: %s", field)
	}
	return nil
}

// обновление - удаление и редактирование
func modifyBooksFile(books []Book, update bool) string {

	tempFilename := "temp_books.txt"
	found := false
	var result strings.Builder

	// книги для обновления/удаления
	bookMap := make(map[string]Book)
	for _, book := range books {
		bookMap[book.ID] = book
	}

	// открытие файлов
	originalFile, err := os.Open(FILENAME)
	if err != nil {
		return fmt.Sprintf("Ошибка открытия файла: %v", err)
	}
	defer originalFile.Close()

	tempFile, err := os.Create(tempFilename)
	if err != nil {
		return fmt.Sprintf("Ошибка создания временного файла: %v", err)
	}
	defer tempFile.Close()

	scanner := bufio.NewScanner(originalFile)
	for scanner.Scan() {
		line := scanner.Text()
		bookData, err := lineToDict(line)
		if err != nil {
			return fmt.Sprintf("Ошибка парсинга строки: %v", err)
		}

		bookID := bookData["id"]
		if bookToModify, exists := bookMap[bookID]; exists {
			found = true
			if update {
				// обновление книги
				newLine := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
					bookToModify.ID,
					bookToModify.Name,
					bookToModify.Year,
					bookToModify.Authors,
					bookToModify.Genres,
					bookToModify.Width,
					bookToModify.Height,
					bookToModify.Cover,
					bookToModify.Source,
					bookToModify.Added,
					bookToModify.Read,
					bookToModify.Rating)

				if _, err := tempFile.WriteString(newLine + "\n"); err != nil {
					return fmt.Sprintf("Ошибка записи во временный файл: %v", err)
				}
				result.WriteString(fmt.Sprintf("Обновлена книга: %s (ID: %s)\n", bookToModify.Name, bookToModify.ID))
			}
			// пропускаем строку
			if !update {
				result.WriteString(fmt.Sprintf("Удалена книга: %s (ID: %s)\n", bookData["name"], bookID))
			}
		} else {
			// записываем не найденные строки
			if _, err := tempFile.WriteString(line + "\n"); err != nil {
				return fmt.Sprintf("Ошибка записи во временный файл: %v", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Sprintf("Ошибка чтения файла: %v", err)
	}

	if !found {
		os.Remove(tempFilename)
		return "Книги не найдены для изменения"
	}

	// Заменим файл обратно
	originalFile.Close()
	if err := os.Remove(FILENAME); err != nil {
		return fmt.Sprintf("Ошибка удаления оригинального файла: %v", err)
	}
	if err := os.Rename(tempFilename, FILENAME); err != nil {
		return fmt.Sprintf("Ошибка переименования временного файла: %v", err)
	}

	return result.String()
}
func searchBooks(field, value string) ([]Book, error) {
	var results []Book

	// Проверим, существует ли файл
	if _, err := os.Stat(FILENAME); os.IsNotExist(err) {
		return results, nil
	}

	file, err := os.Open(FILENAME)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		bookMap, err := lineToDict(line)
		if err != nil {
			return nil, fmt.Errorf("ошибка парсинга строки: %v", err)
		}

		// Сопоставление внешних имен полей с внутренними
		internalField := field
		if field == "cover" {
			internalField = "book_type"
		}

		valueBook, exists := bookMap[internalField]
		if !exists {
			continue
		}

		book := Book{
			ID:      bookMap["id"],
			Name:    bookMap["name"],
			Year:    bookMap["year"],
			Authors: bookMap["authors"],
			Genres:  bookMap["genres"],
			Width:   bookMap["width"],
			Height:  bookMap["height"],
			Cover:   bookMap["book_type"],
			Source:  bookMap["source"],
			Added:   bookMap["date_added"],
			Read:    bookMap["date_read"],
			Rating:  bookMap["rating"],
		}

		if field == "id" {
			if valueBook == value {
				return []Book{book}, nil
			}
		} else if contains([]string{"year", "width", "height"}, field) {
			if valueBook == value {
				results = append(results, book)
			}
		} else {
			if strings.Contains(strings.ToLower(valueBook), strings.ToLower(value)) {
				results = append(results, book)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %v", err)
	}

	return results, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
func displayUpdateMenu() string {
	return `
 ---------------
| Обновить книги |
 ---------------
5/
|---- 0 - Меню
|---- 1 - Обновить книги
|---- exit -  Назад
`
}

func displayDeleteMenu() string {
	return `
---------------
| Удалить книги |
 ---------------
4/
|---- 0 - Меню
|---- 1 - Удалить книги
|---- exit -  Назад
`
}

func main() {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	log.Printf("Сервер слушает на %s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Ошибка соединения: %v", err)
			continue
		}
		go handleClient(conn)
	}
}
func Delete(bookIDs []string) string {
	// Чтение
	books, err := Read()
	if err != nil {
		return fmt.Sprintf("Ошибка при чтении книг: %v", err)
	}

	// Фильтрация
	var booksToDelete []Book
	for _, book := range books {
		for _, id := range bookIDs {
			if book.ID == id {
				booksToDelete = append(booksToDelete, book)
				break
			}
		}
	}

	if len(booksToDelete) == 0 {
		return "Книги не найдены для удаления"
	}

	// Подтверждение
	var builder strings.Builder
	builder.WriteString("Найдены книги для удаления:\n")
	for _, book := range booksToDelete {
		builder.WriteString(fmt.Sprintf("ID: %s, Название: %s, Авторы: %s\n", book.ID, book.Name, book.Authors))
	}
	builder.WriteString("Подтвердите удаление (д/н):")

	return builder.String()
}

func Update(book Book) string {
	// Чтение
	books, err := Read()
	if err != nil {
		return fmt.Sprintf("Ошибка при чтении книг: %v", err)
	}

	// Найти книги для обновления
	var found bool
	for i, b := range books {
		if b.ID == book.ID {
			books[i] = book
			found = true
			break
		}
	}

	if !found {
		return fmt.Sprintf("Книга с ID %s не найдена", book.ID)
	}

	// Записать все книги в файл
	tempFilename := "temp_books.txt"
	tempFile, err := os.Create(tempFilename)
	if err != nil {
		return fmt.Sprintf("Ошибка создания временного файла: %v", err)
	}
	defer tempFile.Close()

	for _, book := range books {
		line := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\n",
			book.ID,
			book.Name,
			book.Year,
			book.Authors,
			book.Genres,
			book.Width,
			book.Height,
			book.Cover,
			book.Source,
			book.Added,
			book.Read,
			book.Rating,
		)
		if _, err := tempFile.WriteString(line); err != nil {
			return fmt.Sprintf("Ошибка записи во временный файл: %v", err)
		}
	}

	// Заменить файл на исходный
	if err := os.Remove(FILENAME); err != nil {
		return fmt.Sprintf("Ошибка удаления оригинального файла: %v", err)
	}
	if err := os.Rename(tempFilename, FILENAME); err != nil {
		return fmt.Sprintf("Ошибка переименования временного файла: %v", err)
	}

	return fmt.Sprintf("Книга с ID %s успешно обновлена", book.ID)
}
func handleClient(conn net.Conn) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr().String()
	log.Printf("Новое соединение: %s", remoteAddr)

	writer := bufio.NewWriter(conn)
	scanner := bufio.NewScanner(conn)

	sendMessage := func(msg string) {
		writer.WriteString(msg + "\n")
		writer.Flush()
	}

	// Отправляем приветствие
	sendMessage("Вы подключились к серверу!")
	sendMessage(displayMenu())

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		log.Printf("%s прислал: %s", remoteAddr, text)
		sendMessage("Вы выбрали действие: " + text)

		switch text {
		case "exit":
			sendMessage("До свидания!")
			log.Printf("Соединение с %s закрыто по команде exit", remoteAddr)
			return
		case "0":
			sendMessage(displayMenu())
		case "1":
			sendMessage(displayCreateMenu())
		createLoop:
			for scanner.Scan() {
				subText := strings.TrimSpace(scanner.Text())
				log.Printf("%s прислал: %s", remoteAddr, subText)
				switch subText {
				case "exit":
					sendMessage("Возврат в главное меню")
					sendMessage(displayMenu())
					break createLoop
				case "0":
					sendMessage(displayCreateMenu())
				case "1":
					var book Book
					var err error
					var input string
					// Name
					sendMessage("Введите название книги:")
					for scanner.Scan() {
						input = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, book.Name)
						if err = ValidateName(input); err == nil {
							book.Name = input
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите название книги:")
					}

					// Authors
					sendMessage("Введите авторов (через запятую):")
					for scanner.Scan() {
						input = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, input)
						normalized, err := ValidateAuthors(input)
						if err == nil {
							book.Authors = normalized
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите авторов (через запятую):")
					}

					// Genres
					sendMessage("Введите жанры (через запятую):")
					for scanner.Scan() {
						input = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, input)
						normalized, err := ValidateGenres(input)
						if err == nil {
							book.Genres = normalized
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите жанры (через запятую):")
					}

					// Year
					sendMessage("Введите год издания:")
					for scanner.Scan() {
						input = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, book.Year)
						if err = ValidateYear(input); err == nil {
							book.Year = input
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите год издания:")
					}

					// Width
					sendMessage("Введите ширину книги (мм):")
					for scanner.Scan() {
						input = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, book.Width)
						if err = ValidateHeightWidth(input, "width"); err == nil {
							book.Width = input
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите ширину книги (мм):")
					}

					// Height
					sendMessage("Введите высоту книги (мм):")
					for scanner.Scan() {
						input = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, input)
						if err = ValidateHeightWidth(input, "height"); err == nil {
							book.Width = input
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите высоту книги (мм):")
					}

					// Book Type
					sendMessage("Введите тип обложки (мягкий/твердый):")
					for scanner.Scan() {
						input = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, input)
						if err = ValidateCover(input); err == nil {
							book.Cover = input
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите тип обложки (мягкий/твердый):")
					}

					// Source
					sendMessage("Введите источник (покупка/подарок/наследство):")
					for scanner.Scan() {
						input = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, input)
						if err = ValidateSource(input); err == nil {
							book.Source = input
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите источник (покупка/подарок/наследство):")
					}

					// Date Added
					sendMessage("Введите дату добавления (ДД-ММ-ГГГГ):")
					for scanner.Scan() {
						input = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, input)
						if err = ValidateAdded(input, book.Year); err == nil {
							book.Added = input
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите дату добавления (ДД-ММ-ГГГГ):")
					}

					// Date Read
					sendMessage("Введите дату прочтения (ДД-ММ-ГГГГ) или оставьте пустым:")
					for scanner.Scan() {
						input = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, input)
						if input == "" {
							break
						}

						if err = ValidateRead(input, book.Added); err == nil {
							book.Read = input
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите дату прочтения (ДД-ММ-ГГГГ) или оставьте пустым:")
					}

					// Rating
					sendMessage("Введите рейтинг (X/10 - комментарий) или оставьте пустым:")
					for scanner.Scan() {
						input = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, input)
						if input == "" {
							break
						}

						if err = ValidateRating(input); err == nil {
							book.Rating = input
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите рейтинг (X/10 - комментарий) или оставьте пустым:")
					}

					// Подтвердить добавление

					sendMessage(fmt.Sprintf(`
ID: %s
Название: %s
Авторы: %s
Жанры: %s
Год: %s
Размер: %sx%s мм
Тип обложки: %s
Источник: %s
Дата добавления: %s
Дата прочтения: %s
Рейтинг: %s
`, book.ID, book.Name, book.Authors, book.Genres, book.Year, book.Width, book.Height,
						book.Cover, book.Source, book.Added, book.Read, book.Rating))
					sendMessage("Добавить книгу? (д/н):")
					for scanner.Scan() {
						confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
						log.Printf("%s прислал: %s", remoteAddr, confirm)
						if confirm == "д" || confirm == "y" {
							sendMessage("Добавление книги... ")
							Create(book)
							sendMessage("Книга добавлена. Отправьте '0' для просмотра меню")
							break
						} else if confirm == "н" || confirm == "n" {
							sendMessage("Добавление отменено. Отправьте '0' для просмотра меню")
							break
						}
					}
				default:
					sendMessage("Неверный выбор в подменю. Попробуйте снова.")
				}
			}
		case "2":
			sendMessage(displayReadMenu())
		readLoop:
			for scanner.Scan() {
				subText := strings.TrimSpace(scanner.Text())
				log.Printf("%s прислал: %s", remoteAddr, subText)
				switch subText {
				case "exit":
					sendMessage("Возврат в главное меню")
					sendMessage(displayMenu())
					break readLoop
				case "0":
					sendMessage(displayReadMenu())
				case "1":
					books, err := Read()
					if err != nil {
						sendMessage("Ошибка при чтении списка книг: " + err.Error())
					} else {
						sendMessage("Вывод всех книг...")
						sendMessage(formatBookList(books))
					}
					sendMessage("Книги выведены. Отправьте '0' для просмотра меню")
				default:
					sendMessage("Неверный выбор в подменю. Попробуйте снова.")
				}
			}
		case "3":
			sendMessage(displaySearchMenu())
		searchLoop:
			for scanner.Scan() {
				subText := strings.TrimSpace(scanner.Text())
				log.Printf("%s прислал: %s", remoteAddr, subText)
				switch subText {
				case "exit":
					sendMessage("Возврат в главное меню")
					sendMessage(displayMenu())
					break searchLoop
				case "0":
					sendMessage(displaySearchMenu())
				case "1":
					sendMessage(displayFilterMenu())
					var field, value string

				filterLoop:
					for scanner.Scan() {
						input := strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, input)
						switch input {
						case "exit":
							sendMessage("Возврат в меню обновления")
							sendMessage(displaySearchMenu())
							break filterLoop
						case "0":
							sendMessage(displayFilterMenu())
						case "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12":
							choice, err := strconv.Atoi(input)
							if err != nil {
								sendMessage("Неверный номер поля")
								continue
							}

							fields := []string{"id", "name", "year", "authors", "genres",
								"width", "height", "cover", "source",
								"added", "read", "rating"}

							if choice < 1 || choice > len(fields) {
								sendMessage("Неверный номер поля")
								continue
							}

							field = fields[choice-1]
							sendMessage(fmt.Sprintf("Введите значение для поиска по полю '%s':", field))

							// Получить значение поиска с валидацией
							for scanner.Scan() {
								value = strings.TrimSpace(scanner.Text())
								if field == "id" || field == "width" || field == "height" {
									if _, err := strconv.Atoi(value); err != nil {
										sendMessage("Должно быть целое число. Попробуйте снова:")
										continue
									}
								}
								break
							}

							// Поиск книг
							books, err := searchBooks(field, value)
							if err != nil {
								sendMessage(fmt.Sprintf("Ошибка поиска: %v", err))
								continue filterLoop
							}

							if len(books) == 0 {
								sendMessage("Книги не найдены")
								continue filterLoop
							}

							sendMessage("Найдены книги:")
							sendMessage(formatBookList(books))

						default:
							sendMessage("Неверный выбор в подменю. Попробуйте снова.")
						}
					}
				default:
					sendMessage("Неверный выбор в подменю. Попробуйте снова.")
				}
			}
		case "4": // Delete
			sendMessage(displayDeleteMenu())
		deleteLoop:
			for scanner.Scan() {
				subText := strings.TrimSpace(scanner.Text())
				log.Printf("%s прислал: %s", remoteAddr, subText)
				switch subText {
				case "exit":
					sendMessage("Возврат в главное меню")
					sendMessage(displayMenu())
					break deleteLoop
				case "0":
					sendMessage(displayDeleteMenu())
				case "1":
					sendMessage("Введите ID книги для удаления (разделяйте запятыми для нескольких):")
					scanner.Scan()
					idsInput := strings.TrimSpace(scanner.Text())
					bookIDs := strings.Split(idsInput, ",")

					// Убрать пробелы
					for i, id := range bookIDs {
						bookIDs[i] = strings.TrimSpace(id)
					}

					// Запросить подтверждение
					response := Delete(bookIDs)
					sendMessage(response)

					scanner.Scan()
					confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
					if confirm == "д" || confirm == "y" {
						// Удалить
						var booksToDelete []Book
						allBooks, _ := Read()
						for _, book := range allBooks {
							for _, id := range bookIDs {
								if book.ID == id {
									booksToDelete = append(booksToDelete, book)
									break
								}
							}
						}

						result := modifyBooksFile(booksToDelete, false)
						sendMessage(result)
					} else {
						sendMessage("Удаление отменено")
					}
					sendMessage("Отправьте '0' для просмотра меню")
				default:
					sendMessage("Неверный выбор в подменю. Попробуйте снова.")
				}
			}

		case "5": // Update
			sendMessage(displayUpdateMenu())
		updateLoop:
			for scanner.Scan() {
				subText := strings.TrimSpace(scanner.Text())
				log.Printf("%s прислал: %s", remoteAddr, subText)
				switch subText {
				case "exit":
					sendMessage("Возврат в главное меню")
					sendMessage(displayMenu())
					break updateLoop
				case "0":
					sendMessage(displayUpdateMenu())
				case "1":
					// Найти
					sendMessage("Введите ID книги для обновления:")
					scanner.Scan()
					bookID := strings.TrimSpace(scanner.Text())

					books, err := searchBooks("id", bookID)
					if err != nil || len(books) == 0 {
						sendMessage("Книга не найдена")
						continue
					}

					book := books[0]
					sendMessage(fmt.Sprintf("Найдена книга: %s", book.Name))
					sendMessage("Введите новые значения (оставьте пустым, чтобы не изменять)")

					// Обновить каждое поле
					fields := []struct {
						name     string
						prompt   string
						validate func(string) error
					}{
						{"name", "Название книги:", ValidateName},
						{"authors", "Авторы (через запятую):", func(s string) error {
							_, err := ValidateAuthors(s)
							return err
						}},
						{"genres", "Жанры (через запятую):", func(s string) error {
							_, err := ValidateGenres(s)
							return err
						}},
						{"year", "Год издания:", ValidateYear},
						{"width", "Ширина (мм):", func(s string) error {
							return ValidateHeightWidth(s, "width")
						}},
						{"height", "Высота (мм):", func(s string) error {
							return ValidateHeightWidth(s, "height")
						}},
						{"cover", "Тип обложки (мягкий/твердый):", ValidateCover},
						{"source", "Источник (покупка/подарок/наследство):", ValidateSource},
						{"added", "Дата добавления (ДД-ММ-ГГГГ):", func(s string) error {
							return ValidateAdded(s, book.Year)
						}},
						{"read", "Дата прочтения (ДД-ММ-ГГГГ) или пусто:", func(s string) error {
							if s == "" {
								return nil
							}
							return ValidateRead(s, book.Added)
						}},
						{"rating", "Рейтинг (X/10 - комментарий) или пусто:", ValidateRating},
					}

					for _, field := range fields {
						for {
							sendMessage(field.prompt + " (Текущее: " + book.getField(field.name) + ")")
							scanner.Scan()
							value := strings.TrimSpace(scanner.Text())

							// Если ввод пустой, оставляем текущее значение и переходим к следующему полю
							if value == "" {
								break
							}

							// Проверяем валидность ввода
							if err := field.validate(value); err != nil {
								sendMessage("Ошибка валидации: " + err.Error())
								sendMessage("Пожалуйста, введите значение снова")
								continue // Повторяем запрос этого же поля
							}

							// Обработка специальных полей (authors и genres)
							if field.name == "authors" {
								normalized, _ := ValidateAuthors(value)
								book.Authors = normalized
							} else if field.name == "genres" {
								normalized, _ := ValidateGenres(value)
								book.Genres = normalized
							} else {
								book.setField(field.name, value)
							}
							break // Выходим из цикла для этого поля, так как ввод корректен
						}
					}

					sendMessage("Изменения:")
					sendMessage(fmt.Sprintf(`
ID: %s
Название: %s
Авторы: %s
Жанры: %s
Год: %s
Размер: %sx%s мм
Тип обложки: %s
Источник: %s
Дата добавления: %s
Дата прочтения: %s
Рейтинг: %s
`, book.ID, book.Name, book.Authors, book.Genres, book.Year, book.Width, book.Height,
						book.Cover, book.Source, book.Added, book.Read, book.Rating))

					sendMessage("Подтвердите обновление (д/н):")
					scanner.Scan()
					confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
					if confirm == "д" || confirm == "y" {
						result := Update(book)
						sendMessage(result)
					} else {
						sendMessage("Обновление отменено")
					}
					sendMessage("Отправьте '0' для просмотра меню")
				default:
					sendMessage("Неверный выбор в подменю. Попробуйте снова.")
				}
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Ошибка чтения от %s: %v", remoteAddr, err)
		}
		log.Printf("Соединение с %s закрыто", remoteAddr)
	}
}
