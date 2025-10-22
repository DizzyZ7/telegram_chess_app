package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/notnil/chess"
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
	chessGame *chess.Game
	players   map[*websocket.Conn]string
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
			chessGame: chess.NewGame(),
			players:   make(map[*websocket.Conn]string),
		}
		hub.mu.Lock()
		hub.games[gameID] = game
		hub.mu.Unlock()
	}

	game.playerMu.Lock()
	game.players[conn] = userID
	game.playerMu.Unlock()
	log.Printf("Игрок %s подключился к игре %s", userID, gameID)

	game.broadcastGameState()

	defer func() {
		game.playerMu.Lock()
		delete(game.players, conn)
		game.playerMu.Unlock()
		log.Printf("Игрок %s отключился от игры %s", userID, gameID)
		game.broadcastGameState()
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
			if payload, ok := msg.Payload.(map[string]interface{}); ok {
				if moveStr, ok := payload["move"].(string); ok {
					// Здесь логика проверки хода
					move, err := chess.UCINotation(moveStr)
					if err != nil {
						log.Printf("Неверный формат хода: %v", err)
						continue
					}

					if game.chessGame.Position().IsValid(move) {
						game.chessGame.Move(move)
						game.broadcastGameState()
					}
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
			"fen": g.chessGame.Position().String(),
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
