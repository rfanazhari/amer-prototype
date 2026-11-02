package accesshandler

import (
	"net/http"

	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

// revokeAssignment withdraws assignment access and cancels a nonterminal execution under current HR authority.
func (h Handler) revokeAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	assignmentID, ok := assignmentSubresourceID(r.URL.Path, revocationPath)
	if !ok {
		fail(w, r, http.StatusNotFound, errs.NotFound)
		return
	}
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err)
		return
	}
	if err := requireJSONContentType(r, r.ContentLength > 0); err != nil {
		h.sendError(w, r, err)
		return
	}
	var body emptyRequest
	if err := readJSON(r, &body, true); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok = h.verifyCSRF(w, r)
	if !ok {
		return
	}
	result, err := h.RevokeAssignment.Execute(r.Context(), assignmentID)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: revocationBody{
		ID: result.ID, AccessRevokedAt: formatted(result.AccessRevokedAt),
	}})
}
