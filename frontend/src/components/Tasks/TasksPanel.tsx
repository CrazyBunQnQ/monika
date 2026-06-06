import { IDockviewPanelProps } from 'dockview'
import { useStore } from '../../store'

export default function TasksPanel({ }: IDockviewPanelProps) {
    const bgTasks = useStore((s) => s.bgTasks)
    const selectedBgTaskId = useStore((s) => s.selectedBgTaskId)
    const selectBgTask = useStore((s) => s.selectBgTask)

    return (
        <div className="flex flex-col h-full">
            <div className="px-3 py-2 text-xs font-semibold text-[var(--text-muted)] tracking-wider border-b border-[var(--border)]">
                TASKS
            </div>
            <div className="flex-1 overflow-auto">
                {bgTasks.length === 0 && (
                    <div className="px-3 py-4 text-xs text-[var(--text-muted)]">No background tasks</div>
                )}
                {bgTasks.map((task) => (
                    <div
                        key={task.id}
                        onClick={() => selectBgTask(task.id)}
                        className={`px-3 py-2 cursor-pointer border-b border-[var(--border)] hover:bg-[var(--bg-hover)] ${selectedBgTaskId === task.id ? 'bg-[var(--bg-active)]' : ''
                            }`}
                    >
                        <div className="flex items-center gap-2">
                            <span
                                className={`w-2 h-2 rounded-full flex-shrink-0 ${task.status === 'running'
                                        ? 'bg-green-500'
                                        : task.status === 'stopped'
                                            ? 'bg-red-500'
                                            : 'bg-gray-500'
                                    }`}
                            />
                            <span className="text-sm text-[var(--text)] truncate">{task.command}</span>
                        </div>
                        <div className="text-xs text-[var(--text-muted)] mt-0.5">
                            PID {task.pid} · {task.status}
                        </div>
                    </div>
                ))}
            </div>
        </div>
    )
}
