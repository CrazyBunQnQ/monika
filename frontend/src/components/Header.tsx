import { Button } from '@/components/ui/button'
import { LogOut } from 'lucide-react'

interface HeaderProps {
  characterName: string
  onLogout?: () => void
}

export function Header({ characterName, onLogout }: HeaderProps) {
  return (
    <header className="h-14 border-b bg-card flex items-center justify-between px-4">
      <div className="flex items-center gap-2">
        <h1 className="font-bold text-lg">Monika</h1>
        <span className="text-muted-foreground">|</span>
        <span className="text-sm text-muted-foreground">
          当前角色: {characterName}
        </span>
      </div>
      <Button variant="ghost" size="sm" onClick={onLogout}>
        <LogOut className="w-4 h-4 mr-2" />
        退出
      </Button>
    </header>
  )
}
