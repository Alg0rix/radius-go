package radius

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/your-org/radius-go/internal/domain"
	"github.com/your-org/radius-go/internal/runtime"
	"golang.org/x/crypto/bcrypt"
)

const requestTimeout = 5 * time.Second

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
		return runtime.Fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err.Error())
	}
	if req.Name == "" || req.IPAddress == "" || req.Secret == "" {
		return runtime.Fail(c, http.StatusBadRequest, "bad_request", "name, ip_address, and secret are required", nil)
	}

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
		return runtime.Fail(c, http.StatusInternalServerError, "db_error", "create NAS failed", err.Error())
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
	var req domain.UpdateNASRequest
	if err := c.Bind(&req); err != nil {
		return runtime.Fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err.Error())
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	existing, err := s.repo.GetNASByID(ctx, id)
	if err != nil {
		return runtime.Fail(c, http.StatusNotFound, "not_found", "NAS not found", err.Error())
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
		return runtime.Fail(c, http.StatusInternalServerError, "db_error", "update NAS failed", err.Error())
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
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	if err := s.repo.DeleteNAS(ctx, id); err != nil {
		return runtime.Fail(c, http.StatusInternalServerError, "db_error", "delete NAS failed", err.Error())
	}
	s.refreshFromDBAsync()
	return runtime.OK(c, map[string]string{"id": id})
}

// --- Subscriber (user) handlers ---

// HandleListSubscribers godoc
//
//	@Summary	List subscribers
//	@Tags		Subscribers
//	@Security	InternalSecret
//	@Produce	json
//	@Success	200	{object}	runtime.Envelope{data=[]domain.Subscriber}
//	@Router		/radius/subscribers [get]
func (s *Service) HandleListSubscribers(c echo.Context) error {
	return runtime.OK(c, s.ListSubscribers())
}

// HandleCreateSubscriber godoc
//
//	@Summary	Create subscriber
//	@Tags		Subscribers
//	@Security	InternalSecret
//	@Accept		json
//	@Produce	json
//	@Param		user	body		domain.CreateUserRequest	true	"Subscriber fields"
//	@Success	201	{object}	runtime.Envelope{data=domain.Subscriber}
//	@Failure	400	{object}	runtime.Envelope
//	@Router		/radius/subscribers [post]
func (s *Service) HandleCreateSubscriber(c echo.Context) error {
	var req domain.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return runtime.Fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err.Error())
	}
	if req.Username == "" || req.Password == "" {
		return runtime.Fail(c, http.StatusBadRequest, "bad_request", "username and password are required", nil)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return runtime.Fail(c, http.StatusInternalServerError, "hash_error", "password hashing failed", err.Error())
	}

	serviceType := domain.ServiceTypeFramed
	if req.ServiceType == string(domain.ServiceTypeLogin) {
		serviceType = domain.ServiceTypeLogin
	}

	user := domain.RadiusUser{
		ID:               uuid.New().String(),
		Username:         req.Username,
		PasswordHash:     string(hash),
		FullName:         req.FullName,
		Email:            req.Email,
		Enabled:          true,
		SimultaneousUse:  req.SimultaneousUse,
		SessionTimeout:   req.SessionTimeout,
		IdleTimeout:      req.IdleTimeout,
		FramedIP:         req.FramedIP,
		MikrotikGroup:    req.MikrotikGroup,
		RateLimit:        req.RateLimit,
		BandwidthMaxUp:   req.BandwidthMaxUp,
		BandwidthMaxDown: req.BandwidthMaxDown,
		MaxTotalOctets:   req.MaxTotalOctets,
		ServiceType:      serviceType,
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return runtime.Fail(c, http.StatusInternalServerError, "db_error", "create user failed", err.Error())
	}
	s.refreshFromDBAsync()
	return runtime.Created(c, domain.SubscriberFromUser(user))
}

// HandleUpdateSubscriber godoc
//
//	@Summary	Update subscriber
//	@Tags		Subscribers
//	@Security	InternalSecret
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string						true	"User UUID"
//	@Param		user	body		domain.UpdateUserRequest	true	"Updated fields"
//	@Success	200	{object}	runtime.Envelope{data=domain.Subscriber}
//	@Failure	404	{object}	runtime.Envelope
//	@Router		/radius/subscribers/{id} [put]
func (s *Service) HandleUpdateSubscriber(c echo.Context) error {
	id := c.Param("id")
	var req domain.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return runtime.Fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err.Error())
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	existing, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return runtime.Fail(c, http.StatusNotFound, "not_found", "user not found", err.Error())
	}

	if req.Username != "" {
		existing.Username = req.Username
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return runtime.Fail(c, http.StatusInternalServerError, "hash_error", "password hashing failed", err.Error())
		}
		existing.PasswordHash = string(hash)
	}
	if req.FullName != "" {
		existing.FullName = req.FullName
	}
	if req.Email != "" {
		existing.Email = req.Email
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.SimultaneousUse != nil {
		existing.SimultaneousUse = *req.SimultaneousUse
	}
	if req.SessionTimeout != nil {
		existing.SessionTimeout = *req.SessionTimeout
	}
	if req.IdleTimeout != nil {
		existing.IdleTimeout = *req.IdleTimeout
	}
	if req.FramedIP != "" {
		existing.FramedIP = req.FramedIP
	}
	if req.MikrotikGroup != "" {
		existing.MikrotikGroup = req.MikrotikGroup
	}
	if req.RateLimit != "" {
		existing.RateLimit = req.RateLimit
	}
	if req.BandwidthMaxUp != nil {
		existing.BandwidthMaxUp = *req.BandwidthMaxUp
	}
	if req.BandwidthMaxDown != nil {
		existing.BandwidthMaxDown = *req.BandwidthMaxDown
	}
	if req.MaxTotalOctets != nil {
		existing.MaxTotalOctets = *req.MaxTotalOctets
	}
	if req.ServiceType == string(domain.ServiceTypeLogin) || req.ServiceType == string(domain.ServiceTypeFramed) {
		existing.ServiceType = domain.ServiceType(req.ServiceType)
	}

	if err := s.repo.UpdateUser(ctx, *existing); err != nil {
		return runtime.Fail(c, http.StatusInternalServerError, "db_error", "update user failed", err.Error())
	}
	s.refreshFromDBAsync()
	return runtime.OK(c, domain.SubscriberFromUser(*existing))
}

// HandleDeleteSubscriber godoc
//
//	@Summary	Delete subscriber
//	@Tags		Subscribers
//	@Security	InternalSecret
//	@Produce	json
//	@Param		id	path		string	true	"User UUID"
//	@Success	200	{object}	runtime.Envelope
//	@Router		/radius/subscribers/{id} [delete]
func (s *Service) HandleDeleteSubscriber(c echo.Context) error {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	if err := s.repo.DeleteUser(ctx, id); err != nil {
		return runtime.Fail(c, http.StatusInternalServerError, "db_error", "delete user failed", err.Error())
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
		return runtime.Fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err.Error())
	}
	if req.Username == "" {
		return runtime.Fail(c, http.StatusBadRequest, "bad_request", "username is required", nil)
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	count, err := s.DisconnectUser(ctx, req.Username, req.Reason)
	if err != nil {
		return runtime.Fail(c, http.StatusInternalServerError, "disconnect_failed", "disconnect failed", err.Error())
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
		return runtime.Fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err.Error())
	}
	if req.Username == "" {
		return runtime.Fail(c, http.StatusBadRequest, "bad_request", "username is required", nil)
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	result, err := s.ChangeUserProfile(ctx, req.Username, req.RateLimit, req.Group,
		req.BandwidthMaxUp, req.BandwidthMaxDown, req.MaxTotalOctets)
	if err != nil {
		return runtime.Fail(c, http.StatusInternalServerError, "coa_failed", "CoA change failed", err.Error())
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
		return runtime.Fail(c, http.StatusInternalServerError, "cleanup_failed", "cleanup failed", err.Error())
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
		return runtime.Fail(c, http.StatusInternalServerError, "reconcile_failed", "reconcile failed", err.Error())
	}
	return runtime.OK(c, map[string]int{"merged": merged})
}

// Voucher HTTP handlers live in voucher_handlers.go.
// refreshFromDBAsync also lives in voucher_handlers.go (shared helper).
