import { create } from 'zustand'
import { App } from '../../bindings/monika'

export type NotificationType =
    | 'reply-complete'
    | 'permission-request'
    | 'operation'

export type OperationKind = 'success' | 'error' | 'info'

export interface NotificationItem {
    id: string
    sessionId: string
    sessionTitle: string
    type: NotificationType
    message: string
    timestamp: number
    operation?: OperationKind
}

interface NotificationState {
    items: NotificationItem[]
    unreadHistory: NotificationItem[]
    unreadCount: number

    push: (item: Omit<NotificationItem, 'id' | 'timestamp'>) => void
    pushToast: (message: string, operation?: OperationKind) => void
    dismiss: (id: string) => void
    clearAll: () => void
    markAllRead: () => void
}

let counter = 0

let isMainWindowVisible = true

export function setMainWindowVisible(visible: boolean) {
    isMainWindowVisible = visible
}

export const useNotificationStore = create<NotificationState>((set, get) => ({
    items: [],
    unreadHistory: [],
    unreadCount: 0,

    push: (item) => {
        const id = `notif-${++counter}-${Date.now()}`
        const full: NotificationItem = { ...item, id, timestamp: Date.now() }
        set((s) => ({
            items: isMainWindowVisible ? [...s.items, full] : s.items,
            unreadHistory: [...s.unreadHistory, full],
            unreadCount: s.unreadCount + 1,
        }))
        // Trigger tray blink via Go
        App.SendTrayNotification(full.sessionId, full.sessionTitle, full.message).catch(() => { })
    },

    pushToast: (message, operation = 'info') => {
        const id = `notif-${++counter}-${Date.now()}`
        const full: NotificationItem = {
            id,
            sessionId: '',
            sessionTitle: '',
            message,
            timestamp: Date.now(),
            type: 'operation',
            operation,
        }
        set((s) => ({
            items: isMainWindowVisible ? [...s.items, full] : s.items,
        }))
    },

    dismiss: (id) => {
        set((s) => ({
            items: s.items.filter((it) => it.id !== id),
        }))
    },

    clearAll: () => {
        set({ items: [], unreadHistory: [], unreadCount: 0 })
        App.ClearTrayNotifications().catch(() => { })
    },

    markAllRead: () => {
        console.log('[monika] markAllRead called, clearing tray notifications')
        set({ unreadHistory: [], unreadCount: 0 })
        App.ClearTrayNotifications().catch(() => { })
    },
}))
