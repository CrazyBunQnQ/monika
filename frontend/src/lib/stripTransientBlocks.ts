// stripTransientBlocks.ts
// Removes system-injected transient XML blocks from user message content,
// leaving only the user's actual input text.

const TRANSIENT_BLOCK_PATTERNS = [
    /<env>[\s\S]*?<\/env>\s*/g,
    /<recalled-memory>[\s\S]*?<\/recalled-memory>\s*/g,
    /<memory-update>[\s\S]*?<\/memory-update>\s*/g,
    /<task-list>[\s\S]*?<\/task-list>\s*/g,
    /<context-summary>[\s\S]*?<\/context-summary>\s*/g,
    /<database-schema-available>[\s\S]*?<\/database-schema-available>\s*/g,
]

export function stripTransientBlocks(content: string): string {
    let result = content
    for (const pattern of TRANSIENT_BLOCK_PATTERNS) {
        result = result.replace(pattern, '')
    }
    return result.trim()
}
