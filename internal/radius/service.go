package radius

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Alg0rix/radius-go/internal/config"
	"github.com/Alg0rix/radius-go/internal/domain"
	"github.com/Alg0rix/radius-go/internal/runtime"

	"layeh.com/radius"
)

type Status struct {
	StartedAt        time.Time `json:"started_at"`
	NASCount         int       `json:"nas_count"`
	SubscriberCount  int       `json:"subscriber_count"`
	ActiveSessions   int       `json:"active_sessions"`
	AuthRequests     uint64    `json:"auth_requests"`
	AuthAccepts      uint64    `json:"auth_accepts"`
	AuthRejects      uint64    `json:"auth_rejects"`
	AcctRequests     uint64    `json:"acct_requests"`
	Health           string    `json:"health"`
}

type CleanupResult struct {
	StaleSessionsCleaned int       `json:"stale_sessions_cleaned"`
	ActiveSessionsKept   int       `json:"active_sessions_kept"`
	CleanedAt            time.Time `json:"cleaned_at"`
}

type CoaChangeResult struct {
	DisconnectedCount int      `json:"disconnected_count"`
	FailedNAS         []string `json:"failed_nas"`
}

type Service struct {
	deps    *runtime.Dependencies
	repo    *Repository
	voucher *VoucherService
	config  config.Config

	startedAt time.Time

	mu          sync.RWMutex
	nases       map[string]domain.NAS
	subscribers map[string]domain.RadiusUser
	sessions    map[string]domain.RadiusSession

	authServer *radius.PacketServer
	acctServer *radius.PacketServer
	coaServer  *radius.PacketServer

	stopRefresh chan struct{}

	authRequests uint64
	authAccepts  uint64
	authRejects  uint64
	acctRequests uint64
}

func NewService(deps *runtime.Dependencies, cfg config.Config) *Service {
	s := &Service{
		deps:        deps,
		repo:        NewRepository(deps.DB),
		config:      cfg,
		startedAt:   time.Now(),
		nases:       make(map[string]domain.NAS),
		subscribers: make(map[string]domain.RadiusUser),
		sessions:    make(map[string]domain.RadiusSession),
		stopRefresh: make(chan struct{}),
	}
	s.voucher = NewVoucherService(deps.DB, s.repo)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.refreshFromDB(ctx); err != nil {
		deps.Logger.Error().Err(err).Msg("initial db refresh failed")
	}
	return s
}

func (s *Service) Start() error {
	log := s.deps.Logger

	s.authServer = &radius.PacketServer{
		Addr:         s.config.RADIUSAuthAddr,
		SecretSource: s,
		Handler:      radius.HandlerFunc(s.handleAuth),
	}
	s.acctServer = &radius.PacketServer{
		Addr:         s.config.RADIUSAcctAddr,
		SecretSource: s,
		Handler:      radius.HandlerFunc(s.handleAccounting),
	}
	if s.config.EnableCoA {
		s.coaServer = &radius.PacketServer{
			Addr:         s.config.RADIUSCoAAddr,
			SecretSource: s,
			Handler:      radius.HandlerFunc(s.handleCoA),
		}
	}

	go func() {
		log.Info().Str("addr", s.config.RADIUSAuthAddr).Msg("radius auth server starting")
		if err := s.authServer.ListenAndServe(); err != nil {
			log.Error().Err(err).Msg("auth server stopped")
		}
	}()
	go func() {
		log.Info().Str("addr", s.config.RADIUSAcctAddr).Msg("radius accounting server starting")
		if err := s.acctServer.ListenAndServe(); err != nil {
			log.Error().Err(err).Msg("accounting server stopped")
		}
	}()
	if s.coaServer != nil {
		go func() {
			log.Info().Str("addr", s.config.RADIUSCoAAddr).Msg("radius coa server starting")
			if err := s.coaServer.ListenAndServe(); err != nil {
				log.Error().Err(err).Msg("coa server stopped")
			}
		}()
	}

	go s.refreshLoop()
	go s.cleanupLoop()

	return nil
}

func (s *Service) Shutdown(ctx context.Context) {
	close(s.stopRefresh)
	s.authServer.Shutdown(ctx)
	s.acctServer.Shutdown(ctx)
	if s.coaServer != nil {
		s.coaServer.Shutdown(ctx)
	}
}

// RADIUSSecret implements radius.SecretSource.
func (s *Service) RADIUSSecret(ctx context.Context, remoteAddr net.Addr) ([]byte, error) {
	host, _, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		host = remoteAddr.String()
	}
	s.mu.RLock()
	nas, ok := s.nases[host]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("radius: unknown NAS %s", host)
	}
	return []byte(nas.Secret), nil
}

// --- refresh ---

func (s *Service) refreshLoop() {
	interval := time.Duration(s.config.DBRefreshInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := s.refreshFromDB(ctx); err != nil {
				s.deps.Logger.Error().Err(err).Msg("db refresh failed")
			}
			cancel()
		case <-s.stopRefresh:
			return
		}
	}
}

func (s *Service) refreshFromDB(ctx context.Context) error {
	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("refresh users: %w", err)
	}
	nases, err := s.repo.ListNAS(ctx)
	if err != nil {
		return fmt.Errorf("refresh nas: %w", err)
	}

	s.mu.Lock()
	s.subscribers = make(map[string]domain.RadiusUser, len(users))
	for _, u := range users {
		s.subscribers[u.Username] = u
	}
	s.nases = make(map[string]domain.NAS, len(nases))
	for _, n := range nases {
		if n.Enabled {
			s.nases[n.IPAddress] = n
		}
	}
	s.mu.Unlock()

	s.deps.Logger.Info().
		Int("users", len(users)).
		Int("nas_total", len(nases)).
		Int("nas_enabled", len(s.nases)).
		Msg("db refresh complete")
	return nil
}

// --- cleanup ---

func (s *Service) cleanupLoop() {
	interval := time.Duration(s.config.SessionCleanupPeriod) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if _, err := s.cleanupStaleSessions(ctx); err != nil {
				s.deps.Logger.Error().Err(err).Msg("session cleanup failed")
			}
			cancel()
		case <-s.stopRefresh:
			return
		}
	}
}

func (s *Service) cleanupStaleSessions(ctx context.Context) (CleanupResult, error) {
	staleDuration := time.Duration(s.config.StaleSessionTimeout) * time.Second
	cutoff := time.Now().Add(-staleDuration)

	s.mu.Lock()
	stale := 0
	active := 0
	for id, sess := range s.sessions {
		if sess.SessionStatus == domain.SessionStateActive && sess.LastUpdate.Before(cutoff) {
			sess.SessionStatus = domain.SessionStateStale
			now := time.Now()
			sess.StopTime = &now
			sess.LastUpdate = now
			s.sessions[id] = sess
			stale++
			go s.repo.UpsertSession(context.Background(), sess)
		} else {
			active++
		}
	}
	s.mu.Unlock()

	return CleanupResult{
		StaleSessionsCleaned: stale,
		ActiveSessionsKept:   active,
		CleanedAt:            time.Now(),
	}, nil
}

// --- Reconcile ---

func (s *Service) ReconcileSessions(ctx context.Context) (int, error) {
	dbSessions, err := s.repo.ListActiveSessions(ctx)
	if err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	merged := 0
	for _, dbSess := range dbSessions {
		if _, exists := s.sessions[dbSess.ID]; !exists {
			s.sessions[dbSess.ID] = dbSess
			merged++
		}
	}
	return merged, nil
}

// --- Accessors ---

func (s *Service) ListNAS() []domain.NAS {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]domain.NAS, 0, len(s.nases))
	for _, n := range s.nases {
		list = append(list, n)
	}
	return list
}

func (s *Service) ListSubscribers() []domain.Subscriber {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]domain.Subscriber, 0, len(s.subscribers))
	for _, u := range s.subscribers {
		list = append(list, domain.SubscriberFromUser(u))
	}
	return list
}

func (s *Service) ListSessions() []domain.RadiusSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]domain.RadiusSession, 0, len(s.sessions))
	for _, sess := range s.sessions {
		list = append(list, sess)
	}
	return list
}

func (s *Service) Snapshot() Status {
	s.mu.RLock()
	activeCount := 0
	for _, sess := range s.sessions {
		if sess.SessionStatus == domain.SessionStateActive {
			activeCount++
		}
	}
	nasCount := len(s.nases)
	subCount := len(s.subscribers)
	s.mu.RUnlock()

	return Status{
		StartedAt:       s.startedAt,
		NASCount:        nasCount,
		SubscriberCount: subCount,
		ActiveSessions:  activeCount,
		AuthRequests:    atomic.LoadUint64(&s.authRequests),
		AuthAccepts:     atomic.LoadUint64(&s.authAccepts),
		AuthRejects:     atomic.LoadUint64(&s.authRejects),
		AcctRequests:    atomic.LoadUint64(&s.acctRequests),
		Health:          "ok",
	}
}

// --- helpers ---

func (s *Service) incAuthRequests()  { atomic.AddUint64(&s.authRequests, 1) }
func (s *Service) incAuthAccepts()   { atomic.AddUint64(&s.authAccepts, 1) }
func (s *Service) incAuthRejects()   { atomic.AddUint64(&s.authRejects, 1) }
func (s *Service) incAcctRequests()  { atomic.AddUint64(&s.acctRequests, 1) }
