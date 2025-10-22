import React, { useEffect, useState, useRef, useCallback } from 'react';
import WebApp from '@twa-dev/sdk';
import { Chessboard } from 'react-chessboard';
import { Chess } from 'chess.js';
import './style.css';

interface GameMessage {
    type: string;
    payload: {
        fen: string;
    };
}

function App() {
    const [game, setGame] = useState(new Chess());
    const [ws, setWs] = useState<WebSocket | null>(null);
    const [gameStatus, setGameStatus] = useState("Подключение...");
    const gameRef = useRef(new Chess());

    useEffect(() => {
        WebApp.ready();
        const urlParams = new URLSearchParams(window.location.search);
        const gameID = urlParams.get('gameID') || "default_game";
        const userID = WebApp.initDataUnsafe?.user?.id?.toString() || "guest";

        const websocket = new WebSocket(`ws://localhost:8080/ws?gameID=${gameID}&userID=${userID}`);

        websocket.onopen = () => {
            setGameStatus("Ожидание второго игрока...");
        };

        websocket.onmessage = (event) => {
            const message: GameMessage = JSON.parse(event.data);
            if (message.type === "game_state") {
                gameRef.current.load(message.payload.fen);
                setGame(new Chess(message.payload.fen));
                setGameStatus("Игра в процессе...");
            }
        };

        websocket.onclose = () => {
            setGameStatus("Соединение потеряно.");
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
            promotion: 'q', // По умолчанию всегда ферзь
        });

        if (move === null) return false;

        const moveMessage = {
            type: "make_move",
            payload: {
                move: move.from + move.to,
            }
        };

        ws.send(JSON.stringify(moveMessage));
        gameRef.current.move(move);
        setGame(new Chess(gameRef.current.fen()));

        return true;
    }, [ws, game]);

    return (
        <div className="App">
            <h1>Telegram Chess Mini App</h1>
            <p>{gameStatus}</p>
            <Chessboard position={game.fen()} onPieceDrop={onDrop} />
        </div>
    );
}

export default App;
