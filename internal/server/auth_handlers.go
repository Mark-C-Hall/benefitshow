package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Mark-C-Hall/benefitshow/internal/auth"
)

const (
	sessionCookieName   = "session"
	stateCookieName     = "oauth_state"
	stateCookiePath     = "/auth/discord/callback"
	stateCookieMaxAge   = 600            // 10 minutes
	sessionCookieMaxAge = 30 * 24 * 3600 // 30 days
	callbackTimeout     = 10 * time.Second
)

func (s *Server) handleAuthStart(w http.ResponseWriter, r *http.Request) {
	state, err := auth.NewState()
	if err != nil {
		log.Printf("auth start: new state: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	signed := auth.SignState(s.cfg.SessionSecret, state)

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    signed,
		Path:     stateCookiePath,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   stateCookieMaxAge,
	})
	http.Redirect(w, r, s.auth.AuthorizeURL(state), http.StatusFound)
}

func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		http.Error(w, "missing state cookie", http.StatusBadRequest)
		return
	}
	wantState, ok := auth.VerifyState(s.cfg.SessionSecret, stateCookie.Value)
	if !ok {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != wantState {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}

	// Clear the state cookie regardless of what comes next.
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     stateCookiePath,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), callbackTimeout)
	defer cancel()

	token, err := s.auth.Exchange(ctx, code)
	if err != nil {
		log.Printf("auth callback: token exchange: %v", err)
		http.Error(w, "could not exchange code", http.StatusBadGateway)
		return
	}

	user, err := s.auth.Me(ctx, token)
	if err != nil {
		log.Printf("auth callback: me: %v", err)
		http.Error(w, "could not fetch profile", http.StatusBadGateway)
		return
	}

	inGuild, err := s.auth.InGuild(ctx, token)
	if err != nil {
		log.Printf("auth callback: guilds: %v", err)
		http.Error(w, "could not fetch guilds", http.StatusBadGateway)
		return
	}
	if !inGuild {
		log.Printf("auth callback: rejecting user %s (%s): not in configured guild", user.ID, displayName(user))
		renderErrorPage(w, http.StatusForbidden,
			"Not a member",
			"You are not a member of the Discord server that's allowed to vote in this election.")
		return
	}

	sessionToken, err := auth.NewSessionToken()
	if err != nil {
		log.Printf("auth callback: new session token: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	uid, err := s.store.LoginUser(user.ID, displayName(user), sessionToken)
	if err != nil {
		log.Printf("auth callback: login user: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	log.Printf("login: user %d (%s)", uid, displayName(user))

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   sessionCookieMaxAge,
	})
	http.Redirect(w, r, "/vote", http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	if err := s.store.ClearSessionToken(uid); err != nil {
		log.Printf("logout: clear session: %v", err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func displayName(u auth.DiscordUser) string {
	if u.GlobalName != "" {
		return u.GlobalName
	}
	return u.Username
}
