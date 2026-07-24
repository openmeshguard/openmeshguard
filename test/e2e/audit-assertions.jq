def allowed_resource:
  (.objectRef.apiGroup // "") as $group |
  .objectRef.resource as $resource |
  if $group == "" then
    ["namespaces", "nodes", "pods", "services"] | index($resource) != null
  elif $group == "apps" then
    ["daemonsets", "deployments", "replicasets", "statefulsets"] | index($resource) != null
  elif $group == "discovery.k8s.io" then
    ["endpointslices"] | index($resource) != null
  elif $group == "networking.istio.io" then
    ["destinationrules", "envoyfilters", "gateways", "proxyconfigs", "serviceentries", "sidecars", "virtualservices", "workloadentries", "workloadgroups"] | index($resource) != null
  elif $group == "security.istio.io" then
    ["authorizationpolicies", "peerauthentications", "requestauthentications"] | index($resource) != null
  elif $group == "telemetry.istio.io" then
    $resource == "telemetries"
  elif $group == "gateway.networking.k8s.io" then
    ["backendtlspolicies", "gatewayclasses", "gateways", "grpcroutes", "httproutes", "referencegrants", "tcproutes", "tlsroutes", "udproutes"] | index($resource) != null
  else false
  end;

map(select(
  .user.username == $cluster_user or
  .user.username == $namespace_user or
  .user.username == $waypoint_limited_user
)) as $events |
($events | length > 0) and
all($events[];
  .verb == "list" and
  ((.objectRef.subresource // "") == "") and
  allowed_resource
) and
any($events[]; .user.username == $cluster_user) and
any($events[]; .user.username == $namespace_user) and
any($events[]; .user.username == $waypoint_limited_user) and
any($events[];
  .user.username == $namespace_user and
  .verb == "list" and
  .objectRef.apiGroup == "security.istio.io" and
  .objectRef.resource == "peerauthentications" and
  .objectRef.namespace == "istio-system" and
  .responseStatus.code == 403
) and
any(.[];
  .user.username == $probe_user and
  .verb == "create" and
  (.objectRef.apiGroup // "") == "" and
  .objectRef.resource == "configmaps" and
  .responseStatus.code == 403
) and
all(.[];
  .user.username != $fixture_manager_user and
  .user.username != $kind_admin_user
)
