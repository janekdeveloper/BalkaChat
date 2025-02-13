package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type Message struct {
	Type     string `json:"type"`     // Тип сообщения: "search", "message", и т.д.
	Message  string `json:"message"`  // Сообщение пользователя
	Nickname string `json:"nickname"` // Никнейм пользователя
}

var waitingQueue = make(chan *websocket.Conn, 100)          // Очередь для ожидания собеседников
var clients = make(map[*websocket.Conn]string)              // Мапа для хранения соединений с никами
var connections = make(map[*websocket.Conn]*websocket.Conn) // Связанные пары клиентов

func main() {
	app := fiber.New()

	app.Static("/", "./", fiber.Static{
		Index: "index.html",
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		defer func() {
			if partner, exists := connections[c]; exists {
				partner.WriteJSON(Message{Type: "disconnect", Message: "Ваш собеседник отключился."})
				delete(connections, partner)
				partner.Close()
			}
			delete(clients, c)
			delete(connections, c)
			c.Close()
		}()

		for {
			var msg Message
			if err := c.ReadJSON(&msg); err != nil {
				break
			}

			switch msg.Type {
			case "search":
				// Добавляем клиента в очередь ожидания
				clients[c] = msg.Nickname
				waitingQueue <- c
				log.Printf("Client %s is waiting for a partner...\n", msg.Nickname)

				// Если в очереди два клиента, создаем пару
				if len(waitingQueue) >= 2 {
					firstClient := <-waitingQueue
					secondClient := <-waitingQueue

					firstNickname := clients[firstClient]
					secondNickname := clients[secondClient]

					// Соединяем клиентов друг с другом
					connections[firstClient] = secondClient
					connections[secondClient] = firstClient

					// Уведомляем клиентов
					firstClient.WriteJSON(Message{Type: "connected", Message: "You are now connected!", Nickname: secondNickname})
					secondClient.WriteJSON(Message{Type: "connected", Message: "You are now connected!", Nickname: firstNickname})
				}

			case "message":
				// Передача сообщения
				if partner, exists := connections[c]; exists {
					partner.WriteJSON(Message{Type: "message", Message: msg.Message, Nickname: clients[c]})
					// log.Printf("%s", partner)
				}
			}
		}
	}))

	log.Fatal(app.Listen("0.0.0.0:8080"))
}
