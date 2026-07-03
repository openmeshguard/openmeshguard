// Overview dashboard screen.
const { Card: OvCard, MetricStat, ProgressBar: OvProgress, StatusBadge: OvStatus, Button: OvButton } = window.OpenMeshGuardDesignSystem_65348c;

function SeverityBar({ items }) {
  const total = items.reduce((a, b) => a + b.count, 0);
  return (
    <div>
      <div style={{ display: 'flex', height: 10, borderRadius: 'var(--radius-pill)', overflow: 'hidden', marginBottom: 14 }}>
        {items.map((s) => (
          <div key={s.label} title={`${s.label}: ${s.count}`}
            style={{ width: `${(s.count / total) * 100}%`, background: `var(--status-${s.status}-solid)` }} />
        ))}
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {items.map((s) => (
          <div key={s.label} style={{ display: 'flex', alignItems: 'center', gap: 10, fontSize: 13 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', background: `var(--status-${s.status}-solid)`, flex: 'none' }} />
            <span style={{ flex: 1, color: 'var(--text-body)' }}>{s.label}</span>
            <span style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, color: 'var(--text-strong)' }}>{s.count}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function ControlRow({ c, onOpen }) {
  const [hover, setHover] = React.useState(false);
  return (
    <div
      onClick={onOpen}
      onMouseEnter={() => setHover(true)} onMouseLeave={() => setHover(false)}
      style={{ display: 'flex', alignItems: 'center', gap: 16, padding: '12px 20px', borderBottom: '1px solid var(--border-subtle)', cursor: 'pointer', background: hover ? 'var(--surface-hover)' : 'transparent' }}
    >
      <OvStatus status={c.status} dot />
      <span style={{ flex: 1, fontSize: 14, color: 'var(--text-strong)', fontWeight: 500 }}>{c.name}</span>
      <div style={{ width: 200 }}><OvProgress value={c.coverage} /></div>
    </div>
  );
}

function Overview({ onOpenFindings, onOpenResource }) {
  const d = window.OMG_DATA;
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      {/* Metrics */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 12 }}>
        <MetricStat label="mTLS coverage" value={d.metrics.mtlsCoverage} unit="%" delta="+4 since last scan" deltaTone="positive" status="warn" />
        <MetricStat label="Open findings" value={d.metrics.openFindings} delta={`${d.metrics.critical} critical`} deltaTone="negative" status="fail" />
        <MetricStat label="Owned resources" value={d.metrics.ownedPct} unit="%" delta="stable" status="pass" />
        <MetricStat label="Active exceptions" value={d.metrics.exceptions} delta="2 expiring soon" deltaTone="neutral" status="warn" />
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1.6fr 1fr', gap: 20, alignItems: 'start' }}>
        {/* Control posture */}
        <OvCard padded={false} title="Control posture" subtitle="Coverage across enterprise controls"
          actions={<OvButton size="sm" variant="secondary" onClick={onOpenFindings}>View all findings</OvButton>}>
          {d.controls.map((c) => <ControlRow key={c.name} c={c} onOpen={onOpenFindings} />)}
        </OvCard>

        {/* Findings by severity */}
        <OvCard title="Findings by severity" subtitle={`${d.metrics.openFindings} open`}>
          <SeverityBar items={d.severityBreakdown} />
        </OvCard>
      </div>

      {/* Recent findings preview */}
      <OvCard padded={false} title="Recent critical findings" actions={<OvButton size="sm" variant="ghost" onClick={onOpenFindings}>Open findings</OvButton>}>
        {d.findings.filter((f) => f.sevLabel === 'Critical' || f.sevLabel === 'High').slice(0, 4).map((f) => (
          <div key={f.id} onClick={() => onOpenResource(f)} style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '12px 20px', borderBottom: '1px solid var(--border-subtle)', cursor: 'pointer' }}>
            <OvStatus status={f.severity} solid={f.severity === 'fail'}>{f.sevLabel}</OvStatus>
            <span style={{ flex: 1, fontSize: 14, color: 'var(--text-strong)' }}>{f.title}</span>
            <code style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-muted)' }}>{f.ns}/{f.resource}</code>
            <Icon name="chevron-right" size={16} style={{ color: 'var(--text-subtle)' }} />
          </div>
        ))}
      </OvCard>
    </div>
  );
}

window.Overview = Overview;
