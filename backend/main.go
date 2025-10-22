package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Message представляет собой структуру сообщения, передаваемого по WebSocket.
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Hub управляет игровыми комнатами и рассылкой сообщений.
type Hub struct {
	games map[string]*Game
	mu    sync.RWMutex
}

// Game представляет собой одну игровую сессию.
type Game struct {
	board     string
	players   map[*websocket.Conn]string // Используем map с ID игрока
	playerMu  sync.RWMutex
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

	gameID := r.URL.Query().Get("gameID")
	userID := r.URL.Query().Get("userID")

	if gameID == "" || userID == "" {
		conn.WriteMessage(websocket.TextMessage, []byte("Ошибка: gameID и userID обязательны"))
		return
	}

	hub.mu.RLock()
	game, ok := hub.games[gameID]
	hub.mu.RUnlock()

	if !ok {
		game = &Game{
			players: make(map[*websocket.Conn]string),
			board:   "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		}
		hub.mu.Lock()
		hub.games[gameID] = game
		hub.mu.Unlock()
	}

	game.playerMu.Lock()
	game.players[conn] = userID
	game.playerMu.Unlock()
	log.Printf("Игрок %s подключился к игре %s", userID, gameID)

	// Оповещаем всех игроков в комнате
	game.broadcastGameState()

	defer func() {
		game.playerMu.Lock()
		delete(game.players, conn)
		game.playerMu.Unlock()
		log.Printf("Игрок %s отключился от игры %s", userID, gameID)
		game.broadcastGameState() // Оповещаем об отключении
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Ошибка чтения сообщения от игрока %s: %v", userID, err)
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Ошибка декодирования JSON: %v", err)
			continue
		}

		switch msg.Type {
		case "make_move":
			// Логика проверки и выполнения хода
			// Пока просто рассылаем новый стейт
			if payload, ok := msg.Payload.(map[string]interface{}); ok {
				if newBoard, ok := payload["board"].(string); ok {
					game.board = newBoard
					game.broadcastGameState()
				}
			}
		}
	}
}

func (g *Game) broadcastGameState() {
	g.playerMu.RLock()
	defer g.playerMu.RUnlock()

	gameState := Message{
		Type: "game_state",
		Payload: map[string]string{
			"board": g.board,
		},
	}

	msgBytes, err := json.Marshal(gameState)
	if err != nil {
		log.Printf("Ошибка кодирования JSON для рассылки: %v", err)
		return
	}

	for client := range g.players {
		if err := client.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			log.Printf("Ошибка при отправке сообщения: %v", err)
		}
	}
}
