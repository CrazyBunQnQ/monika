import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

export function LandingPage() {
  const navigate = useNavigate()

  return (
    <div className="min-h-screen bg-gradient-to-b from-background to-muted/20">
      <div className="container mx-auto px-4 py-16">
        <div className="max-w-3xl mx-auto text-center space-y-8">
          <h1 className="text-5xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-primary to-primary/60">
            Monika
          </h1>
          <p className="text-xl text-muted-foreground">
            AI 驱动的《克苏鲁的呼唤》在线跑团平台
          </p>
          <p className="text-muted-foreground max-w-xl mx-auto">
            与 AI 守密人一起探索未知的世界，体验前所未有的 TRPG 冒险。
          </p>

          <div className="flex gap-4 justify-center">
            <Button size="lg" onClick={() => navigate('/auth')}>
              开始冒险
            </Button>
            <Button size="lg" variant="outline" onClick={() => navigate('/auth')}>
              了解更多
            </Button>
          </div>

          <div className="grid md:grid-cols-3 gap-6 mt-16">
            <Card>
              <CardContent className="pt-6">
                <h3 className="font-semibold mb-2">AI 守密人</h3>
                <p className="text-sm text-muted-foreground">
                  24/7 可用的 AI KP，随时开始你的冒险
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="pt-6">
                <h3 className="font-semibold mb-2">自动化规则</h3>
                <p className="text-sm text-muted-foreground">
                  检定、战斗、SAN 值自动处理，专注于角色扮演
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="pt-6">
                <h3 className="font-semibold mb-2">长团记忆</h3>
                <p className="text-sm text-muted-foreground">
                  结构化记忆系统，随时回顾历史，断点续跑
                </p>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  )
}
