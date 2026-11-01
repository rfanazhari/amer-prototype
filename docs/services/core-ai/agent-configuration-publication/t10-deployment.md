# T10 deployment assessment — CORE-ACP-010

## Design mapping

Technical-design §3 restoration, §6 creation/restoration contracts and §9 Q4 remain the ownership map. T10 adds an explicit deployment catalog in `domain/agentconfig/deployment.go`, adapter restoration guards in `infrastructure/repository/agentconfigrepository/mapping.go`, and a read-only startup gate in `infrastructure/database/schema/agent_configuration_deployment.go` invoked from `infrastructure/runtime/resources.go`.

## Assessment outcomes

- **Retained:** Domain `Restore*` paths still avoid current eligibility/catalog/provider checks; T10 only adds an identity guard so unsupported behavior tokens fail as technical errors instead of partial loads.
- **Added:** `SupportedRestoration`/`RestorationSupported` express the approved retain-and-restrict policy; `RequireAgentConfigurationDeploymentCompatible` scans stored template identities before serving configuration traffic on startup.
- **Fixtures:** Sparse HR revision JSON proves stored historical shape survives restoration while current `AssessHR` remains stricter; behavior `2` rows prove explicit rejection.

## Proof

- Domain: `TestRestorationSupported*`, historical restoration repository test.
- Schema: sqlmock `TestRequireAgentConfigurationDeploymentCompatible` including rollback rehearsal (v1-only rows) and unsupported behavior rejection.
- Architecture: `TestAgentConfigurationDeploymentCompatibility` synchronizes decision/task/design artifacts.

No HTTP contract change; OpenAPI unchanged.
