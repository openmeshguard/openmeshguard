import React from 'react';

/**
 * Removable tag for resource metadata: owners, environments, namespaces, labels.
 * Value is rendered in mono to match resource identifiers.
 */
export function Tag({ label, value, mono = true, onRemove, style, ...rest }) {
  return (
    <span
      style={{
        display: 'inline-flex', alignItems: 'center', gap: '6px',
        fontFamily: 'var(--font-sans)', fontSize: '12px', lineHeight: 1,
        padding: '4px 8px', borderRadius: 'var(--radius-sm)',
        background: 'var(--surface-sunken)', border: '1px solid var(--border-subtle)',
        color: 'var(--text-body)', ...style,
      }}
      {...rest}
    >
      {label && <span style={{ color: 'var(--text-muted)', fontWeight: 600 }}>{label}:</span>}
      <span style={{ fontFamily: mono ? 'var(--font-mono)' : 'inherit', color: 'var(--text-strong)' }}>{value}</span>
      {onRemove && (
        <button
          type="button" onClick={onRemove} aria-label="Remove"
          style={{ border: 'none', background: 'none', cursor: 'pointer', color: 'var(--text-subtle)', padding: 0, marginLeft: 1, fontSize: 14, lineHeight: 1 }}
        >×</button>
      )}
    </span>
  );
}
