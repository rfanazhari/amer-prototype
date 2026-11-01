# Core AI Agent Runtime — Design Direction v0.1

**Status:** Draft direction, not an implementation specification  
**Product context:** Template-Based No-Code AI Agent Platform  
**Reference use case:** HR Interview Agent (not a runtime-specific dependency)

## 1. Objective

Build a reusable, provider-agnostic execution foundation that manages agent lifecycle, context, model interaction, tool execution, and state across different use cases.

The existing proof of concept uses a Go backend, plain HTML, REST API, WebRTC, and OpenAI Live-1. Speech integration has been demonstrated. The PoC can be adjusted later; it should not dictate the framework's abstractions.

## 2. Scope and Responsibilities

| Component | Responsibility | Initial priority |
|---|---|---|
| Agent Definition | Agent identity, instructions, capabilities, and configuration contract | P0 |
| Agent Runner | Starts and coordinates an execution | P0 |
| Lifecycle Manager | Manages execution status and transitions | P0 |
| Context Manager | Prepares and maintains run context | P0 |
| Model Adapter | Isolates model-provider interaction | P0 |
| Tool Registry & Executor | Registers, validates, authorizes, and executes tools | P0 |
| State Store | Persists execution state | P0 |
| Policy Enforcement | Validates actions against boundaries | P0 |
| Observability | Logs, traces, execution events, and errors | P1 |

**Priority note:** P0 does not mean each component must become a separate service or package.

## 3. Logical Architecture

```text
Agent Definition
  - Identity / Instructions / Capabilities / Tools / Policies
                     |
                     v
             Core Agent Runtime
  +--------------------------------------------+
  | Agent Runner        Lifecycle Manager       |
  | Context Manager     Model Adapter           |
  | Tool Executor       State Store             |
  | Policy Enforcement  Execution Events        |
  +--------------------------------------------+
                     |
                     v
        Provider & Infrastructure Adapters
       Model Provider / Voice & WebRTC /
          Persistence / External APIs
```

The diagram expresses logical responsibilities, not a prescribed deployment topology.

## 4. Responsibility Boundaries

### Core Runtime
Owns execution lifecycle, context, state, tool execution, and enforcement of common policies. It must not embed HR-specific concepts such as candidates, interview stages, or salary rules.

### Agent Implementation
Owns domain-specific instructions, workflow, capabilities, and output requirements. The HR Interview Agent is one consumer of the runtime.

### Provider & Infrastructure Adapters
Own communication with AI models, transport, persistence, and external systems. OpenAI Live-1 and WebRTC are integrations, not the runtime's architectural center.

## 5. Execution Model Direction

Different agent types may need different execution models. Do not assume that all agents use voice, conversation sessions, or finite-state interview workflows.

```text
Agent Runtime
├── Core
│   ├── Agent Definition
│   ├── Agent Lifecycle
│   ├── Context
│   ├── State
│   ├── Tools
│   └── Policies
├── Execution Models
│   ├── Conversational (first candidate)
│   └── Task-based (future)
└── Adapters
    ├── Model Provider
    ├── State Storage
    └── Transport
```

This is a conceptual decomposition, **not** a requirement to create these directories or interfaces now.

## 6. Design Principles

1. **Provider-agnostic core:** do not couple runtime contracts to OpenAI Live-1.
2. **Domain separation:** keep HR-specific behavior outside the runtime.
3. **Controlled execution:** tools and transitions are validated by runtime/backend logic, not merely by prompts.
4. **Minimal abstractions:** implement only contracts justified by an actual execution path.
5. **Incremental adoption:** the existing PoC may be adapted after runtime boundaries become clear; avoid premature rewrites.
6. **Traceable outcomes:** execution state, failures, and consequential tool actions should be inspectable.

## 7. Suggested Design Sequence

1. **Runtime responsibilities and boundaries:** identify what belongs in core versus agent implementation and adapters.
2. **Agent execution model:** describe the full path from incoming task/input to completed outcome, including tool calls.
3. **Core contracts:** derive `AgentDefinition`, `AgentRun`, `AgentContext`, `ModelAdapter`, `Tool`, and `StateStore` from that flow.
4. **Lifecycle and failure handling:** define states, cancellation, timeouts, retries, and error propagation.
5. **Reference-agent validation:** exercise the runtime with HR Interview Agent as a consumer, without hard-coding its domain.

**Next design focus:** Agent Execution Model. Determine what happens inside the runtime from receiving input to producing an outcome **before** committing to Go interfaces, package layout, or infrastructure choices.

## 8. Explicit Non-Goals for v0.1

- General-purpose multi-agent collaboration.
- Universal visual workflow builder.
- Long-term agent memory or vector database by default.
- A separate microservice for every runtime component.
- Full support for all conversational and asynchronous execution patterns from day one.
- Rewriting the existing HR Interviewer PoC before the core contracts are understood.

## 9. Open Decisions

- What is the minimum agent execution lifecycle and its state transitions?
- What is the first supported execution model and its input/output contract?
- Which model interactions and tool-call semantics belong in the common abstraction?
- What state must survive interruption, reconnection, or process restart?
- Which policy checks are common versus agent-specific?
- How will a second, non-HR use case validate the abstraction?

---

*This document captures architectural direction only. Detailed API contracts, persistence schema, package structure, and implementation plan remain undecided.*
