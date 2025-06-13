package main

import (
	"bufio"
	"log"
	"net"
)

const port = ":8080"

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

	// Отправляем приветствие
	conn.Write([]byte("Вы подключились к серверу!\n"))

	// Читаем ввод клиента
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		log.Printf("%s прислал: %s", remoteAddr, text)

		// Эхо-ответ
		conn.Write([]byte("Вы написали: " + text + "\n"))

		// Выход по команде
		if text == "exit" {
			conn.Write([]byte("До свидания!\n"))
			break
		}
	}

	log.Printf("Соединение с %s закрыто", remoteAddr)
}
