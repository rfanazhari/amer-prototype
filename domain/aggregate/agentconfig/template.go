package agentconfig

// TemplateRef identifies an immutable template and its behavior interpretation.
// Supported catalog identities are selected by the versioned template catalog.
type TemplateRef struct{ Template, Version, Behavior string }

// valid reports whether the opaque version reference is structurally complete.
func (r TemplateRef) valid() bool { return r.Template != "" && r.Version != "" && r.Behavior != "" }
