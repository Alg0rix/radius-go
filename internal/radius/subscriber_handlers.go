package radius

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/Alg0rix/radius-go/internal/domain"
	"github.com/Alg0rix/radius-go/internal/runtime"
	"golang.org/x/crypto/bcrypt"
)

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
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err)
	}
	if !nonEmpty(req.Username) || req.Password == "" {
		return s.fail(c, http.StatusBadRequest, "bad_request", "username and password are required", nil)
	}
	if len(req.Username) > maxUsernameLen {
		return s.fail(c, http.StatusBadRequest, "bad_request", "username too long", nil)
	}
	if len(req.Password) > maxPasswordLen {
		return s.fail(c, http.StatusBadRequest, "bad_request", "password too long", nil)
	}
	if !validEmail(req.Email) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid email", nil)
	}
	if !validServiceType(req.ServiceType) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid service_type", nil)
	}
	req.FullName = limitString(req.FullName, maxNameLen)
	req.FramedIP = limitString(req.FramedIP, maxIPLen)
	req.MikrotikGroup = limitString(req.MikrotikGroup, maxGroupLen)
	req.RateLimit = limitString(req.RateLimit, maxRateLimitLen)

	var pppoeProfileID *string
	if req.PPPoEProfileID != "" {
		if !validUUID(req.PPPoEProfileID) {
			return s.fail(c, http.StatusBadRequest, "bad_request", "invalid pppoe_profile_id", nil)
		}
		ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
		if err := s.pppoe.ValidateAssignment(ctx, req.PPPoEProfileID); err != nil {
			cancel()
			return s.fail(c, http.StatusBadRequest, "bad_request", err.Error(), nil)
		}
		cancel()
		pppoeProfileID = &req.PPPoEProfileID
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return s.fail(c, http.StatusInternalServerError, "hash_error", "password hashing failed", err)
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
		PPPoEProfileID:   pppoeProfileID,
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "create user failed", err)
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
//	@Param		id		path		string						true	"User UUID"
//	@Param		user	body		domain.UpdateUserRequest	true	"Updated fields"
//	@Success	200		{object}	runtime.Envelope{data=domain.Subscriber}
//	@Failure	404		{object}	runtime.Envelope
//	@Router		/radius/subscribers/{id} [put]
func (s *Service) HandleUpdateSubscriber(c echo.Context) error {
	id := c.Param("id")
	if !validUUID(id) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid id", nil)
	}
	var req domain.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err)
	}
	if req.Username != "" && (!nonEmpty(req.Username) || len(req.Username) > maxUsernameLen) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid username", nil)
	}
	if req.Password != "" && len(req.Password) > maxPasswordLen {
		return s.fail(c, http.StatusBadRequest, "bad_request", "password too long", nil)
	}
	if !validEmail(req.Email) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid email", nil)
	}
	if !validServiceType(req.ServiceType) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid service_type", nil)
	}
	req.FullName = limitString(req.FullName, maxNameLen)
	req.FramedIP = limitString(req.FramedIP, maxIPLen)
	req.MikrotikGroup = limitString(req.MikrotikGroup, maxGroupLen)
	req.RateLimit = limitString(req.RateLimit, maxRateLimitLen)

	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	existing, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.fail(c, http.StatusNotFound, "not_found", "user not found", nil)
		}
		return s.fail(c, http.StatusInternalServerError, "db_error", "get user failed", err)
	}

	if req.Username != "" {
		existing.Username = req.Username
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return s.fail(c, http.StatusInternalServerError, "hash_error", "password hashing failed", err)
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
	if req.PPPoEProfileID != nil {
		if *req.PPPoEProfileID != "" {
			if !validUUID(*req.PPPoEProfileID) {
				return s.fail(c, http.StatusBadRequest, "bad_request", "invalid pppoe_profile_id", nil)
			}
			if err := validateMutualExclusion(existing.VoucherPackageID, req.PPPoEProfileID); err != nil {
				return s.fail(c, http.StatusBadRequest, "bad_request", err.Error(), nil)
			}
			if err := s.pppoe.ValidateAssignment(ctx, *req.PPPoEProfileID); err != nil {
				return s.fail(c, http.StatusBadRequest, "bad_request", err.Error(), nil)
			}
			existing.PPPoEProfileID = req.PPPoEProfileID
		} else {
			existing.PPPoEProfileID = nil
		}
	}

	if err := s.repo.UpdateUser(ctx, *existing); err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "update user failed", err)
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
	if !validUUID(id) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid id", nil)
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	if err := s.repo.DeleteUser(ctx, id); err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "delete user failed", err)
	}
	s.refreshFromDBAsync()
	return runtime.OK(c, map[string]string{"id": id})
}
