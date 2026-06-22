import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import rehypeRaw from 'rehype-raw'
import { CodeBlock, ExternalLink } from '../MarkdownShared'
import { useStore } from '../../store'

interface FrontmatterField {
    key: string
    value: string
}

interface ParsedFrontmatter {
    fields: FrontmatterField[]
    body: string
}

const DELIM_RE = /^[-_=]{3,}\s*$/
const KV_RE = /^([a-zA-Z_][\w\-. ]*)\s*:\s*(.*)$/

function parseFrontmatter(content: string): ParsedFrontmatter | null {
    const lines = content.split('\n')
    if (lines.length < 3 || !DELIM_RE.test(lines[0])) return null
    let closeIdx = -1
    for (let i = 1; i < lines.length; i++) {
        if (DELIM_RE.test(lines[i])) { closeIdx = i; break }
    }
    if (closeIdx === -1) return null
    const fmBody = lines.slice(1, closeIdx).join('\n')
    const rest = lines.slice(closeIdx + 1).join('\n').replace(/^\n+/, '')
    const fields: FrontmatterField[] = []
    for (const line of fmBody.split('\n')) {
        const m = line.match(KV_RE)
        if (m) fields.push({ key: m[1].trim(), value: m[2].trim() })
    }
    if (fields.length === 0) return null
    return { fields, body: rest }
}

function resolveImageUrl(src: string | undefined, filePath: string, projectPath: string): string {
    if (!src) return ''
    if (/^(https?:|data:|\/)/.test(src)) return src
    // Ensure filePath is absolute — FileTree passes relative paths
    const normProject = projectPath.replace(/\\/g, '/')
    const absFilePath = /^[a-zA-Z]:\//.test(filePath) || filePath.startsWith('/')
        ? filePath.replace(/\\/g, '/')
        : `${normProject}/${filePath.replace(/\\/g, '/')}`
    const dir = absFilePath.substring(0, absFilePath.lastIndexOf('/'))
    const absPath = `${dir}/${src}`
    if (absPath.startsWith(normProject)) {
        const rel = absPath.substring(normProject.length + 1)
        return `/__local__/${rel}`
    }
    return src
}


interface MarkdownPreviewProps {
    content: string
    filePath: string
}

export default function MarkdownPreview({ content, filePath }: MarkdownPreviewProps) {
    const projectPath = useStore(s => s.projectPath)
    const fm = parseFrontmatter(content)
    const mdContent = fm ? fm.body : content
    return (
        <div
            className="markdown-preview markdown-body"
            style={{
                flex: 1,
                overflow: 'auto',
                padding: '24px 32px',
                background: 'var(--bg-root)',
            }}
        >
            {fm && (
                <div className="md-frontmatter">
                    {fm.fields.map((f, i) => (
                        <div key={i} className="md-frontmatter-row">
                            <span className="md-frontmatter-key">{f.key}</span>
                            <span className="md-frontmatter-val">{f.value}</span>
                        </div>
                    ))}
                </div>
            )}
            <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeRaw, rehypeHighlight]}
                components={{
                    pre: CodeBlock,
                    a: ExternalLink,
                    img: ({ src, alt, ...rest }) => (
                        <img src={resolveImageUrl(src, filePath, projectPath)} alt={alt} {...rest} />
                    ),
                }}
            >
                {mdContent}
            </ReactMarkdown>
        </div>
    )
}
