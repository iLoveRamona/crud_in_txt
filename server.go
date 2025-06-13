package main

import (
	"bufio"
	"log"
	"net"
	"strings"
)

const port = ":8080"

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
				switch subText {
				case "exit":
					sendMessage("Возврат в главное меню")
					sendMessage(displayMenu())
					break createLoop
				case "0":
					sendMessage(displayCreateMenu())
				case "1":
					sendMessage("Добавление книги... ")
					sendMessage("Книга добавлена. Отправьте '0' для возвращения в меню")
				default:
					sendMessage("Неверный выбор в подменю. Попробуйте снова.")
				}
			}
		case "2":
			sendMessage(displayReadMenu())
		readLoop:
			for scanner.Scan() {
				subText := strings.TrimSpace(scanner.Text())
				switch subText {
				case "exit":
					sendMessage("Возврат в главное меню")
					sendMessage(displayMenu())
					break readLoop
				case "0":
					sendMessage(displayReadMenu())
				case "1":
					sendMessage("Вывод всех книг...")
					sendMessage("Книги выведены. Отправьте '0' для возвращения в меню")
				default:
					sendMessage("Неверный выбор в подменю. Попробуйте снова.")
				}
			}
		case "3":
			sendMessage(displayUpdateMenu())
		updateLoop:
			for scanner.Scan() {
				subText := strings.TrimSpace(scanner.Text())
				switch subText {
				case "exit":
					sendMessage("Возврат в главное меню")
					sendMessage(displayMenu())
					break updateLoop
				case "0":
					sendMessage(displayUpdateMenu())
				case "1":
					sendMessage("Обновление книг...")
					sendMessage("Книги обновлены. Отправьте '0' для возвращения в меню")
				default:
					sendMessage("Неверный выбор в подменю. Попробуйте снова.")
				}
			}
		case "4":
			sendMessage(displayDeleteMenu())
		deleteLoop:
			for scanner.Scan() {
				subText := strings.TrimSpace(scanner.Text())
				switch subText {
				case "exit":
					sendMessage("Возврат в главное меню")
					sendMessage(displayMenu())
					break deleteLoop
				case "0":
					sendMessage(displayDeleteMenu())
				case "1":
					sendMessage("Удаление книг...")
					sendMessage("Книги удалены. Отправьте '0' для возвращения в меню")
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
