package sessions

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/WilliamTrojniak/TabAppBackend/services"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	logger               *slog.Logger
	store                *redis.Client
	authExpirationTime   time.Duration
	unauthExpirationTime time.Duration
	handleError          services.HTTPErrorHandler
}

type sessionData struct {
	UserId    string
	CSRFToken string
	Ip        string
}

type Session struct {
	data sessionData
}

const (
	session_cookie = "session"
	csrf_header    = "X-CSRF-TOKEN"
	csrf_field     = "xcsrftoken"
)

var (
	safe_methods = []string{"GET", "HEAD", "OPTIONS", "TRACE"}
)

func New(store *redis.Client, authExpiryTime time.Duration, unauthExpiryTime time.Duration, h services.HTTPErrorHandler, logger *slog.Logger) *Handler {

	return &Handler{
		logger:               logger,
		store:                store,
		authExpirationTime:   authExpiryTime,
		unauthExpirationTime: unauthExpiryTime,
		handleError:          h,
	}
}

func (s *Handler) CreateSession(w http.ResponseWriter, r *http.Request, userId string) (*Session, error) {
	sessionID, err := randString(32)
	if err != nil {
		return nil, services.NewInternalServiceError(err)
	}
	csrfToken, err := randString(32)
	if err != nil {
		return nil, services.NewInternalServiceError(err)
	}
	s.logger.Debug("Creating session", "sessionId", sessionID)

	session := Session{}
	session.data = sessionData{UserId: userId, Ip: readUserIP(r), CSRFToken: csrfToken}
	jsonString, err := json.Marshal(session.data)
	if err != nil {
		return nil, services.NewInternalServiceError(err)
	}

	currentSession, err := r.Cookie(session_cookie)
	if err == nil {
		// i.e. The client has a previous session
		err := s.store.Del(r.Context(), currentSession.Value).Err()
		if err != nil {
			s.logger.Warn("Attempt to delete old session failed", "err", err)
		}
	}

	expiryTime := s.authExpirationTime
	if userId == "" {
		expiryTime = s.unauthExpirationTime
	}
	if err := s.store.Set(r.Context(), sessionID, jsonString, expiryTime).Err(); err != nil {
		s.logger.Error("Session Manager could not save session to redis")
		return nil, services.NewInternalServiceError(err)
	}

	s.createSessionCookie(w, r, sessionID, int(expiryTime.Seconds()))
	s.setCSRFHeader(w, &session)
	s.logger.Debug("Session created", "sessionId", sessionID)

	return &session, nil
}

func (s *Handler) GetSession(r *http.Request) (*Session, error) {
	sessionCookie, err := r.Cookie(session_cookie)
	if err != nil {
		return nil, services.NewUnauthenticatedServiceError(err)
	}

	sessionId := sessionCookie.Value
	jsonString, err := s.store.Get(r.Context(), sessionId).Bytes()
	if err != nil {
		return nil, services.NewUnauthenticatedServiceError(err)
	}

	session := Session{}
	err = json.Unmarshal(jsonString, &session.data)
	if err != nil {
		s.logger.Warn("Failed to parse json data from redis")
		return nil, services.NewUnauthenticatedServiceError(err)
	}

	if session.data.Ip != readUserIP(r) {
		s.logger.Debug("Attempted to access session with different ip", "stored-ip", session.data.Ip, "request-ip", readUserIP(r))
		return nil, services.NewUnauthenticatedServiceError(err)
	}

	return &session, nil
}

func (s *Handler) ClearSession(w http.ResponseWriter, r *http.Request) error {
	_, err := s.CreateSession(w, r, "") // Create an anonymous session
	if err != nil {
		return err
	}
	return nil
}

func (s *Handler) RequireCSRFHeader(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		requestToken := r.Header.Get(csrf_header)
		safeMethod := false
		for _, val := range safe_methods {
			if val == r.Method {
				safeMethod = true
				break
			}
		}

		if !safeMethod {
			// Check for an active session
			session, err := s.GetSession(r)
			if err != nil {
				s.handleError(w, services.NewServiceError(errors.New("No CSRF token to match"), http.StatusForbidden, nil))
				s.handleError(w, services.NewInternalServiceError(err))
				return
			}
			// Set the CSRF header in the response
			s.setCSRFHeader(w, session)

			if requestToken != session.data.CSRFToken {
				s.logger.Warn("CSRF Tokens did not match", "incoming-token", requestToken, "stored-token", session.data.CSRFToken)
				s.handleError(w, services.NewServiceError(errors.New("CSRF Tokens did not match"), http.StatusForbidden, nil))
				return
			}

		}
		next.ServeHTTP(w, r)

	}
}

func (s *Handler) RequireAuth(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := s.GetSession(r)
		if err != nil {
			s.handleError(w, err)
			return
		}
		if _, err := session.GetUserId(); err != nil {
			s.handleError(w, err)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (s *Session) GetUserId() (string, error) {
	if s.data.UserId == "" {
		return "", services.NewUnauthenticatedServiceError(nil)
	}
	return s.data.UserId, nil
}

func (s *Handler) setCSRFHeader(w http.ResponseWriter, session *Session) {
	w.Header().Set(csrf_header, session.data.CSRFToken)
}

func (s *Handler) createSessionCookie(w http.ResponseWriter, _ *http.Request, sessionId string, expiryTime int) {

	c := &http.Cookie{
		Name:     session_cookie,
		Value:    sessionId,
		MaxAge:   expiryTime,
		Secure:   true,
		HttpOnly: true,
		Path:     "/",
		SameSite: 4,
	}
	http.SetCookie(w, c)
}

func readUserIP(r *http.Request) string {
	addr := r.RemoteAddr
	ip := strings.Split(addr, ":")[0]
	return ip

}

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func getRequestToken(r *http.Request) string {
	token := r.Header.Get(csrf_header)
	if token != "" {
		return token
	}

	token = r.PostFormValue(csrf_field)
	if token != "" {
		return token
	}

	return ""
}
