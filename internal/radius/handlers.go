package radius

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/Alg0rix/radius-go/internal/domain"
	"github.com/Alg0rix/radius-go/internal/runtime"
)

const requestTimeout = 5 * time.Second

func (s *Service) fail(c echo.Context, status int, code, message string, err error) error {
	if status >= 500 && err != nil {
		s.deps.Logger.Error().Err(err).Str("method", c.Request().Method).Str("path", c.Request().URL.Path).Msg("handler error")
	}
	return runtime.Fail(c, status, code, message, err)
}

// --- NAS handlers ---

// HandleListNAS godoc
//
//	@Summary		List NAS devices
//	@Description	List all configured Network Access Servers
//	@Tags			NAS
//	@Security		InternalSecret
//	@Produce		json
//	@Success		200	{object}	runtime.Envelope{data=[]domain.NAS}
//	@Router			/radius/nases [get]
func (s *Service) HandleListNAS(c echo.Context) error {
	return runtime.OK(c, s.ListNAS())
}

// HandleCreateNAS godoc
//
//	@Summary		Create NAS device
//	@Description	Register a new Network Access Server
//	@Tags			NAS
//	@Security		InternalSecret
//	@Accept			json
//	@Produce		json
//	@Param			nas	body		domain.CreateNASRequest	true	"NAS details"
//	@Success		201	{object}	runtime.Envelope{data=domain.NAS}
//	@Failure		400	{object}	runtime.Envelope
//	@Router			/radius/nases [post]
func (s *Service) HandleCreateNAS(c echo.Context) error {
	var req domain.CreateNASRequest
	if err := c.Bind(&req); err != nil {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err)
	}
	if !nonEmpty(req.Name) || req.IPAddress == "" || req.Secret == "" {
		return s.fail(c, http.StatusBadRequest, "bad_request", "name, ip_address, and secret are required", nil)
	}
	if !validIP(req.IPAddress) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid ip_address", nil)
	}
	req.Name = limitString(req.Name, maxNameLen)
	req.Secret = limitString(req.Secret, maxSecretLen)
	req.Description = limitString(req.Description, maxDescriptionLen)

	nas := domain.NAS{
		ID:          uuid.New().String(),
		Name:        req.Name,
		IPAddress:   req.IPAddress,
		Secret:      req.Secret,
		Description: req.Description,
		Enabled:     true,
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	if err := s.repo.CreateNAS(ctx, nas); err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "create NAS failed", err)
	}
	s.refreshFromDBAsync()
	return runtime.Created(c, nas)
}

// HandleUpdateNAS godoc
//
//	@Summary		Update NAS device
//	@Tags			NAS
//	@Security		InternalSecret
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"NAS UUID"
//	@Param			nas	body		domain.UpdateNASRequest	true	"Updated NAS fields"
//	@Success		200	{object}	runtime.Envelope{data=domain.NAS}
//	@Failure		404	{object}	runtime.Envelope
//	@Router			/radius/nases/{id} [put]
func (s *Service) HandleUpdateNAS(c echo.Context) error {
	id := c.Param("id")
	if !validUUID(id) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid id", nil)
	}
	var req domain.UpdateNASRequest
	if err := c.Bind(&req); err != nil {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err)
	}
	if req.IPAddress != "" && !validIP(req.IPAddress) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid ip_address", nil)
	}
	req.Name = limitString(req.Name, maxNameLen)
	req.Secret = limitString(req.Secret, maxSecretLen)
	req.Description = limitString(req.Description, maxDescriptionLen)

	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	existing, err := s.repo.GetNASByID(ctx, id)
	if err != nil {
		return s.fail(c, http.StatusNotFound, "not_found", "NAS not found", nil)
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.IPAddress != "" {
		existing.IPAddress = req.IPAddress
	}
	if req.Secret != "" {
		existing.Secret = req.Secret
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	if err := s.repo.UpdateNAS(ctx, *existing); err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "update NAS failed", err)
	}
	s.refreshFromDBAsync()
	return runtime.OK(c, existing)
}

// HandleDeleteNAS godoc
//
//	@Summary	Delete NAS device
//	@Tags		NAS
//	@Security	InternalSecret
//	@Produce	json
//	@Param		id	path		string	true	"NAS UUID"
//	@Success	200	{object}	runtime.Envelope
//	@Router		/radius/nases/{id} [delete]
func (s *Service) HandleDeleteNAS(c echo.Context) error {
	id := c.Param("id")
	if !validUUID(id) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid id", nil)
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	if err := s.repo.DeleteNAS(ctx, id); err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "delete NAS failed", err)
	}
	s.refreshFromDBAsync()
	return runtime.OK(c, map[string]string{"id": id})
}

// --- Session handlers ---

// HandleListSessions godoc
//
//	@Summary	List active sessions
//	@Tags		Sessions
//	@Security	InternalSecret
//	@Produce	json
//	@Success	200	{object}	runtime.Envelope{data=[]domain.RadiusSession}
//	@Router		/radius/sessions [get]
func (s *Service) HandleListSessions(c echo.Context) error {
	return runtime.OK(c, s.ListSessions())
}

// HandleStatus godoc
//
//	@Summary	Server status and counters
//	@Tags		Status
//	@Security	InternalSecret
//	@Produce	json
//	@Success	200	{object}	runtime.Envelope{data=Status}
//	@Router		/radius/status [get]
func (s *Service) HandleStatus(c echo.Context) error {
	return runtime.OK(c, s.Snapshot())
}

// HandleDisconnectUser godoc
//
//	@Summary	Disconnect user sessions (PoD)
//	@Tags		Sessions
//	@Security	InternalSecret
//	@Accept		json
//	@Produce	json
//	@Param		body	body		domain.DisconnectRequest	true	"Disconnect request"
//	@Success	200	{object}	runtime.Envelope
//	@Failure	400	{object}	runtime.Envelope
//	@Router		/radius/sessions/disconnect [post]
func (s *Service) HandleDisconnectUser(c echo.Context) error {
	var req domain.DisconnectRequest
	if err := c.Bind(&req); err != nil {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err)
	}
	if !nonEmpty(req.Username) || len(req.Username) > maxUsernameLen {
		return s.fail(c, http.StatusBadRequest, "bad_request", "username is required", nil)
	}
	req.Reason = limitString(req.Reason, maxDescriptionLen)

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	count, err := s.DisconnectUser(ctx, req.Username, req.Reason)
	if err != nil {
		return s.fail(c, http.StatusInternalServerError, "disconnect_failed", "disconnect failed", err)
	}
	return runtime.OK(c, map[string]any{
		"username":           req.Username,
		"disconnected_count": count,
	})
}

// HandleCoAChange godoc
//
//	@Summary	Change user profile via CoA
//	@Tags		Sessions
//	@Security	InternalSecret
//	@Accept		json
//	@Produce	json
//	@Param		body	body		domain.CoAChangeRequest	true	"CoA change request"
//	@Success	200	{object}	runtime.Envelope{data=CoaChangeResult}
//	@Failure	400	{object}	runtime.Envelope
//	@Router		/radius/subscribers/coa-change [post]
func (s *Service) HandleCoAChange(c echo.Context) error {
	var req domain.CoAChangeRequest
	if err := c.Bind(&req); err != nil {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err)
	}
	if !nonEmpty(req.Username) || len(req.Username) > maxUsernameLen {
		return s.fail(c, http.StatusBadRequest, "bad_request", "username is required", nil)
	}
	req.RateLimit = limitString(req.RateLimit, maxRateLimitLen)
	req.Group = limitString(req.Group, maxGroupLen)

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	result, err := s.ChangeUserProfile(ctx, req.Username, req.RateLimit, req.Group,
		req.BandwidthMaxUp, req.BandwidthMaxDown, req.MaxTotalOctets)
	if err != nil {
		return s.fail(c, http.StatusInternalServerError, "coa_failed", "CoA change failed", err)
	}
	return runtime.OK(c, result)
}

// HandleSessionCleanup godoc
//
//	@Summary	Cleanup stale sessions
//	@Tags		Sessions
//	@Security	InternalSecret
//	@Produce	json
//	@Success	200	{object}	runtime.Envelope{data=CleanupResult}
//	@Router		/radius/sessions/cleanup [post]
func (s *Service) HandleSessionCleanup(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()
	result, err := s.cleanupStaleSessions(ctx)
	if err != nil {
		return s.fail(c, http.StatusInternalServerError, "cleanup_failed", "cleanup failed", err)
	}
	return runtime.OK(c, result)
}

// HandleSessionReconcile godoc
//
//	@Summary	Reconcile sessions from DB into memory
//	@Tags		Sessions
//	@Security	InternalSecret
//	@Produce	json
//	@Success	200	{object}	runtime.Envelope
//	@Router		/radius/sessions/reconcile [post]
func (s *Service) HandleSessionReconcile(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	merged, err := s.ReconcileSessions(ctx)
	if err != nil {
		return s.fail(c, http.StatusInternalServerError, "reconcile_failed", "reconcile failed", err)
	}
	return runtime.OK(c, map[string]int{"merged": merged})
}

// Voucher HTTP handlers live in voucher_handlers.go.
// refreshFromDBAsync also lives in voucher_handlers.go (shared helper).
