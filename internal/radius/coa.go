package radius

import (
	"context"
	"net"
	"strings"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

// handleCoA processes inbound CoA/Disconnect requests from NASes.
// Currently we NAK all inbound requests by default.
func (s *Service) handleCoA(w radius.ResponseWriter, r *radius.Request) {
	resp := r.Response(radius.CodeCoANAK)
	w.Write(resp)
}

// DisconnectUser sends PoD (RFC 3576) to all NASes hosting active sessions
// for the given username. Returns the number of successful disconnects.
func (s *Service) DisconnectUser(ctx context.Context, username, reason string) (int, error) {
	disconnectCount := 0
	nasIPs := s.collectNASForUser(username)

	for nasAddr, secret := range nasIPs {
		pkt := radius.New(radius.CodeDisconnectRequest, []byte(secret))
		rfc2865.UserName_AddString(pkt, username)
		addMessageAuthenticator(pkt)

		resp, err := radius.Exchange(ctx, pkt, nasAddr)
		if err != nil {
			s.deps.Logger.Error().Err(err).Str("nas", nasAddr).Msg("disconnect request failed; marking sessions stopped locally")
			s.markSessionsStopped(username)
		} else if resp.Code == radius.CodeDisconnectACK {
			disconnectCount++
			s.markSessionsByNAS(username, nasAddr)
		}
	}

	return disconnectCount, nil
}

// ChangeUserProfile sends a CoA-Request to all NASes hosting active sessions
// for the given username, applying the given MikroTik and/or pfSense/OPNsense attributes.
func (s *Service) ChangeUserProfile(ctx context.Context, username, rateLimit, group string, bwUp, bwDown, maxOctets *uint32) (CoaChangeResult, error) {
	result := CoaChangeResult{}
	nasIPs := s.collectNASForUser(username)

	for nasAddr, secret := range nasIPs {
		pkt := radius.New(radius.CodeCoARequest, []byte(secret))
		rfc2865.UserName_AddString(pkt, username)
		if rateLimit != "" {
			MikrotikRateLimit_SetString(pkt, rateLimit)
		}
		if group != "" {
			MikrotikGroup_SetString(pkt, group)
		}
		if bwUp != nil {
			PfSenseBandwidthMaxUp_Set(pkt, *bwUp)
		}
		if bwDown != nil {
			PfSenseBandwidthMaxDown_Set(pkt, *bwDown)
		}
		if maxOctets != nil {
			PfSenseMaxTotalOctets_Set(pkt, *maxOctets)
		}
		addMessageAuthenticator(pkt)

		resp, err := radius.Exchange(ctx, pkt, nasAddr)
		if err != nil || resp.Code != radius.CodeCoAACK {
			result.FailedNAS = append(result.FailedNAS, nasAddr)
			continue
		}
		result.DisconnectedCount++

		s.mu.Lock()
		for id, sess := range s.sessions {
			if sess.SessionStatus == "active" && sess.Username == username && nasContains(sess.NASIP, nasAddr) {
				if rateLimit != "" {
					sess.RateLimit = rateLimit
				}
				if group != "" {
					sess.MikrotikGroup = group
				}
				if bwUp != nil {
					sess.BandwidthMaxUp = *bwUp
				}
				if bwDown != nil {
					sess.BandwidthMaxDown = *bwDown
				}
				if maxOctets != nil {
					sess.MaxTotalOctets = *maxOctets
				}
				s.sessions[id] = sess
			}
		}
		s.mu.Unlock()
	}

	return result, nil
}

// collectNASForUser returns a map of NAS UDP address → secret for all NASes
// hosting active sessions of the given user.
func (s *Service) collectNASForUser(username string) map[string]string {
	nasMap := make(map[string]string)

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sess := range s.sessions {
		if sess.Username != username || sess.SessionStatus != "active" {
			continue
		}
		host, _, err := net.SplitHostPort(sess.NASIP)
		if err != nil {
			host = sess.NASIP
		}
		if nas, ok := s.nases[host]; ok {
			nasMap[net.JoinHostPort(host, "3799")] = nas.Secret
		}
	}
	return nasMap
}

func (s *Service) markSessionsStopped(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sess := range s.sessions {
		if sess.Username == username && sess.SessionStatus == "active" {
			sess.SessionStatus = "stopped"
			now := timeNowPtr()
			sess.StopTime = now
			s.sessions[id] = sess
		}
	}
}

func (s *Service) markSessionsByNAS(username, nasAddr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sess := range s.sessions {
		if sess.Username == username && sess.SessionStatus == "active" && nasContains(sess.NASIP, nasAddr) {
			sess.SessionStatus = "stopped"
			sess.StopTime = timeNowPtr()
			s.sessions[id] = sess
		}
	}
}

// nasContains checks whether a session NASIP (host:port) matches a nasAddr (host[:port]).
func nasContains(sessNASIP, nasAddr string) bool {
	host, _, err := net.SplitHostPort(nasAddr)
	if err != nil {
		host = nasAddr
	}
	return strings.HasPrefix(sessNASIP, host)
}