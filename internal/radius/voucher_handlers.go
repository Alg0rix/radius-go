package radius

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/Alg0rix/radius-go/internal/domain"
	"github.com/Alg0rix/radius-go/internal/runtime"
)

// Voucher HTTP handlers. Swagger annotations included.

// HandleListVoucherPackages godoc
//
//	@Summary	List voucher packages
//	@Tags		VoucherPackages
//	@Security	InternalSecret
//	@Produce	json
//	@Success	200	{object}	runtime.Envelope{data=[]domain.VoucherPackage}
//	@Router		/voucher-packages [get]
func (s *Service) HandleListVoucherPackages(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	pkgs, err := s.voucher.ListPackages(ctx)
	if err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "list voucher packages failed", err)
	}
	return runtime.OK(c, pkgs)
}

// HandleCreateVoucherPackage godoc
//
//	@Summary	Create voucher package
//	@Tags		VoucherPackages
//	@Security	InternalSecret
//	@Accept		json
//	@Produce	json
//	@Param		pkg	body		domain.CreateVoucherPackageRequest	true	"Package fields"
//	@Success	201	{object}	runtime.Envelope{data=domain.VoucherPackage}
//	@Failure	400	{object}	runtime.Envelope
//	@Router		/voucher-packages [post]
func (s *Service) HandleCreateVoucherPackage(c echo.Context) error {
	var req domain.CreateVoucherPackageRequest
	if err := c.Bind(&req); err != nil {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err)
	}
	if !nonEmpty(req.Name) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "name is required", nil)
	}
	req.Name = limitString(req.Name, maxNameLen)
	req.Description = limitString(req.Description, maxDescriptionLen)
	if !validTimeLimitType(req.TimeLimitType) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid time_limit_type", nil)
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	pkg, err := s.voucher.CreatePackage(ctx, req)
	if err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "create voucher package failed", err)
	}
	return runtime.Created(c, pkg)
}

// HandleUpdateVoucherPackage godoc
//
//	@Summary	Update voucher package
//	@Tags		VoucherPackages
//	@Security	InternalSecret
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string								true	"Package UUID"
//	@Param		pkg	body		domain.UpdateVoucherPackageRequest	true	"Updated fields"
//	@Success	200	{object}	runtime.Envelope{data=domain.VoucherPackage}
//	@Failure	404	{object}	runtime.Envelope
//	@Router		/voucher-packages/{id} [put]
func (s *Service) HandleUpdateVoucherPackage(c echo.Context) error {
	id := c.Param("id")
	if !validUUID(id) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid id", nil)
	}
	var req domain.UpdateVoucherPackageRequest
	if err := c.Bind(&req); err != nil {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err)
	}
	if req.Name != nil {
		*req.Name = limitString(*req.Name, maxNameLen)
	}
	if req.Description != nil {
		*req.Description = limitString(*req.Description, maxDescriptionLen)
	}
	if req.TimeLimitType != nil && !validTimeLimitType(*req.TimeLimitType) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid time_limit_type", nil)
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	pkg, err := s.voucher.UpdatePackage(ctx, id, req)
	if err != nil {
		return s.fail(c, http.StatusNotFound, "not_found", "voucher package not found", nil)
	}
	return runtime.OK(c, pkg)
}

// HandleDeleteVoucherPackage godoc
//
//	@Summary	Delete voucher package
//	@Tags		VoucherPackages
//	@Security	InternalSecret
//	@Produce	json
//	@Param		id	path		string	true	"Package UUID"
//	@Success	200	{object}	runtime.Envelope
//	@Router		/voucher-packages/{id} [delete]
func (s *Service) HandleDeleteVoucherPackage(c echo.Context) error {
	id := c.Param("id")
	if !validUUID(id) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid id", nil)
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	if err := s.voucher.DeletePackage(ctx, id); err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "delete voucher package failed", err)
	}
	return runtime.OK(c, map[string]string{"id": id})
}

// HandleGenerateVouchers godoc
//
//	@Summary	Generate voucher(s)
//	@Tags		Vouchers
//	@Security	InternalSecret
//	@Accept		json
//	@Produce	json
//	@Param		body	body		domain.GenerateVoucherRequest	true	"Generation request"
//	@Success	201	{object}	runtime.Envelope{data=[]radius.GeneratedVoucher}
//	@Failure	400	{object}	runtime.Envelope
//	@Router		/vouchers/generate [post]
func (s *Service) HandleGenerateVouchers(c echo.Context) error {
	var req domain.GenerateVoucherRequest
	if err := c.Bind(&req); err != nil {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid JSON", err)
	}
	if !validUUID(req.PackageID) {
		return s.fail(c, http.StatusBadRequest, "bad_request", "package_id is required", nil)
	}
	if req.CodeFormat != "random" && req.CodeFormat != "custom" {
		req.CodeFormat = "random"
	}
	if req.PasswordMode != "same_as_user" && req.PasswordMode != "random" && req.PasswordMode != "custom" {
		req.PasswordMode = "same_as_user"
	}
	req.CustomCode = limitString(req.CustomCode, maxUsernameLen)
	req.CustomPassword = limitString(req.CustomPassword, maxPasswordLen)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()
	vouchers, err := s.voucher.GenerateVouchers(ctx, req)
	if err != nil {
		return s.fail(c, http.StatusInternalServerError, "generate_failed", "voucher generation failed", err)
	}
	s.refreshFromDBAsync()
	return runtime.Created(c, vouchers)
}

// HandleListVouchers godoc
//
//	@Summary	List vouchers
//	@Tags		Vouchers
//	@Security	InternalSecret
//	@Produce	json
//	@Success	200	{object}	runtime.Envelope{data=[]domain.Subscriber}
//	@Router		/vouchers [get]
func (s *Service) HandleListVouchers(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	subs, err := s.voucher.ListVouchers(ctx)
	if err != nil {
		return s.fail(c, http.StatusInternalServerError, "db_error", "list vouchers failed", err)
	}
	return runtime.OK(c, subs)
}

// HandleVoucherBalance godoc
//
//	@Summary	Get voucher remaining balance
//	@Tags		Vouchers
//	@Security	InternalSecret
//	@Produce	json
//	@Param		code	path		string	true	"Voucher code (username)"
//	@Success	200	{object}	runtime.Envelope{data=domain.VoucherBalance}
//	@Failure	404	{object}	runtime.Envelope
//	@Router		/vouchers/{code}/balance [get]
func (s *Service) HandleVoucherBalance(c echo.Context) error {
	code := c.Param("code")
	if !nonEmpty(code) || len(code) > maxUsernameLen {
		return s.fail(c, http.StatusBadRequest, "bad_request", "invalid code", nil)
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()
	bal, err := s.voucher.GetBalance(ctx, code)
	if err != nil {
		return s.fail(c, http.StatusNotFound, "not_found", "voucher not found", nil)
	}
	return runtime.OK(c, bal)
}

// refreshFromDBAsync triggers a DB refresh in a goroutine so handlers return fast.
func (s *Service) refreshFromDBAsync() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.refreshFromDB(ctx); err != nil {
			s.deps.Logger.Error().Err(err).Msg("async db refresh failed")
		}
	}()
}