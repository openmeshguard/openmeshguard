Dense governance data table. Define columns with optional cell renderers; pass row objects.

```jsx
<DataTable
  columns={[
    { key: 'name', header: 'Resource', mono: true },
    { key: 'ns', header: 'Namespace', mono: true },
    { key: 'mtls', header: 'mTLS', render: v => <StatusBadge status={v==='Enforced'?'pass':'fail'}>{v}</StatusBadge> },
    { key: 'cov', header: 'Coverage', align: 'right', render: v => <ProgressBar value={v} /> },
  ]}
  rows={rows}
  rowKey="name"
  onRowClick={openDetail}
/>
```

Wrap in `<Card padded={false}>` for a bordered panel. Abbreviate long lists in mockups (a few rows stand in for many).
