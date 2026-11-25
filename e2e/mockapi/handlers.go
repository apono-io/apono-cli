package mockapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const newCredentialsStatus = "new"

// queryValues returns every value of a query parameter, accepting both repeated keys and
// comma-separated lists.
func queryValues(r *http.Request, key string) []string {
	var values []string
	for _, v := range r.URL.Query()[key] {
		for _, part := range strings.Split(v, ",") {
			if part != "" {
				values = append(values, part)
			}
		}
	}
	return values
}

func matches(filter []string, value string) bool {
	return len(filter) == 0 || slices.Contains(filter, value)
}

func notFound(w http.ResponseWriter, what, id string) {
	writeJSON(w, http.StatusNotFound, map[string]any{"status": http.StatusNotFound, "message": fmt.Sprintf("%s %s not found", what, id)})
}

func (s *Server) listAccessSessions(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	integrationIDs := queryValues(r, "integration_id")
	bundleIDs := queryValues(r, "bundle_id")
	requestIDs := queryValues(r, "request_id")

	sessions := []clientapi.AccessSessionClientModel{}
	for _, session := range s.state.Sessions {
		if matches(integrationIDs, session.Model.Integration.Id) &&
			matches(bundleIDs, session.BundleID) &&
			matches(requestIDs, session.RequestID) {
			sessions = append(sessions, session.Model)
		}
	}

	writeJSON(w, http.StatusOK, clientapi.PaginatedClientResponseModelAccessSessionClientModel{
		Data:       page(sessions, r),
		Pagination: pagination(len(sessions)),
	})
}

func (s *Server) getAccessSession(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.state.Session(r.PathValue("id"))
	if session == nil {
		notFound(w, "access session", r.PathValue("id"))
		return
	}

	if session.resetCalled && session.Model.Credentials.IsSet() {
		if session.ResetPolls > 0 {
			session.ResetPolls--
		} else {
			session.Model.Credentials.Get().Status = newCredentialsStatus
		}
	}

	writeJSON(w, http.StatusOK, session.Model)
}

func (s *Server) getAccessDetails(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.state.Session(r.PathValue("id"))
	if session == nil {
		notFound(w, "access session", r.PathValue("id"))
		return
	}

	details := session.Details
	if session.resetCalled && session.DetailsAfterReset != nil {
		details = *session.DetailsAfterReset
	}

	writeJSON(w, http.StatusOK, details)
}

func (s *Server) resetCredentials(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.state.Session(r.PathValue("id"))
	if session == nil {
		notFound(w, "access session", r.PathValue("id"))
		return
	}

	session.resetCalled = true
	writeJSON(w, http.StatusOK, clientapi.MessageResponse{Message: "credentials reset started"})
}

func (s *Server) listAccessRequests(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	requests := []clientapi.AccessRequestClientModel{}
	// Newest first, like the real backend.
	for i := len(s.state.Requests) - 1; i >= 0; i-- {
		requests = append(requests, *s.state.Requests[i])
	}

	writeJSON(w, http.StatusOK, clientapi.PaginatedClientResponseModelAccessRequestClientModel{
		Data:       page(requests, r),
		Pagination: pagination(len(requests)),
	})
}

func (s *Server) getAccessRequest(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	request := s.state.Request(r.PathValue("id"))
	if request == nil {
		notFound(w, "access request", r.PathValue("id"))
		return
	}

	writeJSON(w, http.StatusOK, request)
}

// decodeCreateRequest decodes the body with the generated model, whose UnmarshalJSON rejects
// payloads that miss required properties, so a CLI change that breaks the contract fails here.
func decodeCreateRequest(w http.ResponseWriter, r *http.Request) (*clientapi.CreateAccessRequestClientModel, bool) {
	body, _ := io.ReadAll(r.Body)
	var req clientapi.CreateAccessRequestClientModel
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": "invalid request body: " + err.Error()})
		return nil, false
	}
	return &req, true
}

func (s *Server) dryRunAccessRequest(w http.ResponseWriter, r *http.Request) {
	if _, ok := decodeCreateRequest(w, r); !ok {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, s.state.DryRun)
}

func (s *Server) createAccessRequest(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeCreateRequest(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	integration := s.state.Integrations[0]
	if len(req.FilterIntegrationIds) > 0 {
		for _, candidate := range s.state.Integrations {
			if candidate.Id == req.FilterIntegrationIds[0] {
				integration = candidate
			}
		}
	}

	s.nextReqSeq++
	created := Request(fmt.Sprintf("req-new-%d", s.nextReqSeq), s.state.NewRequestStatus, integration)
	created.Justification = req.Justification
	created.DurationInSec = req.DurationInSec
	created.CustomFields = req.CustomFields
	s.state.Requests = append(s.state.Requests, created)

	writeJSON(w, http.StatusOK, clientapi.AccessRequestSubmittedClientResponse{RequestIds: []string{created.Id}})
}

func (s *Server) revokeAccessRequest(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	request := s.state.Request(r.PathValue("id"))
	if request == nil {
		notFound(w, "access request", r.PathValue("id"))
		return
	}

	revokedAt := float64(time.Now().Unix())
	request.Status.Status = "Revoked"
	request.RevocationTime = *clientapi.NewNullableFloat64(&revokedAt)

	writeJSON(w, http.StatusOK, clientapi.MessageResponse{Message: "revoking"})
}

func (s *Server) getAccessGroupUnits(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	units := s.state.AccessUnits[r.PathValue("id")]
	if units == nil {
		units = []clientapi.AccessUnitClientModel{}
	}

	writeJSON(w, http.StatusOK, clientapi.PaginatedClientResponseModelAccessUnitClientModel{
		Data:       page(units, r),
		Pagination: pagination(len(units)),
	})
}

func (s *Server) listIntegrations(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	writeJSON(w, http.StatusOK, clientapi.PaginatedClientResponseModelIntegrationClientModel{
		Data:       page(s.state.Integrations, r),
		Pagination: pagination(len(s.state.Integrations)),
	})
}

func (s *Server) listBundles(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	search := strings.ToLower(r.URL.Query().Get("search"))
	bundles := []clientapi.BundleClientModel{}
	for _, bundle := range s.state.Bundles {
		if strings.Contains(strings.ToLower(bundle.Name), search) {
			bundles = append(bundles, bundle)
		}
	}

	writeJSON(w, http.StatusOK, clientapi.PaginatedClientResponseModelBundleClientModel{
		Data:       page(bundles, r),
		Pagination: pagination(len(bundles)),
	})
}

func (s *Server) listResourceTypes(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceTypes := s.state.ResourceTypes[r.URL.Query().Get("integration-id")]
	if resourceTypes == nil {
		resourceTypes = []clientapi.ResourceTypeClientModel{}
	}

	writeJSON(w, http.StatusOK, clientapi.PaginatedClientResponseModelResourceTypeClientModel{
		Data:       page(resourceTypes, r),
		Pagination: pagination(len(resourceTypes)),
	})
}

func (s *Server) listResources(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	integrationID := r.URL.Query().Get("integration-id")
	resourceTypeIDs := queryValues(r, "resource-type-id")
	sourceIDs := queryValues(r, "source-id")

	resources := []clientapi.ResourceClientModel{}
	for _, resource := range s.state.Resources {
		if resource.Integration.Id == integrationID &&
			matches(resourceTypeIDs, resource.Type.Id) &&
			matches(sourceIDs, resource.SourceId) {
			resources = append(resources, resource)
		}
	}

	writeJSON(w, http.StatusOK, clientapi.PaginatedClientResponseModelResourceClientModel{
		Data:       page(resources, r),
		Pagination: pagination(len(resources)),
	})
}

func (s *Server) listPermissions(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	permissions := s.state.Permissions[r.URL.Query().Get("resource-type-id")]
	if permissions == nil {
		permissions = []clientapi.PermissionClientModel{}
	}

	writeJSON(w, http.StatusOK, clientapi.PaginatedClientResponseModelPermissionClientModel{
		Data:       page(permissions, r),
		Pagination: pagination(len(permissions)),
	})
}
