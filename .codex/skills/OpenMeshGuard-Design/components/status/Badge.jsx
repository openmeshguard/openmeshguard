import React from 'react';

const TONES = {
  neutral: { color: 'var(--text-body)', background: 'var(--surface-sunken)', border: 'var(--border-subtle)' },
  brand: { color: 'var(--brand-700)', background: 'var(--brand-50)', border: 'var(--brand-100)' },
  outline: { color: 'var(--text-body)', background: 'transparent', border: 'var(--border-default)' },
};

/**
 * Small neutral label for counts, categories, and metadata.
 */
export function Badge({ tone = 'neutral', children, style, ...rest }) {
  const t = TONES[tone] || TONES.neutral;
  return (
    <span
      style={{
        display: 'inline-flex', alignItems: 'center', gap: '5px',
        fontFamily: 'var(--font-sans)', fontSize: '12px', fontWeight: 600, lineHeight: 1,
        padding: '3px 8px', borderRadius: 'var(--radius-sm)',
        color: t.color, background: t.background, border: `1px solid ${t.border}`,
        ...style,
      }}
      {...rest}
    >
      {children}
    </span>
  );
}
