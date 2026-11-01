package agentconfighandler

import (
	"net/http"
	"strings"

	"github.com/rfanazhari/amer-prototype/application/query/agentconfigquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentconfigusecase"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

// Handler translates configuration HTTP requests through injected application operation interfaces.
type Handler struct {
	Catalog       agentconfigquery.CatalogOperation
	Create        agentconfigusecase.CreateOperation
	Instance      agentconfigquery.InstanceOperation
	Replace       agentconfigusecase.ReplaceOperation
	Validate      agentconfigusecase.ValidateOperation
	Publish       agentconfigusecase.PublishOperation
	Revision      agentconfigquery.RevisionOperation
	ListInstances agentconfigquery.ListInstancesOperation
	CSRF          service.CSRFTokens
	Sessions      service.CurrentAdministrativeSession
	Origin        string
	SecureCookie  bool
	LogError      func(error)
}

// ServeHTTP dispatches administrative configuration routes.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "admin" {
		fail(w, r, http.StatusNotFound, errs.NotFound)
		return
	}
	switch {
	case len(parts) == 2 && parts[1] == "agent-templates":
		h.catalog(w, r)
	case len(parts) == 2 && parts[1] == "agent-instances":
		h.instances(w, r)
	case len(parts) == 3 && parts[1] == "agent-instances":
		h.readInstance(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "agent-instances" && parts[3] == "draft":
		h.replace(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "agent-instances" && parts[3] == "validations":
		h.validate(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "agent-instances" && parts[3] == "publications":
		h.publish(w, r, parts[2])
	case len(parts) == 5 && parts[1] == "agent-instances" && parts[3] == "revisions":
		h.readRevision(w, r, parts[2], parts[4])
	default:
		fail(w, r, http.StatusNotFound, errs.NotFound)
	}
}

// catalog lists both supported template versions after administrative authentication.
func (h Handler) catalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	r, ok := h.prepareRead(w, r)
	if !ok {
		return
	}
	templates, err := h.Catalog.Execute(r.Context())
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: encodeCatalog(templates)})
}

// instances dispatches collection read and creation on the exact instance path.
func (h Handler) instances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listInstances(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
	}
}

// listInstances returns every authorized instance view, using an empty array when none exist.
func (h Handler) listInstances(w http.ResponseWriter, r *http.Request) {
	r, ok := h.prepareRead(w, r)
	if !ok {
		return
	}
	instances, err := h.ListInstances.Execute(r.Context())
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: encodeInstances(instances)})
}

// create inserts an unpublished instance for a catalog template version.
func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	r, ok := h.prepareMutation(w, r, true)
	if !ok {
		return
	}
	in, err := parseCreate(r)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	instance, err := h.Create.Execute(r.Context(), in)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusCreated, envelope{Status: "success", Data: encodeInstance(instance)})
}

// readInstance returns an authorized detached draft after cookie authentication.
func (h Handler) readInstance(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := parseResourceID(id); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok := h.prepareRead(w, r)
	if !ok {
		return
	}
	instance, err := h.Instance.Execute(r.Context(), id)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: encodeInstance(instance)})
}

// replace performs an exact-version full draft replacement.
func (h Handler) replace(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPut {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := parseResourceID(id); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok := h.prepareMutation(w, r, true)
	if !ok {
		return
	}
	in, err := parseReplace(r, id)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	instance, err := h.Replace.Execute(r.Context(), in)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: encodeInstance(instance)})
}

// validate records an exact-version assessment, including blocking findings.
func (h Handler) validate(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := parseResourceID(id); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok := h.prepareMutation(w, r, true)
	if !ok {
		return
	}
	in, err := parseValidate(r, id)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	result, err := h.Validate.Execute(r.Context(), in)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusCreated, envelope{Status: "success", Data: encodeValidation(result)})
}

// publish commits a new revision or returns the original replayed result.
func (h Handler) publish(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := parseResourceID(id); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok := h.prepareMutation(w, r, true)
	if !ok {
		return
	}
	in, err := parsePublish(r, id)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	result, err := h.Publish.Execute(r.Context(), in)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	respond(w, status, envelope{Status: "success", Data: encodeRevision(result.Revision)})
}

// readRevision returns one authorized immutable revision.
func (h Handler) readRevision(w http.ResponseWriter, r *http.Request, instanceID, revisionID string) {
	if r.Method != http.MethodGet {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := parseResourceID(instanceID); err != nil {
		h.sendError(w, r, err)
		return
	}
	if err := parseResourceID(revisionID); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok := h.prepareRead(w, r)
	if !ok {
		return
	}
	revision, err := h.Revision.Execute(r.Context(), instanceID, revisionID)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: encodeRevision(revision)})
}
