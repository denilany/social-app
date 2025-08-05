// backend/pkg/auth/middleware.go
package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"ripple/pkg/utils"
)

type contextKey string

const UserIDKey contextKey = "userID"

func (sm *SessionManager) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("AuthMiddleware: Processing request to %s", r.URL.Path)

		var sessionID string
		var sessionSource string

		// Try to get session from Authorization header first (for desktop app)
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			sessionID = authHeader[7:]
			sessionSource = "Authorization header"
			log.Printf("AuthMiddleware: Found session token in Authorization header")
		} else {
			// Fall back to session cookie (for web app)
			cookie, err := r.Cookie("session_id")
			if err != nil {
				log.Printf("AuthMiddleware: No session cookie or Authorization header found: %v", err)
				utils.WriteErrorResponse(w, http.StatusUnauthorized, "Authentication required")
				return
			}
			sessionID = cookie.Value
			sessionSource = "session cookie"
			log.Printf("AuthMiddleware: Found session cookie: %s", sessionID)
		}

		// Validate session
		session, err := sm.GetSession(sessionID)
		if err != nil {
			log.Printf("AuthMiddleware: Session validation failed for %s (from %s): %v", sessionID, sessionSource, err)
			utils.WriteErrorResponse(w, http.StatusUnauthorized, "Invalid or expired session")
			return
		}

		log.Printf("AuthMiddleware: Session validated successfully: UserID=%d (from %s)", session.UserID, sessionSource)

		// Add user ID to request context
		ctx := context.WithValue(r.Context(), UserIDKey, session.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserIDFromContext(ctx context.Context) (int, error) {
	userID, ok := ctx.Value(UserIDKey).(int)
	if !ok {
		return 0, fmt.Errorf("user ID not found in context")
	}
	return userID, nil
}
