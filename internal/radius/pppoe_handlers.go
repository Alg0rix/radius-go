package radius

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/Alg0rix/radius-go/internal/domain"
	"github.com/Alg0rix/radius-go/internal/runtime"
)

func (s *Service) HandleListPPPoEProfiles(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	profiles, err := s.pppoe.ListProfiles(ctx)
	if err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "list pppoe profiles failed", err)
	}
	return runtime.OK(c, profiles)
}

func (s *Service) HandleCreatePPPoEProfile(c echo.Context) error {
	var req domain.CreatePPPoEProfileRequest
	if err := c.Bind(&req); err != nil {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err)
	}
	if !nonEmpty(req.Name) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "name is required", nil)
	}
	if err := validatePPPoEProfile(req); err != nil {
		return s.fail(c, http.StatusBadRequest, "bad_request", err.Error(), nil)
	}
	req.Name = limitString(req.Name, maxNameLen)
	req.Description = limitString(req.Description, maxDescriptionLen)
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	profile, err := s.pppoe.CreateProfile(ctx, req)
	if err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "create pppoe profile failed", err)
	}
	s.refreshFromDBAsync()
	return runtime.Created(c, profile)
}

func (s *Service) HandleGetPPPoEProfile(c echo.Context) error {
	id := c.Param("id")
	if !validUUID(id) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid id", nil)
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	profile, err := s.pppoe.GetProfile(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.fail(c, http.StatusNotFound, "not_found", "pppoe profile not found", nil)
		}
		return s.fail(c, http.StatusInternalServerError, "db_error", "get pppoe profile failed", err)
	}
	return runtime.OK(c, profile)
}

func (s *Service) HandleUpdatePPPoEProfile(c echo.Context) error {
	id := c.Param("id")
	if !validUUID(id) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid id", nil)
	}
	var req domain.UpdatePPPoEProfileRequest
	if err := c.Bind(&req); err != nil {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err)
	}
	if req.Name != nil {
		*req.Name = limitString(*req.Name, maxNameLen)
	}
	if req.Description != nil {
		*req.Description = limitString(*req.Description, maxDescriptionLen)
	}
	if err := validateUpdatePPPoEProfile(req); err != nil {
		return s.fail(c, http.StatusBadRequest, "bad_request", err.Error(), nil)
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	profile, err := s.pppoe.UpdateProfile(ctx, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.fail(c, http.StatusNotFound, "not_found", "pppoe profile not found", nil)
		}
		return s.fail(c, http.StatusInternalServerError, "db_error", "update pppoe profile failed", err)
	}
	s.refreshFromDBAsync()
	return runtime.OK(c, profile)
}

func (s *Service) HandleDeletePPPoEProfile(c echo.Context) error {
	id := c.Param("id")
	if !validUUID(id) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid id", nil)
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	if err := s.pppoe.DeleteProfile(ctx, id); err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "delete pppoe profile failed", err)
	}
	s.refreshFromDBAsync()
	return runtime.OK(c, map[string]string{"id": id})
}
