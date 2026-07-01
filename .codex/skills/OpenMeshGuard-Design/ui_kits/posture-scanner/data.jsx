// Shared mock data for the OpenMeshGuard Posture Scanner UI kit.

window.OMG_DATA = {
  clusters: ['prod-eu-1', 'prod-us-1', 'staging-eu-1'],
  metrics: {
    mtlsCoverage: 82,
    openFindings: 37,
    critical: 12,
    ownedPct: 94,
    exceptions: 8,
    drifted: 5,
  },
  severityBreakdown: [
    { label: 'Critical', count: 12, status: 'fail' },
    { label: 'High', count: 9, status: 'fail' },
    { label: 'Medium', count: 11, status: 'warn' },
    { label: 'Low', count: 5, status: 'info' },
  ],
  controls: [
    { name: 'Strict mTLS in production', coverage: 82, status: 'warn' },
    { name: 'AuthorizationPolicy present', coverage: 71, status: 'warn' },
    { name: 'No public exposure of internal services', coverage: 96, status: 'pass' },
    { name: 'Ownership metadata complete', coverage: 94, status: 'pass' },
    { name: 'GitOps in sync (no drift)', coverage: 88, status: 'warn' },
  ],
  findings: [
    { id: 'OMG-1042', title: 'Ingress Gateway exposes an internal service to the public', kind: 'Gateway', resource: 'checkout-gateway', ns: 'checkout', owner: 'team-web', severity: 'fail', sevLabel: 'Critical', control: 'Exposure', age: '2d' },
    { id: 'OMG-1039', title: 'Namespace not enforcing strict mTLS', kind: 'PeerAuthentication', resource: 'ledger', ns: 'ledger', owner: 'team-core', severity: 'fail', sevLabel: 'Critical', control: 'mTLS', age: '2d' },
    { id: 'OMG-1031', title: 'Mesh-enabled app has no AuthorizationPolicy', kind: 'Workload', resource: 'ledger-svc', ns: 'ledger', owner: 'team-core', severity: 'fail', sevLabel: 'High', control: 'Authorization', age: '4d' },
    { id: 'OMG-1028', title: 'VirtualService routes to a workload outside the mesh', kind: 'VirtualService', resource: 'legacy-router', ns: 'edge', owner: 'unowned', severity: 'warn', sevLabel: 'High', control: 'Exposure', age: '5d' },
    { id: 'OMG-1024', title: 'Istio resource missing owner and repo metadata', kind: 'DestinationRule', resource: 'db-mtls', ns: 'data', owner: 'unowned', severity: 'warn', sevLabel: 'Medium', control: 'Ownership', age: '6d' },
    { id: 'OMG-1019', title: 'Policy drifted from GitOps source of truth', kind: 'AuthorizationPolicy', resource: 'payments-allow', ns: 'payments', owner: 'team-payments', severity: 'warn', sevLabel: 'Medium', control: 'Drift', age: '8d' },
    { id: 'OMG-1004', title: 'Exception expires in 5 days', kind: 'Exception', resource: 'permit-permissive-mtls', ns: 'checkout', owner: 'team-web', severity: 'info', sevLabel: 'Low', control: 'Exception', age: '12d' },
  ],
  workloads: [
    { name: 'payments-api', ns: 'payments', owner: 'team-payments', mtls: 'Enforced', authz: 'Present', coverage: 98, gitops: 'In sync' },
    { name: 'checkout-web', ns: 'checkout', owner: 'team-web', mtls: 'Permissive', authz: 'Present', coverage: 64, gitops: 'In sync' },
    { name: 'ledger-svc', ns: 'ledger', owner: 'team-core', mtls: 'Disabled', authz: 'Missing', coverage: 22, gitops: 'Drifted' },
    { name: 'search-api', ns: 'search', owner: 'team-discovery', mtls: 'Enforced', authz: 'Present', coverage: 91, gitops: 'In sync' },
    { name: 'legacy-router', ns: 'edge', owner: 'unowned', mtls: 'Permissive', authz: 'Missing', coverage: 40, gitops: 'Drifted' },
  ],
};
