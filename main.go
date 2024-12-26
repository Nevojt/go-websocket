package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

// Глобальна змінна для зберігання підключень
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Mutex для синхронізації доступу до клієнтів
var mutex = sync.Mutex{}

// Структура для повідомлень
type Message struct {
	MessageType int    `json:"-"`
	Message     string `json:"message"`
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer func() {
		mutex.Lock()
		delete(clients, conn) // Видаляємо клієнта при відключенні
		mutex.Unlock()
		err := conn.Close()
		if err != nil {
			return
		}
	}()

	// Додаємо нового клієнта
	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()

	fmt.Println("New client connected")

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}
		// Надсилаємо повідомлення в канал трансляції
		broadcast <- Message{MessageType: messageType, Message: string(message)}
	}
}

func handleMessages() {
	for {
		// Отримуємо повідомлення з каналу
		msg := <-broadcast

		// Транслюємо повідомлення всім клієнтам
		mutex.Lock()
		for client := range clients {
			err := client.WriteMessage(msg.MessageType, []byte(msg.Message))
			if err != nil {
				fmt.Println("Error writing message:", err)
				err := client.Close()
				if err != nil {
					return
				}
				delete(clients, client) // Видаляємо клієнта, якщо виникла помилка
			}
		}
		mutex.Unlock()
	}
}

// ws://localhost:8080/ws
func main() {
	http.HandleFunc("/ws", handleWebSocket)

	// Запускаємо обробку повідомлень у окремій горутині
	go handleMessages()

	fmt.Println("Starting WebSocket server on port 8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
