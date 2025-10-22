import React, { useEffect } from 'react';
import WebApp from '@twa-dev/sdk';

function App() {
  useEffect(() => {
    WebApp.ready();
  }, []);

  return (
    <div className="App">
      <h1>Telegram Chess Mini App</h1>
      <p>Добро пожаловать в игру!</p>
    </div>
  );
}

export default App;
