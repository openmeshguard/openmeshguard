import React from 'react';

/**
 * Coverage / posture bar. Value 0–100. Color follows thresholds unless `status` given.
 */
export function ProgressBar({ value = 0, status, showLabel = true, label, height = 8, style, ...rest }) {
  const pct = Math.max(0, Math.min(100, value));
  const auto = pct >= 90 ? 'pass' : pct >= 60 ? 'warn' : 'fail';
  const s = status || auto;
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '10px', ...style }} {...rest}>
      <div style={{ flex: 1, height, background: 'var(--surface-sunken)', borderRadius: 'var(--radius-pill)', overflow: 'hidden' }}>
        <div style={{
          width: `${pct}%`, height: '100%', background: `var(--status-${s}-solid)`,
          borderRadius: 'var(--radius-pill)', transition: 'width var(--dur-slow) var(--ease-out)',
        }} />
      </div>
      {showLabel && (
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: '12px', fontWeight: 600, color: 'var(--text-strong)', minWidth: 38, textAlign: 'right' }}>
          {label != null ? label : `${Math.round(pct)}%`}
        </span>
      )}
    </div>
  );
}
