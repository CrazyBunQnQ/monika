import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import rehypeRaw from 'rehype-raw'
import { CodeBlock, ExternalLink } from '../MarkdownShared'
import { useStore } from '../../store'

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
                {content}
            </ReactMarkdown>
        </div>
    )
}
