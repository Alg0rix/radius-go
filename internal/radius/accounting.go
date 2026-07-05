package radius

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/radius-go/internal/domain"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
)

func (s *Service) handleAccounting(w radius.ResponseWriter, r *radius.Request) {
	s.incAcctRequests()

	statusType, err := rfc2866.AcctStatusType_Lookup(r.Packet)
	if err != nil {
		s.deps.Logger.Error().Err(err).Msg("accounting: missing acct-status-type")
		return
	}

	username := rfc2865.UserName_GetString(r.Packet)
	sessionID := rfc2866.AcctSessionID_GetString(r.Packet)

	switch statusType {
	case rfc2866.AcctStatusType_Value_Start:
		s.handleAcctStart(r, username, sessionID)
	case rfc2866.AcctStatusType_Value_Stop:
		s.handleAcctStop(r, username, sessionID)
	case rfc2866.AcctStatusType_Value_InterimUpdate:
		s.handleAcctInterim(r, username, sessionID)
	}

	resp := r.Response(radius.CodeAccountingResponse)
	w.Write(resp)
}

func (s *Service) handleAcctStart(r *radius.Request, username, sessionID string) {
	now := time.Now()

	framedIP := ""
	if ip, err := rfc2865.FramedIPAddress_Lookup(r.Packet); err == nil {
		framedIP = ip.String()
	}

	sess := domain.RadiusSession{
		ID:             uuid.New().String(),
		SessionID:      sessionID,
		Username:       username,
		NASIP:          r.RemoteAddr.String(),
		NASID:          rfc2865.NASIdentifier_GetString(r.Packet),
		NASIdentifier:  rfc2865.NASIdentifier_GetString(r.Packet),
		FramedIP:       framedIP,
		CallingStation: rfc2865.CallingStationID_GetString(r.Packet),
		CalledStation:  rfc2865.CalledStationID_GetString(r.Packet),
		ServiceType:    domain.ServiceTypeFramed,
		SessionStatus:  domain.SessionStateActive,
		StartTime:      now,
		LastUpdate:     now,
	}

	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	go s.repo.UpsertSession(context.Background(), sess)
}

// applyAcctUpdate computes deltas vs the stored session, updates the session,
// persists it, and accumulates voucher usage. Returns true if a voucher limit
// was exceeded (caller should disconnect + disable).
func (s *Service) applyAcctUpdate(r *radius.Request, username, sessionID string, stop bool) bool {
	var inputOctets, outputOctets, sessionTime int64
	if v, err := rfc2866.AcctInputOctets_Lookup(r.Packet); err == nil {
		inputOctets = int64(v)
	}
	if v, err := rfc2866.AcctOutputOctets_Lookup(r.Packet); err == nil {
		outputOctets = int64(v)
	}
	if v, err := rfc2866.AcctSessionTime_Lookup(r.Packet); err == nil {
		sessionTime = int64(v)
	}

	var (
		deltaSec   int64
		deltaOct   int64
		userID     string
		voucherHit bool
	)

	s.mu.Lock()
	for id, sess := range s.sessions {
		if sess.SessionID != sessionID || sess.Username != username {
			continue
		}
		// Compute deltas vs previous cumulative values.
		deltaSec = sessionTime - sess.SessionTime
		if deltaSec < 0 {
			deltaSec = sessionTime
		}
		deltaOct = (inputOctets + outputOctets) - (sess.InputOctets + sess.OutputOctets)
		if deltaOct < 0 {
			deltaOct = inputOctets + outputOctets
		}

		sess.LastUpdate = time.Now()
		sess.InputOctets = inputOctets
		sess.OutputOctets = outputOctets
		sess.SessionTime = sessionTime

		if stop {
			sess.SessionStatus = domain.SessionStateStop
			now := time.Now()
			sess.StopTime = &now
		}

		s.sessions[id] = sess
		go s.repo.UpsertSession(context.Background(), sess)

		// Accumulate voucher usage in memory and detect limit breach.
		if u, ok := s.subscribers[username]; ok && u.IsVoucher {
			userID = u.ID
			u.UsageSecondsUsed += int(deltaSec)
			u.DataBytesUsed += deltaOct
			s.subscribers[username] = u

			if u.VoucherTimeLimitType == string(domain.TimeLimitUsage) &&
				u.VoucherTimeLimitSeconds > 0 &&
				u.UsageSecondsUsed >= u.VoucherTimeLimitSeconds {
				voucherHit = true
			}
			if u.VoucherDataCapBytes > 0 && u.DataBytesUsed >= u.VoucherDataCapBytes {
				voucherHit = true
			}
		}
		break // only one matching session
	}
	s.mu.Unlock()

	if userID != "" && (deltaSec > 0 || deltaOct > 0) {
		go s.repo.AddUsageDelta(context.Background(), userID, int(deltaSec), deltaOct)
	}

	return voucherHit
}

func (s *Service) handleAcctStop(r *radius.Request, username, sessionID string) {
	if s.applyAcctUpdate(r, username, sessionID, true) {
		s.disconnectAndDisableVoucher(username)
	}
}

func (s *Service) handleAcctInterim(r *radius.Request, username, sessionID string) {
	if s.applyAcctUpdate(r, username, sessionID, false) {
		s.disconnectAndDisableVoucher(username)
	}
}

// disconnectAndDisableVoucher issues a PoD for the user and disables the account.
func (s *Service) disconnectAndDisableVoucher(username string) {
	s.mu.RLock()
	user, ok := s.subscribers[username]
	s.mu.RUnlock()
	if !ok {
		return
	}
	s.deps.Logger.Info().Str("username", username).Msg("voucher limit reached; disconnecting + disabling")
	// Disable in memory.
	s.mu.Lock()
	if u, ok := s.subscribers[username]; ok {
		u.Enabled = false
		s.subscribers[username] = u
	}
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.DisconnectUser(ctx, username, "voucher limit exceeded")
		s.repo.DisableUser(context.Background(), user.ID)
	}()
}