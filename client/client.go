package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	// Клиент 1
	go func() {
		defer wg.Done()
		conn, err := net.Dial("tcp", "localhost:5000")
		if err != nil {
			fmt.Printf("Клиент 1: ошибка подключения: %v\n", err)
			return
		}
		defer conn.Close()

		addTestBook(conn, "Клиент 1")
	}()

	// Клиент 2
	go func() {
		defer wg.Done()
		conn, err := net.Dial("tcp", "localhost:5000")
		if err != nil {
			fmt.Printf("Клиент 2: ошибка подключения: %v\n", err)
			return
		}
		defer conn.Close()

		addTestBook(conn, "Клиент 2")
	}()

	wg.Wait()
	fmt.Println("Оба клиента завершили работу")
}
func addTestBook(conn net.Conn, clientName string) {
	currentYear := time.Now().Year()
	book := []string{
		"1", // Create
		"1", // Ввести книгу
		fmt.Sprintf("Тестовая книга от %s %s", clientName, time.Now().Format("150405")),
		"Автор ",
		"Жанр, ЖанрН",
		fmt.Sprintf("%d", currentYear-1),
		"150",
		"200",
		"твердый",
		"покупка",
		time.Now().Format("02-01-2006"),
		"",
		"8/10 - Хорошая книга",
		"д", // Подтверждение отправляется автоматически
	}

	// Сначала отправляем все данные
	for _, line := range book {
		fmt.Printf("[%s] Отправка: %s\n", clientName, line)
		fmt.Fprintf(conn, line+"\n")
		time.Sleep(100 * time.Millisecond)
	}

	// Затем ждем ответа от сервера
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		response := scanner.Text()
		fmt.Printf("[%s] Ответ сервера: %s\n", clientName, response)
		if strings.Contains(response, "успешно") { // Или другой маркер завершения
			break
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("[%s] Ошибка чтения: %v\n", clientName, err)
	}

	fmt.Printf("[%s] Завершил работу\n", clientName)
}
