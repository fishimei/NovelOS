import ReactMarkdown from 'react-markdown';

type MarkdownRendererProps = {
  className?: string;
  source?: string;
  variant?: 'compact' | 'reading';
};

export function MarkdownRenderer({ className = '', source = '', variant }: MarkdownRendererProps) {
  const classes = ['markdown-content', variant ? `markdown-content--${variant}` : '', className].filter(Boolean).join(' ');

  return (
    <div className={classes}>
      <ReactMarkdown>{normalizeMarkdownSource(source)}</ReactMarkdown>
    </div>
  );
}

function normalizeMarkdownSource(source: string) {
  return source
    .replace(/[ \t]+(#{1,6}\s+)/g, '\n\n$1')
    .replace(/[ \t]+(>\s+)/g, '\n\n$1')
    .replace(/[ \t]+(-\s+\*\*)/g, '\n\n$1')
    .replace(/[ \t]+(\d+\.\s+\*\*)/g, '\n\n$1')
    .trim();
}
