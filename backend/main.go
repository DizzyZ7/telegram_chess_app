package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub управляет игровыми комнатами и рассылкой сообщений.
type Hub struct {
	games map[string]*Game
	mu    sync.RWMutex
}

// Game представляет собой одну игровую сессию.
type Game struct {
	board     string // Пока что простая строка, позже будет использоваться chess.js
	players   map[*websocket.Conn]bool
	playerMu  sync.RWMutex
}

// Message представляет собой структуру сообщения, передаваемого по WebSocket.
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

var hub = Hub{
	games: make(map[string]*Game),
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	log.Println("WebSocket сервер запущен на :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Не удалось обновить соединение: %v", err)
		return
	}
	defer conn.Close()

	// Поиск или создание игровой комнаты
	gameID := r.URL.Query().Get("gameID")
	if gameID == "" {
		// TODO: Реализовать логику создания новой игры
		return
	}

	hub.mu.RLock()
	game, ok := hub.games[gameID]
	hub.mu.RUnlock()

	if !ok {
		game = &Game{
			players: make(map[*websocket.Conn]bool),
		}
		hub.mu.Lock()
		hub.games[gameID] = game
		hub.mu.Unlock()
	}

	game.playerMu.Lock()
	game.players[conn] = true
	game.playerMu.Unlock()

	defer func() {
		game.playerMu.Lock()
		delete(game.players, conn)
		game.playerMu.Unlock()
	}()

	// Чтение и обработка сообщений
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Рассылка сообщения всем игрокам в комнате
		game.playerMu.RLock()
		for client := range game.players {
			if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Ошибка при отправке сообщения: %v", err)
			}
		}
		game.playerMu.RUnlock()
	}
}
