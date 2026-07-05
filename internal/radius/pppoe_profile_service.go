package radius

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/Alg0rix/radius-go/internal/domain"
)

type PPPoEProfileService struct {
	repo *Repository
}

func NewPPPoEProfileService(repo *Repository) *PPPoEProfileService {
	return &PPPoEProfileService{repo: repo}
}

func (p *PPPoEProfileService) ListProfiles(ctx context.Context) ([]domain.PPPoEProfile, error) {
	return p.repo.ListPPPoEProfiles(ctx)
}

func (p *PPPoEProfileService) GetProfile(ctx context.Context, id string) (*domain.PPPoEProfile, error) {
	profile, err := p.repo.GetPPPoEProfile(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("pppoe profile not found")
		}
		return nil, err
	}
	return profile, nil
}

func (p *PPPoEProfileService) CreateProfile(ctx context.Context, req domain.CreatePPPoEProfileRequest) (domain.PPPoEProfile, error) {
	profile := domain.PPPoEProfile{
		ID:                uuid.New().String(),
		Name:              req.Name,
		Description:       req.Description,
		FramedIPPool:      req.FramedIPPool,
		FramedIPNetmask:   req.FramedIPNetmask,
		PrimaryDNS:        req.PrimaryDNS,
		SecondaryDNS:      req.SecondaryDNS,
		PPPCompression:    req.PPPCompression,
		MTU:               req.MTU,
		MRU:               req.MRU,
		KeepaliveInterval: req.KeepaliveInterval,
		RateLimit:         req.RateLimit,
		BandwidthMaxUp:    req.BandwidthMaxUp,
		BandwidthMaxDown:  req.BandwidthMaxDown,
		SessionTimeout:    req.SessionTimeout,
		IdleTimeout:       req.IdleTimeout,
		MaxTotalOctets:    req.MaxTotalOctets,
		Enabled:           true,
	}
	if err := p.repo.CreatePPPoEProfile(ctx, profile); err != nil {
		return domain.PPPoEProfile{}, fmt.Errorf("create pppoe profile: %w", err)
	}
	return profile, nil
}

func (p *PPPoEProfileService) UpdateProfile(ctx context.Context, id string, req domain.UpdatePPPoEProfileRequest) (domain.PPPoEProfile, error) {
	profile, err := p.repo.GetPPPoEProfile(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.PPPoEProfile{}, fmt.Errorf("pppoe profile not found")
		}
		return domain.PPPoEProfile{}, err
	}
	if req.Name != nil {
		profile.Name = *req.Name
	}
	if req.Description != nil {
		profile.Description = *req.Description
	}
	if req.FramedIPPool != nil {
		profile.FramedIPPool = *req.FramedIPPool
	}
	if req.FramedIPNetmask != nil {
		profile.FramedIPNetmask = *req.FramedIPNetmask
	}
	if req.PrimaryDNS != nil {
		profile.PrimaryDNS = *req.PrimaryDNS
	}
	if req.SecondaryDNS != nil {
		profile.SecondaryDNS = *req.SecondaryDNS
	}
	if req.PPPCompression != nil {
		profile.PPPCompression = *req.PPPCompression
	}
	if req.MTU != nil {
		profile.MTU = *req.MTU
	}
	if req.MRU != nil {
		profile.MRU = *req.MRU
	}
	if req.KeepaliveInterval != nil {
		profile.KeepaliveInterval = *req.KeepaliveInterval
	}
	if req.RateLimit != nil {
		profile.RateLimit = *req.RateLimit
	}
	if req.BandwidthMaxUp != nil {
		profile.BandwidthMaxUp = *req.BandwidthMaxUp
	}
	if req.BandwidthMaxDown != nil {
		profile.BandwidthMaxDown = *req.BandwidthMaxDown
	}
	if req.SessionTimeout != nil {
		profile.SessionTimeout = *req.SessionTimeout
	}
	if req.IdleTimeout != nil {
		profile.IdleTimeout = *req.IdleTimeout
	}
	if req.MaxTotalOctets != nil {
		profile.MaxTotalOctets = *req.MaxTotalOctets
	}
	if req.Enabled != nil {
		profile.Enabled = *req.Enabled
	}
	if err := p.repo.UpdatePPPoEProfile(ctx, *profile); err != nil {
		return domain.PPPoEProfile{}, fmt.Errorf("update pppoe profile: %w", err)
	}
	return *profile, nil
}

func (p *PPPoEProfileService) DeleteProfile(ctx context.Context, id string) error {
	if err := p.repo.DeletePPPoEProfile(ctx, id); err != nil {
		return fmt.Errorf("delete pppoe profile: %w", err)
	}
	return nil
}

func (p *PPPoEProfileService) ValidateAssignment(ctx context.Context, profileID string) error {
	if profileID == "" {
		return nil
	}
	profile, err := p.repo.GetPPPoEProfile(ctx, profileID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("pppoe_profile_id not found")
		}
		return fmt.Errorf("lookup pppoe profile: %w", err)
	}
	if !profile.Enabled {
		return fmt.Errorf("pppoe profile is disabled")
	}
	return nil
}
