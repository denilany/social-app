// backend/pkg/handlers/auth.go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ripple/pkg/auth"
	"ripple/pkg/constants"
	"ripple/pkg/models"
	"ripple/pkg/utils"
)

type AuthHandler struct {
	userRepo       *models.UserRepository
	followRepo     *models.FollowRepository
	postRepo       *models.PostRepository
	sessionManager *auth.SessionManager
}

func NewAuthHandler(userRepo *models.UserRepository, followRepo *models.FollowRepository, postRepo *models.PostRepository, sessionManager *auth.SessionManager) *AuthHandler {
	return &AuthHandler{
		userRepo:       userRepo,
		followRepo:     followRepo,
		postRepo:       postRepo,
		sessionManager: sessionManager,
	}
}

type RegisterRequest struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	DateOfBirth string  `json:"date_of_birth"`
	Nickname    *string `json:"nickname"`
	AboutMe     *string `json:"about_me"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	User         *models.UserResponse `json:"user"`
	Message      string               `json:"message"`
	SessionToken string               `json:"session_token,omitempty"`
}

func (ah *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Registration JSON decode error: %v", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	log.Printf("Registration attempt for email: %s", req.Email)

	// Validate input
	errors := ah.validateRegisterRequest(&req)
	if errors.HasErrors() {
		log.Printf("Registration validation failed for %s: %v", req.Email, errors)
		utils.WriteValidationErrorResponse(w, errors)
		return
	}

	// Check if email already exists
	exists, err := ah.userRepo.EmailExists(req.Email)
	if err != nil {
		log.Printf("Error checking email existence for %s: %v", req.Email, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}
	if exists {
		log.Printf("Registration failed - email already exists: %s", req.Email)
		utils.WriteErrorResponse(w, http.StatusConflict, constants.ErrEmailExists)
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Password hashing failed for %s: %v", req.Email, err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create user
	createUserReq := &models.CreateUserRequest{
		Email:       strings.ToLower(strings.TrimSpace(req.Email)),
		Password:    req.Password,
		FirstName:   strings.TrimSpace(req.FirstName),
		LastName:    strings.TrimSpace(req.LastName),
		DateOfBirth: req.DateOfBirth,
		Nickname:    req.Nickname,
		AboutMe:     req.AboutMe,
	}

	user, err := ah.userRepo.CreateUser(createUserReq, passwordHash)
	if err != nil {
		log.Printf("User creation failed for %s: %v", req.Email, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	log.Printf("User created successfully: ID=%d, Email=%s", user.ID, user.Email)

	// Create session
	session, err := ah.sessionManager.CreateSession(user.ID)
	if err != nil {
		log.Printf("Session creation failed for user ID %d: %v", user.ID, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	log.Printf("Session created successfully: ID=%s, UserID=%d, Expires=%v",
		session.ID, session.UserID, session.ExpiresAt)

	// Set session cookie
	ah.setSessionCookie(w, session.ID, session.ExpiresAt)
	log.Printf("Session cookie set for user ID %d", user.ID)

	// Return success response
	response := &AuthResponse{
		User:    user.ToResponse(),
		Message: "Registration successful",
	}

	utils.WriteSuccessResponse(w, http.StatusCreated, response)
	log.Printf("Registration completed successfully for user ID %d", user.ID)
}

func (ah *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Login JSON decode error: %v", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	log.Printf("Login attempt for email: %s", req.Email)

	// Validate input
	errors := ah.validateLoginRequest(&req)
	if errors.HasErrors() {
		log.Printf("Login validation failed for %s: %v", req.Email, errors)
		utils.WriteValidationErrorResponse(w, errors)
		return
	}

	// Get user by email
	user, err := ah.userRepo.GetUserByEmail(strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrUserNotFound) {
			log.Printf("Login failed - user not found: %s", req.Email)
			utils.WriteErrorResponse(w, http.StatusUnauthorized, constants.ErrInvalidCredentials)
			return
		}
		log.Printf("Error retrieving user %s: %v", req.Email, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	// Check password
	if err := auth.CheckPassword(req.Password, user.PasswordHash); err != nil {
		log.Printf("Login failed - invalid password for %s", req.Email)
		utils.WriteErrorResponse(w, http.StatusUnauthorized, constants.ErrInvalidCredentials)
		return
	}

	// Create session
	session, err := ah.sessionManager.CreateSession(user.ID)
	if err != nil {
		log.Printf("Session creation failed during login for user ID %d: %v", user.ID, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	log.Printf("Login session created: ID=%s, UserID=%d", session.ID, session.UserID)

	// Set session cookie
	ah.setSessionCookie(w, session.ID, session.ExpiresAt)

	// Return success response
	response := &AuthResponse{
		User:         user.ToResponse(),
		Message:      "Login successful",
		SessionToken: session.ID,
	}

	utils.WriteSuccessResponse(w, http.StatusOK, response)
	log.Printf("Login completed successfully for user ID %d", user.ID)
}

func (ah *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get session cookie
	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Printf("Logout failed - no session cookie: %v", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, "No active session")
		return
	}

	log.Printf("Logout attempt for session: %s", cookie.Value)

	// Delete session from database
	if err := ah.sessionManager.DeleteSession(cookie.Value); err != nil {
		log.Printf("Failed to delete session %s: %v", cookie.Value, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	// Clear session cookie
	ah.clearSessionCookie(w)

	utils.WriteSuccessResponse(w, http.StatusOK, map[string]string{
		"message": "Logout successful",
	})
	log.Printf("Logout completed for session: %s", cookie.Value)
}

func (ah *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		log.Printf("GetProfile failed - user ID not in context: %v", err)
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	log.Printf("GetProfile request for user ID: %d", userID)

	// Get user from database
	user, err := ah.userRepo.GetUserByID(userID)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrUserNotFound) {
			log.Printf("GetProfile failed - user not found: %d", userID)
			utils.WriteErrorResponse(w, http.StatusNotFound, "User not found")
			return
		}
		log.Printf("GetProfile database error for user ID %d: %v", userID, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	// Get follow stats
	followStats, err := ah.followRepo.GetFollowStats(userID)
	if err != nil {
		log.Printf("GetProfile - failed to get follow stats for user ID %d: %v", userID, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	// Get post count
	postCount, err := ah.postRepo.GetPostCount(userID)
	if err != nil {
		log.Printf("GetProfile - failed to get post count for user ID %d: %v", userID, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	// Create profile response with stats (isFollowing is false for own profile)
	profileResponse := user.ToProfileResponse(
		followStats.FollowersCount,
		followStats.FollowingCount,
		postCount,
		false, // User viewing their own profile
	)

	utils.WriteSuccessResponse(w, http.StatusOK, profileResponse)
	log.Printf("GetProfile completed for user ID: %d", userID)
}

func (ah *AuthHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get current user ID from context
	currentUserID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		log.Printf("GetUserProfile failed - user ID not in context: %v", err)
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Get target user ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "User ID required")
		return
	}

	targetUserID, err := strconv.Atoi(pathParts[3])
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	log.Printf("GetUserProfile request for target user ID: %d by user ID: %d", targetUserID, currentUserID)

	// Get target user from database
	user, err := ah.userRepo.GetUserByID(targetUserID)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrUserNotFound) {
			log.Printf("GetUserProfile failed - user not found: %d", targetUserID)
			utils.WriteErrorResponse(w, http.StatusNotFound, "User not found")
			return
		}
		log.Printf("GetUserProfile database error for user ID %d: %v", targetUserID, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	// Get follow stats
	followStats, err := ah.followRepo.GetFollowStats(targetUserID)
	if err != nil {
		log.Printf("GetUserProfile - failed to get follow stats for user ID %d: %v", targetUserID, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	// Get post count
	postCount, err := ah.postRepo.GetPostCount(targetUserID)
	if err != nil {
		log.Printf("GetUserProfile - failed to get post count for user ID %d: %v", targetUserID, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	// Check if current user is following target user
	isFollowing := false
	if currentUserID != targetUserID {
		followStatus, err := ah.followRepo.GetFollowRelationshipStatus(currentUserID, targetUserID)
		if err != nil {
			log.Printf("GetUserProfile - failed to get follow status: %v", err)
			// Don't fail the request, just set isFollowing to false
		} else {
			isFollowing = (followStatus == constants.FollowStatusAccepted)
		}
	}

	// Create profile response with stats
	profileResponse := user.ToProfileResponse(
		followStats.FollowersCount,
		followStats.FollowingCount,
		postCount,
		isFollowing,
	)

	utils.WriteSuccessResponse(w, http.StatusOK, profileResponse)
	log.Printf("GetUserProfile completed for target user ID: %d", targetUserID)
}

func (ah *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get user ID from context
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	log.Printf("UpdateProfile request for user ID: %d", userID)

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		log.Printf("UpdateProfile JSON decode error for user ID %d: %v", userID, err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Validate and sanitize updates
	allowedFields := map[string]bool{
		"first_name":  true,
		"last_name":   true,
		"nickname":    true,
		"about_me":    true,
		"is_public":   true,
		"avatar_path": true,
		"cover_path":  true,
	}

	validUpdates := make(map[string]interface{})
	for field, value := range updates {
		if allowedFields[field] {
			validUpdates[field] = value
		}
	}

	if len(validUpdates) == 0 {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "No valid fields to update")
		return
	}

	log.Printf("UpdateProfile valid updates for user ID %d: %v", userID, validUpdates)

	// Update user profile
	if err := ah.userRepo.UpdateProfile(userID, validUpdates); err != nil {
		log.Printf("UpdateProfile database error for user ID %d: %v", userID, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	// Get updated user
	user, err := ah.userRepo.GetUserByID(userID)
	if err != nil {
		log.Printf("UpdateProfile - failed to retrieve updated user %d: %v", userID, err)
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	utils.WriteSuccessResponse(w, http.StatusOK, user.ToResponse())
	log.Printf("UpdateProfile completed for user ID: %d", userID)
}

func (ah *AuthHandler) validateRegisterRequest(req *RegisterRequest) utils.ValidationErrors {
	var errors utils.ValidationErrors

	// Validate email
	if err := utils.ValidateEmail(req.Email); err != nil {
		errors = append(errors, *err)
	}

	// Validate password
	if err := utils.ValidatePassword(req.Password); err != nil {
		errors = append(errors, *err)
	}

	// Validate first name
	if err := utils.ValidateName(req.FirstName, "first_name"); err != nil {
		errors = append(errors, *err)
	}

	// Validate last name
	if err := utils.ValidateName(req.LastName, "last_name"); err != nil {
		errors = append(errors, *err)
	}

	// Validate date of birth
	if err := utils.ValidateDateOfBirth(req.DateOfBirth); err != nil {
		errors = append(errors, *err)
	}

	// Validate optional fields
	if req.Nickname != nil {
		if err := utils.ValidateOptionalString(*req.Nickname, 50, "nickname"); err != nil {
			errors = append(errors, *err)
		}
	}

	if req.AboutMe != nil {
		if err := utils.ValidateOptionalString(*req.AboutMe, 500, "about_me"); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

func (ah *AuthHandler) validateLoginRequest(req *LoginRequest) utils.ValidationErrors {
	var errors utils.ValidationErrors

	// Validate email
	if err := utils.ValidateEmail(req.Email); err != nil {
		errors = append(errors, *err)
	}

	// Validate password
	if err := utils.ValidateRequired(req.Password, "password"); err != nil {
		errors = append(errors, *err)
	}

	return errors
}

func (ah *AuthHandler) setSessionCookie(w http.ResponseWriter, sessionID string, expiresAt time.Time) {
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
	http.SetCookie(w, cookie)
	log.Printf("Session cookie set: Name=%s, Value=%s, Expires=%v",
		cookie.Name, sessionID, cookie.Expires)
}

func (ah *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
	http.SetCookie(w, cookie)
	log.Printf("Session cookie cleared")
}

// SearchUsers searches for users by name or email
func (ah *AuthHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Get search query from URL parameters
	query := r.URL.Query().Get("q")
	if strings.TrimSpace(query) == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Search query is required")
		return
	}

	// Parse pagination parameters
	limit := 20
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Search users
	users, err := ah.userRepo.SearchUsers(query, limit, offset)
	if err != nil {
		utils.WriteInternalErrorResponse(w, err)
		return
	}

	// Convert to response format with follow status
	type SearchUserResponse struct {
		*models.UserResponse
		IsFollowing  bool   `json:"is_following"`
		FollowStatus string `json:"follow_status"`
	}

	var userResponses []*SearchUserResponse
	for _, user := range users {
		if user.ID != userID { // Exclude current user from search results
			// Check follow status
			followStatus, err := ah.followRepo.GetFollowRelationshipStatus(userID, user.ID)
			if err != nil {
				log.Printf("SearchUsers - failed to get follow status for user %d: %v", user.ID, err)
				followStatus = "" // Default to no relationship
			}

			userResponse := &SearchUserResponse{
				UserResponse: user.ToResponse(),
				IsFollowing:  followStatus == constants.FollowStatusAccepted,
				FollowStatus: followStatus,
			}
			userResponses = append(userResponses, userResponse)
		}
	}

	utils.WriteSuccessResponse(w, http.StatusOK, map[string]any{
		"users":  userResponses,
		"query":  query,
		"limit":  limit,
		"offset": offset,
		"count":  len(userResponses),
	})
}
