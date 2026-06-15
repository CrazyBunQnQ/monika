import { useState, useEffect, useCallback } from 'react'
import { Events, Call } from '@wailsio/runtime'
import type { StreamEvent } from '../../../bindings/monika'

export interface DebugSessionState {
    id: string
    adapter: string
    status: string
    cwd: string
    program?: string
    threadId?: number
    stopReason?: string
    stopDescription?: string
    source?: { path?: string; name?: string }
    line?: number
    column?: number
    exitCode?: number
}

export interface DebugFrame {
    id: number
    name: string
    source?: { path?: string; name?: string }
    line: number
}

export interface DebugVariable {
    name: string
    value: string
    type?: string
    variablesReference?: number
}

export interface DebugBreakpoint {
    file: string
    line: number
    verified: boolean
    condition?: string
}

export interface DebugThread {
    id: number
    name: string
}

function parseSummary(content: string): Partial<DebugSessionState> | null {
    try {
        const raw = JSON.parse(content)
        const state: Partial<DebugSessionState> = {
            id: raw.id || '',
            adapter: raw.adapter || '',
            status: raw.status || 'unknown',
            exitCode: raw.hasExited ? raw.exitCode : undefined,
        }
        if (raw.stopLocation) {
            state.threadId = raw.stopLocation.threadId
            state.stopReason = raw.stopLocation.reason
            state.stopDescription = raw.stopLocation.text
            state.line = raw.stopLocation.line
            state.column = raw.stopLocation.column
            if (raw.stopLocation.source) {
                state.source = {
                    path: raw.stopLocation.source.path,
                    name: raw.stopLocation.source.name,
                }
            }
        }
        return state
    } catch {
        return null
    }
}

async function callDebugApi(method: string, sessionId: string, ...args: any[]): Promise<any> {
    try {
        return await Call.ByName(
            `monika/internal/api.App.Debug${method}`,
            sessionId,
            ...args
        )
    } catch {
        return null
    }
}

export function useDebugState(): {
    sessions: DebugSessionState[]
    activeSession: DebugSessionState | null
    activeSessionId: string | null
    frames: DebugFrame[]
    variables: DebugVariable[]
    breakpoints: DebugBreakpoint[]
    threads: DebugThread[]
    outputs: Record<string, string>
    activeFrameId: number | undefined
    setActiveFrame: (frameId: number) => Promise<void>
    refreshState: () => Promise<void>
} {
    const [sessions, setSessions] = useState<DebugSessionState[]>([])
    const [frames, setFrames] = useState<DebugFrame[]>([])
    const [variables, setVariables] = useState<DebugVariable[]>([])
    const [breakpoints, setBreakpoints] = useState<DebugBreakpoint[]>([])
    const [threads, setThreads] = useState<DebugThread[]>([])
    const [activeFrameId, setActiveFrameId] = useState<number | undefined>()
    const [outputs, setOutputs] = useState<Record<string, string>>({})

    // Fetch stack trace, then scopes, then variables — all on "stopped"
    const autoFetchStopped = useCallback(async (sessionId: string) => {
        const framesResult = await callDebugApi('GetStackTrace', sessionId, 20)
        if (framesResult && Array.isArray(framesResult)) {
            const mappedFrames: DebugFrame[] = framesResult.map((f: any) => ({
                id: f.id,
                name: f.name,
                source: f.source,
                line: f.line,
            }))
            setFrames(mappedFrames)

            if (mappedFrames.length > 0) {
                const topId = mappedFrames[0].id
                setActiveFrameId(topId)

                // Get scopes for the top frame
                const scopesResult = await callDebugApi('GetScopes', sessionId, topId)
                if (scopesResult && Array.isArray(scopesResult)) {
                    // Find Locals scope
                    const locals = scopesResult.find((s: any) => s.name === 'Locals')
                    if (locals && locals.variablesReference > 0) {
                        const varsResult = await callDebugApi('GetVariables', sessionId, locals.variablesReference)
                        if (varsResult && Array.isArray(varsResult)) {
                            setVariables(varsResult.map((v: any) => ({
                                name: v.name,
                                value: v.value,
                                type: v.type,
                                variablesReference: v.variablesReference,
                            })))
                        }
                    }
                }
            }
        }

        // Fetch threads
        const threadsResult = await callDebugApi('GetThreads', sessionId)
        if (threadsResult && Array.isArray(threadsResult)) {
            setThreads(threadsResult.map((t: any) => ({
                id: t.id,
                name: t.name,
            })))
        }
    }, [])

    // Switch frame context
    const setActiveFrame = useCallback(async (frameId: number) => {
        setActiveFrameId(frameId)
        const sid = sessions.find(s => s.status !== 'terminated')?.id
        if (!sid) return

        const scopesResult = await callDebugApi('GetScopes', sid, frameId)
        if (scopesResult && Array.isArray(scopesResult)) {
            const locals = scopesResult.find((s: any) => s.name === 'Locals')
            if (locals && locals.variablesReference > 0) {
                const varsResult = await callDebugApi('GetVariables', sid, locals.variablesReference)
                if (varsResult && Array.isArray(varsResult)) {
                    setVariables(varsResult.map((v: any) => ({
                        name: v.name,
                        value: v.value,
                        type: v.type,
                        variablesReference: v.variablesReference,
                    })))
                }
            }
        }
    }, [sessions])

    useEffect(() => {
        const unsubStream = Events.On('stream', (ev: any) => {
            const data = ev.data as StreamEvent
            if (!data.type || !data.type.startsWith('debug.')) return

            switch (data.type) {
                case 'debug.session.created': {
                    const summary = parseSummary(data.content || '')
                    if (!summary || !summary.id) break
                    setSessions((prev) => {
                        if (prev.find((s) => s.id === summary.id)) return prev
                        return [...prev, {
                            id: summary.id!,
                            adapter: summary.adapter || '',
                            status: summary.status || 'launching',
                            cwd: '',
                        }]
                    })
                    break
                }
                case 'debug.session.terminated': {
                    const summary = parseSummary(data.content || '')
                    const targetId = summary?.id || ''
                    setSessions((prev) =>
                        prev.map((s) =>
                            s.id === targetId ? { ...s, status: 'terminated', exitCode: summary?.exitCode } : s
                        )
                    )
                    if (targetId) {
                        setFrames([])
                        setVariables([])
                        setThreads([])
                        setBreakpoints([])
                    }
                    break
                }
                case 'debug.stopped': {
                    const summary = parseSummary(data.content || '')
                    if (!summary || !summary.id) break
                    setSessions((prev) =>
                        prev.map((s) =>
                            s.id === summary!.id
                                ? {
                                    ...s,
                                    status: 'stopped',
                                    threadId: summary!.threadId,
                                    stopReason: summary!.stopReason,
                                    stopDescription: summary!.stopDescription,
                                    source: summary!.source,
                                    line: summary!.line,
                                    column: summary!.column,
                                }
                                : s
                        )
                    )
                    autoFetchStopped(summary!.id)
                    break
                }
                case 'debug.continued': {
                    const summary = parseSummary(data.content || '')
                    if (!summary || !summary.id) break
                    setSessions((prev) =>
                        prev.map((s) =>
                            s.id === summary!.id
                                ? { ...s, status: 'running', threadId: undefined, stopReason: undefined }
                                : s
                        )
                    )
                    break
                }
                case 'debug.state.changed': {
                    const summary = parseSummary(data.content || '')
                    if (!summary || !summary.id) break
                    setSessions((prev) =>
                        prev.map((s) => {
                            if (s.id !== summary!.id) return s
                            return {
                                ...s,
                                status: summary!.status || s.status,
                                threadId: summary!.threadId ?? s.threadId,
                                stopReason: summary!.stopReason ?? s.stopReason,
                                source: summary!.source ?? s.source,
                                line: summary!.line ?? s.line,
                            }
                        })
                    )
                    if (summary!.status === 'stopped') {
                        autoFetchStopped(summary!.id)
                    }
                    break
                }
                case 'debug.output': {
                    setOutputs((prev) => ({ ...prev, _latest: data.content || '' }))
                    break
                }
            }
        })

        return () => { unsubStream() }
    }, [autoFetchStopped])

    const refreshState = useCallback(async () => {
        const sid = sessions.find(s => s.status !== 'terminated')?.id
        if (!sid) return

        const listResult = await callDebugApi('ListSessions', '')
        if (listResult && Array.isArray(listResult)) {
            setSessions(listResult.map((raw: any) => ({
                id: raw.id || '',
                adapter: raw.adapter || '',
                status: raw.status || 'unknown',
                cwd: '',
                program: raw.processName,
                threadId: raw.stopLocation?.threadId,
                stopReason: raw.stopLocation?.reason,
                line: raw.stopLocation?.line,
                source: raw.stopLocation?.source,
                exitCode: raw.hasExited ? undefined : undefined,
            })))
        }

        await autoFetchStopped(sid)
    }, [sessions, autoFetchStopped])

    const activeSession = sessions.find((s) => s.status !== 'terminated') || null
    const activeSessionId = activeSession?.id || null

    return {
        sessions, activeSession, activeSessionId,
        frames, variables, breakpoints, threads,
        activeFrameId, setActiveFrame,
        outputs, refreshState,
    }
}
