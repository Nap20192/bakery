import { useEffect, useState } from 'react';
import { Button } from './Button';
import { Icon } from './Icon';

// CopyButton copies text to the clipboard and shows brief "copied" feedback.
export function CopyButton({ getText, label = 'Копировать', copiedLabel = 'Скопировано', className = '', labelClassName = '', ...props }) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return undefined;
    const id = setTimeout(() => setCopied(false), 1500);
    return () => clearTimeout(id);
  }, [copied]);

  async function copy() {
    const text = typeof getText === 'function' ? getText() : String(getText ?? '');
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
    } catch {
      // Clipboard may be unavailable (insecure context); fail silently.
    }
  }

  return (
    <Button type="button" onClick={copy} className={className} title={label} aria-label={label} {...props}>
      <Icon name={copied ? 'select' : 'copy'} size={15} />
      <span className={labelClassName}>{copied ? copiedLabel : label}</span>
    </Button>
  );
}
