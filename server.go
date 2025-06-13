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
3 - Search
4 - Update
5 - Delete
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
|---- 1 - Сгенерировать книгу
|---- 2 - Ввести книгу вручную
|---- exit - Назад
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
					sendMessage("Генерация книги... ")
					sendMessage("Книга сгенерирована. Отправьте '0' для возвращения в меню")
				case "2":
					sendMessage("Ручной ввод книги... ")
					sendMessage("Книга добавлена. Отправьте '0' для возвращения в меню")
				default:
					sendMessage("Неверный выбор в подменю. Попробуйте снова.")
				}
			}
		default:
			sendMessage("Неверный выбор. Попробуйте снова.")
			sendMessage(displayCreateMenu())
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Ошибка чтения от %s: %v", remoteAddr, err)
	}
	log.Printf("Соединение с %s закрыто", remoteAddr)
}
