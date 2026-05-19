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
      <ReactMarkdown>{source}</ReactMarkdown>
    </div>
  );
}
