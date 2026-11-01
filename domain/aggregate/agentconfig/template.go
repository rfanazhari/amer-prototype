package agentconfig

// TemplateRef identifies an immutable template and its behavior interpretation.
// Its identity is opaque here; supported catalog entries are selected by T5.
type TemplateRef struct{ Template, Version, Behavior string }

// valid reports whether the opaque version reference is structurally complete.
func (r TemplateRef) valid() bool { return r.Template != "" && r.Version != "" && r.Behavior != "" }
