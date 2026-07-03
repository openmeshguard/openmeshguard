import React from 'react';

/**
 * Dense governance data table. Columns: [{key,header,render?,width?,align?,mono?}].
 * Rows are plain objects. Optional row click, hover highlight, and sticky header.
 */
export function DataTable({ columns = [], rows = [], onRowClick, rowKey, empty = 'No data.', style, ...rest }) {
  const [hover, setHover] = React.useState(-1);
  return (
    <div style={{ width: '100%', overflowX: 'auto', ...style }} {...rest}>
      <table style={{ width: '100%', borderCollapse: 'collapse', fontFamily: 'var(--font-sans)' }}>
        <thead>
          <tr>
            {columns.map((c) => (
              <th key={c.key} style={{
                textAlign: c.align || 'left', padding: 'var(--pad-cell-y) var(--pad-cell-x)',
                fontSize: '11px', fontWeight: 600, letterSpacing: '0.05em', textTransform: 'uppercase',
                color: 'var(--text-muted)', background: 'var(--surface-sunken)',
                borderBottom: '1px solid var(--border-subtle)', whiteSpace: 'nowrap',
                width: c.width, position: 'sticky', top: 0,
              }}>{c.header}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && (
            <tr><td colSpan={columns.length} style={{ padding: '28px', textAlign: 'center', color: 'var(--text-muted)', fontSize: '13px' }}>{empty}</td></tr>
          )}
          {rows.map((row, i) => (
            <tr
              key={rowKey ? row[rowKey] : i}
              onMouseEnter={() => setHover(i)}
              onMouseLeave={() => setHover(-1)}
              onClick={onRowClick ? () => onRowClick(row, i) : undefined}
              style={{
                background: hover === i ? 'var(--surface-hover)' : 'transparent',
                cursor: onRowClick ? 'pointer' : 'default',
                transition: 'background var(--dur-fast)',
              }}
            >
              {columns.map((c) => (
                <td key={c.key} style={{
                  padding: 'var(--pad-cell-y) var(--pad-cell-x)', textAlign: c.align || 'left',
                  fontSize: c.mono ? '13px' : '13px', fontFamily: c.mono ? 'var(--font-mono)' : 'var(--font-sans)',
                  color: c.mono ? 'var(--text-strong)' : 'var(--text-body)',
                  borderBottom: '1px solid var(--border-subtle)', whiteSpace: 'nowrap',
                }}>
                  {c.render ? c.render(row[c.key], row, i) : row[c.key]}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
