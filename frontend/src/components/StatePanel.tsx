import { cn } from '@/lib/utils'

interface StatePanelProps {
  character: {
    id: number
    name: string
    hp: number
    maxHp: number
    mp: number
    maxMp: number
    san: number
    maxSan: number
    luck: number
  }
}

export function StatePanel({ character }: StatePanelProps) {
  const stats = [
    { label: 'HP', value: character.hp, max: character.maxHp, color: 'bg-red-500' },
    { label: 'MP', value: character.mp, max: character.maxMp, color: 'bg-blue-500' },
    { label: 'SAN', value: character.san, max: character.maxSan, color: 'bg-yellow-500' },
    { label: '幸运', value: character.luck, max: 100, color: 'bg-green-500' },
  ]

  return (
    <aside className="w-72 border-l bg-card p-4 space-y-6">
      <h2 className="font-bold text-lg">{character.name}</h2>

      <div className="space-y-4">
        {stats.map((stat) => (
          <div key={stat.label} className="space-y-1">
            <div className="flex justify-between text-sm">
              <span>{stat.label}</span>
              <span>
                {stat.value}/{stat.max}
              </span>
            </div>
            <div className="h-2 bg-muted rounded-full overflow-hidden">
              <div
                className={cn('h-full transition-all', stat.color)}
                style={{ width: `${(stat.value / stat.max) * 100}%` }}
              />
            </div>
          </div>
        ))}
      </div>

      <div className="pt-4 border-t space-y-2">
        <h3 className="font-medium text-sm text-muted-foreground">属性</h3>
        <div className="grid grid-cols-2 gap-2 text-sm">
          <div className="bg-muted p-2 rounded">力量: 50</div>
          <div className="bg-muted p-2 rounded">体质: 50</div>
          <div className="bg-muted p-2 rounded">敏捷: 50</div>
          <div className="bg-muted p-2 rounded">外貌: 50</div>
          <div className="bg-muted p-2 rounded">意志: 50</div>
          <div className="bg-muted p-2 rounded">智力: 50</div>
          <div className="bg-muted p-2 rounded">体型: 50</div>
          <div className="bg-muted p-2 rounded">教育: 50</div>
        </div>
      </div>
    </aside>
  )
}
