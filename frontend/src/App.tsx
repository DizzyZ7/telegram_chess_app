import React, { useEffect, useState, useRef, useCallback } from 'react';
import WebApp from '@twa-dev/sdk';
import { Chessboard } from 'react-chessboard';
import { Chess } from 'chess.js';
import './style.css';

function App() {
    const [game, setGame] = useState(new Chess());
    const [gameID, setGameID] = useState<string | null>(null);
    const [ws, setWs] = useState<WebSocket | null>(null);
    const [boardPosition, setBoardPosition] = useState("start");
    const [gameStatus, setGameStatus] = useState("Ожидание второго игрока...");

    useEffect(() => {
        WebApp.ready();
        // В реальном приложении gameID должен передаваться через бота
        const id = "test_game_id"; 
        setGameID(id);

        const websocket = new WebSocket(`ws://localhost:8080/ws?gameID=${id}`);
        websocket.onopen = () => {
            console.log("Connected to WebSocket server");
        };
        websocket.onmessage = (event) => {
            const message = JSON.parse(event.data);
            if (message.type === "game_state") {
                setBoardPosition(message.payload.board);
                setGame(new Chess(message.payload.board));
            }
        };
        websocket.onclose = () => {
            console.log("Disconnected from WebSocket server");
        };
        setWs(websocket);

        return () => {
            websocket.close();
        };
    }, []);

    const onDrop = useCallback((sourceSquare: string, targetSquare: string) => {
        if (!ws) return false;

        const move = game.move({
            from: sourceSquare,
            to: targetSquare,
            promotion: 'q', // По умолчанию всегда ферзь, можно сделать диалог
        });

        if (move === null) return false;

        const gameState = {
            type: "game_state",
            payload: {
                board: game.fen(),
            }
        };
        ws.send(JSON.stringify(gameState));
        setBoardPosition(game.fen());

        return true;
    }, [game, ws]);


    return (
        <div className="App">
            <h1>Telegram Chess Mini App</h1>
            <p>{gameStatus}</p>
            <Chessboard position={boardPosition} onPieceDrop={onDrop} />
        </div>
    );
}

export default App;
