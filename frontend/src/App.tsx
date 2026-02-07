import { useState } from 'react'
import { GameConsole } from '@/components/GameConsole'
import { LoginPage } from '@/pages/LoginPage'

function App() {
  const [isLoggedIn, setIsLoggedIn] = useState(false)

  return (
    <div className="min-h-screen bg-background text-foreground">
      {isLoggedIn ? (
        <GameConsole />
      ) : (
        <LoginPage onLogin={() => setIsLoggedIn(true)} />
      )}
    </div>
  )
}

export default App
