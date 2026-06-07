import { useState } from 'react'
import { LspSymbol } from '../../lib/lspService'
import {
    Box,
    SquareFunction,
    Hammer,
    Layers,
    Sigma,
    CircleDot,
    Lock,
    Boxes,
    Circle,
    List,
    Minus,
    Dot,
} from 'lucide-react'

const kindIconMap: Record<number, React.ReactNode> = {
    5: <Box size={10} strokeWidth={1.5} />,            // Class
    6: <SquareFunction size={10} strokeWidth={1.5} />,  // Method
    9: <Hammer size={10} strokeWidth={1.5} />,          // Constructor
    11: <Layers size={10} strokeWidth={1.5} />,          // Interface
    12: <Sigma size={10} strokeWidth={1.5} />,           // Function
    13: <CircleDot size={10} strokeWidth={1.5} />,       // Variable
    14: <Lock size={10} strokeWidth={1.5} />,            // Constant
    23: <Boxes size={10} strokeWidth={1.5} />,           // Struct
    22: <Circle size={10} strokeWidth={1.5} />,          // EnumMember
    10: <List size={10} strokeWidth={1.5} />,            // Enum
    7: <Minus size={10} strokeWidth={1.5} />,           // Property
    8: <Dot size={10} strokeWidth={1.5} />,             // Field
}

function kindIcon(kind: number): React.ReactNode {
    return kindIconMap[kind] || <span style={{ fontSize: 8 }}>·</span>
}

interface Props {
    symbols: LspSymbol[]
    onSymbolClick: (sym: LspSymbol) => void
    currentLine: number | null
}

export function LspSymbolSidebar({ symbols, onSymbolClick, currentLine }: Props) {
    const [collapsed, setCollapsed] = useState<Set<string>>(new Set())

    const toggle = (name: string) => {
        const next = new Set(collapsed)
        next.has(name) ? next.delete(name) : next.add(name)
        setCollapsed(next)
    }

    // visible state managed by parent via showSymbols

    return (
        <div
            style={{
                width: 220,
                overflow: 'auto',
                borderLeft: '1px solid var(--sym-border)',
                background: 'var(--sym-bg)',
                fontSize: 11,
                fontFamily: 'var(--font-mono)',
                userSelect: 'none',
            }}
        >
            <div
                style={{
                    padding: '6px 10px',
                    color: 'var(--text-dim)',
                    fontSize: 10,
                    fontWeight: 600,
                    letterSpacing: 0.5,
                    borderBottom: '1px solid var(--sym-border)',
                }}
            >
                SYMBOLS
            </div>
            {symbols.length === 0 && (
                <div style={{ padding: '12px 10px', color: 'var(--sym-kind-fg)', fontSize: 10 }}>
                    No symbols
                </div>
            )}
            {symbols.map((s, i) => (
                <SymbolNode
                    key={`${s.name}-${i}`}
                    sym={s}
                    depth={0}
                    collapsed={collapsed}
                    onToggle={toggle}
                    onClick={onSymbolClick}
                    currentLine={currentLine}
                />
            ))}
        </div>
    )
}

function SymbolNode({ sym, depth, collapsed, onToggle, onClick, currentLine }: {
    sym: LspSymbol
    depth: number
    collapsed: Set<string>
    onToggle: (name: string) => void
    onClick: (sym: LspSymbol) => void
    currentLine: number | null
}) {
    const hasChildren = sym.children && sym.children.length > 0
    const isCollapsed = hasChildren && collapsed.has(sym.name)
    const isActive = currentLine !== null && sym.startLine <= currentLine && sym.endLine >= currentLine

    return (
        <div>
            <div
                onClick={() => {
                    if (hasChildren) onToggle(sym.name)
                    onClick(sym)
                }}
                style={{
                    padding: '2px 6px 2px ' + (8 + depth * 12) + 'px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    color: isActive ? 'var(--sym-active-fg)' : 'var(--text-primary)',
                    background: isActive ? 'var(--sym-active-bg)' : 'transparent',
                    whiteSpace: 'nowrap',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    transition: 'background 0.12s, color 0.12s',
                }}
                onMouseEnter={e => {
                    if (!isActive) e.currentTarget.style.background = 'var(--bg-hover)'
                }}
                onMouseLeave={e => {
                    if (!isActive) e.currentTarget.style.background = 'transparent'
                }}
                title={sym.name}
            >
                {hasChildren ? (
                    <span style={{ width: 10, flexShrink: 0, color: 'var(--sym-expand-fg)' }}>
                        {isCollapsed ? '▸' : '▾'}
                    </span>
                ) : (
                    <span style={{ width: 10, flexShrink: 0 }} />
                )}
                <span style={{
                    color: 'var(--sym-kind-fg)',
                    width: 12,
                    flexShrink: 0,
                    display: 'inline-flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                }}>
                    {kindIcon(sym.kind)}
                </span>
                <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{sym.name}</span>
            </div>
            {hasChildren && !isCollapsed && sym.children.map((child, i) => (
                <SymbolNode
                    key={`${child.name}-${i}`}
                    sym={child}
                    depth={depth + 1}
                    collapsed={collapsed}
                    onToggle={onToggle}
                    onClick={onClick}
                    currentLine={currentLine}
                />
            ))}
        </div>
    )
}