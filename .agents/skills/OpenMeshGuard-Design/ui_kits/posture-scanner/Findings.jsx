// Findings list screen with filter tabs and a data table.
const { Card: FCard, DataTable: FTable, Tabs: FTabs, StatusBadge: FStatus, Badge: FBadge, Avatar: FAvatar, Button: FButton, Input: FInput } = window.OpenMeshGuardDesignSystem_65348c;

function Findings({ onOpenResource }) {
  const d = window.OMG_DATA;
  const [tab, setTab] = React.useState('all');
  const controls = ['all', 'mTLS', 'Authorization', 'Exposure', 'Drift', 'Ownership'];
  const rows = tab === 'all' ? d.findings : d.findings.filter((f) => f.control === tab);

  return (
    <FCard padded={false}
      title="Findings"
      subtitle={`${rows.length} of ${d.findings.length} findings`}
      actions={<FButton size="sm" variant="secondary" leftIcon={<Icon name="download" size={15} />}>Export evidence</FButton>}
    >
      <div style={{ padding: '0 20px' }}>
        <FTabs value={tab} onChange={setTab} items={controls.map((c) => ({ id: c, label: c === 'all' ? 'All' : c, count: c === 'all' ? d.findings.length : d.findings.filter((f) => f.control === c).length || undefined }))} />
      </div>
      <FTable
        rowKey="id"
        onRowClick={onOpenResource}
        columns={[
          { key: 'severity', header: 'Severity', width: 110, render: (v, r) => <FStatus status={v} solid={v === 'fail'}>{r.sevLabel}</FStatus> },
          { key: 'title', header: 'Finding', render: (v, r) => (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <span style={{ color: 'var(--text-strong)', fontFamily: 'var(--font-sans)', fontWeight: 500, whiteSpace: 'normal' }}>{v}</span>
              <code style={{ fontFamily: 'var(--font-mono)', fontSize: 11.5, color: 'var(--text-muted)' }}>{r.id}</code>
            </div>
          ) },
          { key: 'kind', header: 'Kind', render: (v) => <code style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-body)' }}>{v}</code> },
          { key: 'resource', header: 'Resource', mono: true },
          { key: 'owner', header: 'Owner', render: (v) => v === 'unowned'
            ? <FStatus status="warn">Unowned</FStatus>
            : <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><FAvatar name={v} size="sm" /><span style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{v}</span></span> },
          { key: 'control', header: 'Control', render: (v) => <FBadge tone="neutral">{v}</FBadge> },
          { key: 'age', header: 'Age', align: 'right', mono: true },
        ]}
        rows={rows}
      />
    </FCard>
  );
}

window.Findings = Findings;
