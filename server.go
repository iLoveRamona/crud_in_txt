package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
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

	// Check allowed characters and length
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

// createBackup создает бэкап в папке /backups
func createBackup() error {
	if _, err := os.Stat(FILENAME); os.IsNotExist(err) {
		return nil // Файла нет - бэкап не нужен
	}

	// Создаем папку backups если ее нет
	if err := os.MkdirAll(BACKUP_DIR, 0755); err != nil {
		return fmt.Errorf("ошибка создания папки бэкапов: %v", err)
	}

	// Формируем имя файла с timestamp
	backupName := fmt.Sprintf("%s/%s_%s.bak",
		BACKUP_DIR,
		FILENAME,
		time.Now().Format("20060102_150405"))

	// Читаем исходный файл
	input, err := os.ReadFile(FILENAME)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла для бэкапа: %v", err)
	}

	// Пишем бэкап
	err = os.WriteFile(backupName, input, 0644)
	if err != nil {
		return fmt.Errorf("ошибка создания бэкапа: %v", err)
	}

	return nil
}

// cleanupBackups удаляет старые бэкапы, оставляя последние 5
func cleanupBackups() error {
	files, err := os.ReadDir(BACKUP_DIR)
	if err != nil {
		return fmt.Errorf("ошибка чтения папки бэкапов: %v", err)
	}

	// Сортируем по времени изменения (новые сначала)
	sort.Slice(files, func(i, j int) bool {
		infoI, _ := files[i].Info()
		infoJ, _ := files[j].Info()
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Удаляем все кроме 5 последних
	for i := 5; i < len(files); i++ {
		err := os.Remove(filepath.Join(BACKUP_DIR, files[i].Name()))
		if err != nil {
			return fmt.Errorf("ошибка удаления старого бэкапа: %v", err)
		}
	}

	return nil
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

const port = ":8080"

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
3 - Update
4 - Delete
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

const (
	FILENAME   = "books"
	BACKUP_DIR = "backups"
)

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
	// бекап
	if err := createBackup(); err != nil {
		return fmt.Sprintf("Ошибка при создании бэкапа: %v", err)
	}

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

func displayUpdateMenu() string {
	return `
 ---------------
| Обновить книги |
 ---------------
3/
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
					// Name
					sendMessage("Введите название книги:")
					for scanner.Scan() {
						book.Name = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, book.Name)
						if err = ValidateName(book.Name); err == nil {
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите название книги:")
					}

					// Authors
					sendMessage("Введите авторов (через запятую):")
					for scanner.Scan() {
						input := strings.TrimSpace(scanner.Text())
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
						input := strings.TrimSpace(scanner.Text())
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
						book.Year = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, book.Year)
						if err = ValidateYear(book.Year); err == nil {
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите год издания:")
					}

					// Width
					sendMessage("Введите ширину книги (мм):")
					for scanner.Scan() {
						book.Width = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, book.Width)
						if err = ValidateHeightWidth(book.Width, "width"); err == nil {
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите ширину книги (мм):")
					}

					// Height
					sendMessage("Введите высоту книги (мм):")
					for scanner.Scan() {
						book.Height = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, book.Height)
						if err = ValidateHeightWidth(book.Height, "height"); err == nil {
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите высоту книги (мм):")
					}

					// Book Type
					sendMessage("Введите тип обложки (мягкий/твердый):")
					for scanner.Scan() {
						book.Cover = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, book.Cover)
						if err = ValidateCover(book.Cover); err == nil {
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите тип обложки (мягкий/твердый):")
					}

					// Source
					sendMessage("Введите источник (покупка/подарок/наследство):")
					for scanner.Scan() {
						book.Source = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, book.Source)
						if err = ValidateSource(book.Source); err == nil {
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите источник (покупка/подарок/наследство):")
					}

					// Date Added
					sendMessage("Введите дату добавления (ДД-ММ-ГГГГ):")
					for scanner.Scan() {
						book.Added = strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, book.Added)
						if err = ValidateAdded(book.Added, book.Year); err == nil {
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите дату добавления (ДД-ММ-ГГГГ):")
					}

					// Date Read
					sendMessage("Введите дату прочтения (ДД-ММ-ГГГГ) или оставьте пустым:")
					for scanner.Scan() {
						input := strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, input)
						if input == "" {
							break
						}
						book.Read = input
						if err = ValidateRead(book.Read, book.Added); err == nil {
							break
						}
						sendMessage("Неверный ввод: " + err.Error())
						sendMessage("Введите дату прочтения (ДД-ММ-ГГГГ) или оставьте пустым:")
					}

					// Rating
					sendMessage("Введите рейтинг (X/10 - комментарий) или оставьте пустым:")
					for scanner.Scan() {
						input := strings.TrimSpace(scanner.Text())
						log.Printf("%s прислал: %s", remoteAddr, input)
						if input == "" {
							break
						}
						book.Rating = input
						if err = ValidateRating(book.Rating); err == nil {
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
							// Here you would typically save the book to your storage
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
					sendMessage("Вывод всех книг...")
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
					sendMessage("Обновление книг...")
					sendMessage("Книги обновлены. Отправьте '0' для просмотра меню")
				default:
					sendMessage("Неверный выбор в подменю. Попробуйте снова.")
				}
			}
		case "4":
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
					sendMessage("Удаление книг...")
					sendMessage("Книги удалены. Отправьте '0' для просмотра меню")
				default:
					sendMessage("Неверный выбор в подменю. Попробуйте снова.")
				}
			}
		default:
			sendMessage("Неверный выбор. Попробуйте снова.")
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Ошибка чтения от %s: %v", remoteAddr, err)
	}
	log.Printf("Соединение с %s закрыто", remoteAddr)
}
