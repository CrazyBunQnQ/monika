import React, { CSSProperties } from 'react';

// 16-colour palette tuned for the app's dark background.
// Normal (0-7) reuses the project semantic colours; bright (8-15) lightens them.
const PALETTE = [
    '#4d4d4d', '#cd5454', '#44a573', '#b89238', // 0-3 black red green yellow
    '#588dd5', '#877bb5', '#4ab4b4', '#d4d4dc', // 4-7 blue magenta cyan white
    '#6e6e6e', '#ff6b6b', '#6dd58c', '#d9b85a', // 8-11 bright black/red/green/yellow
    '#7fa9e8', '#a89bd0', '#6ec8c8', '#ffffff', // 12-15 bright blue/magenta/cyan/white
];

interface AnsiStyle {
    fg: string | null;
    bg: string | null;
    bold: boolean;
    dim: boolean;
    italic: boolean;
    underline: boolean;
    strike: boolean;
    inverse: boolean;
}

function newStyle(): AnsiStyle {
    return { fg: null, bg: null, bold: false, dim: false, italic: false, underline: false, strike: false, inverse: false };
}

function color256(n: number): string {
    if (n < 16) return PALETTE[n] ?? PALETTE[7];
    if (n >= 232) {
        const v = 8 + (n - 232) * 10;
        return `rgb(${v},${v},${v})`;
    }
    n -= 16;
    const conv = (c: number) => (c === 0 ? 0 : 55 + c * 40);
    return `rgb(${conv((n / 36) | 0)},${conv(((n / 6) | 0) % 6)},${conv(n % 6)})`;
}

function applySGR(params: number[], s: AnsiStyle): void {
    for (let i = 0; i < params.length; i++) {
        switch (params[i]) {
            case 0:
                Object.assign(s, newStyle());
                break;
            case 1: s.bold = true; break;
            case 2: s.dim = true; break;
            case 3: s.italic = true; break;
            case 4: s.underline = true; break;
            case 7: s.inverse = true; break;
            case 9: s.strike = true; break;
            case 22: s.bold = false; s.dim = false; break;
            case 23: s.italic = false; break;
            case 24: s.underline = false; break;
            case 27: s.inverse = false; break;
            case 29: s.strike = false; break;
            case 39: s.fg = null; break;
            case 49: s.bg = null; break;
            case 38:
                if (params[i + 1] === 5) { s.fg = color256(params[i + 2] ?? 0); i += 2; }
                else if (params[i + 1] === 2) { s.fg = `rgb(${params[i + 2] ?? 0},${params[i + 3] ?? 0},${params[i + 4] ?? 0})`; i += 4; }
                break;
            case 48:
                if (params[i + 1] === 5) { s.bg = color256(params[i + 2] ?? 0); i += 2; }
                else if (params[i + 1] === 2) { s.bg = `rgb(${params[i + 2] ?? 0},${params[i + 3] ?? 0},${params[i + 4] ?? 0})`; i += 4; }
                break;
            default:
                if (params[i] >= 30 && params[i] <= 37) s.fg = PALETTE[params[i] - 30];
                else if (params[i] >= 40 && params[i] <= 47) s.bg = PALETTE[params[i] - 40];
                else if (params[i] >= 90 && params[i] <= 97) s.fg = PALETTE[params[i] - 90 + 8];
                else if (params[i] >= 100 && params[i] <= 107) s.bg = PALETTE[params[i] - 100 + 8];
                break;
        }
    }
}

function styleToCSS(s: AnsiStyle): CSSProperties {
    const css: CSSProperties = {};
    if (s.inverse) {
        css.color = s.bg ?? '#08090d';
        css.background = s.fg ?? '#d4d4dc';
    } else {
        if (s.fg) css.color = s.fg;
        if (s.bg) css.background = s.bg;
    }
    if (s.bold) css.fontWeight = 700;
    if (s.dim) css.opacity = 0.5;
    if (s.italic) css.fontStyle = 'italic';
    if (s.underline || s.strike) {
        const decos: string[] = [];
        if (s.underline) decos.push('underline');
        if (s.strike) decos.push('line-through');
        css.textDecoration = decos.join(' ');
    }
    return css;
}

// Matches OSC, CSI (incl. private modes & SGR), and any other 2-byte escape.
const ESC_RE = /\x1b\][^\x1b]*(?:\x07|\x1b\\)|\x1b\[[0-?]*[ -/]*[@-~]|\x1b./g;

export function renderAnsi(text: string): React.ReactNode[] {
    const nodes: React.ReactNode[] = [];
    const s = newStyle();
    let last = 0;
    let key = 0;
    let m: RegExpExecArray | null;

    ESC_RE.lastIndex = 0;
    while ((m = ESC_RE.exec(text)) !== null) {
        if (m.index > last) {
            const seg = text.slice(last, m.index);
            nodes.push(<span key={key++} style={styleToCSS(s)}>{seg}</span>);
        }
        const esc = m[0];
        // Only SGR (CSI ... 'm') carries meaning for us; everything else is dropped.
        const sgr = esc.match(/^\x1b\[([0-9;]*)m$/);
        if (sgr) {
            const params = sgr[1] === '' ? [0] : sgr[1].split(';').map(Number);
            applySGR(params, s);
        }
        last = m.index + esc.length;
        // Guard against zero-length matches causing an infinite loop.
        if (esc.length === 0) ESC_RE.lastIndex++;
    }
    if (last < text.length) {
        nodes.push(<span key={key++} style={styleToCSS(s)}>{text.slice(last)}</span>);
    }
    return nodes;
}

export const AnsiText: React.FC<{ text: string; className?: string }> = ({ text, className }) => (
    <span className={className}>{renderAnsi(text)}</span>
);
