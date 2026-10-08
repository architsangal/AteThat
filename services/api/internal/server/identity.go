package server

import (
	"context"
	"net/http"
	"strings"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
)

type registerBody struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Timezone    string `json:"timezone"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	var body registerBody
	if err := decodeJSON(w, r, &body); err != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.Identity.Register(s.rpcContext(ctx, pb.IdentityService_Register_FullMethodName, ""),
		&pb.RegisterRequest{Email: body.Email, Password: body.Password, DisplayName: body.DisplayName, Timezone: body.Timezone})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 202, map[string]string{"message": result.Message})
}

func (s *Server) verifyEmail(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	var body struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.Identity.VerifyEmail(s.rpcContext(ctx, pb.IdentityService_VerifyEmail_FullMethodName, ""), &pb.VerifyEmailRequest{Token: body.Token})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, map[string]bool{"email_verified": result.EmailVerified})
}

func (s *Server) emailAction(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, action string) {
	var body struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	request := &pb.EmailRequest{Email: body.Email}
	var result *pb.GenericMessage
	var err error
	if action == "resend" {
		result, err = s.Identity.ResendVerification(s.rpcContext(ctx, pb.IdentityService_ResendVerification_FullMethodName, ""), request)
	} else {
		result, err = s.Identity.RequestPasswordReset(s.rpcContext(ctx, pb.IdentityService_RequestPasswordReset_FullMethodName, ""), request)
	}
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 202, map[string]string{"message": result.Message})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.Identity.Login(s.rpcContext(ctx, pb.IdentityService_Login_FullMethodName, ""), &pb.LoginRequest{Email: body.Email, Password: body.Password})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.setRefreshCookie(w, result.RefreshToken)
	s.sessionJSON(w, result)
}

func (s *Server) startGoogle(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	returnTo := r.URL.Query().Get("return_to")
	if returnTo != "/app" && returnTo != "/app/settings" {
		s.problem(w, requestID, 422, "VALIDATION_FAILED", "Invalid return path")
		return
	}
	result, err := s.Identity.StartGoogleSignIn(s.rpcContext(ctx, pb.IdentityService_StartGoogleSignIn_FullMethodName, ""),
		&pb.StartGoogleSignInRequest{ReturnTo: returnTo})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	http.Redirect(w, r, result.AuthorizationUrl, http.StatusFound)
}

func (s *Server) startGoogleLink(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	userID, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	result, err := s.Identity.StartGoogleLink(s.rpcContext(ctx, pb.IdentityService_StartGoogleLink_FullMethodName, userID.UserID, userID.SessionFamily), &pb.Empty{})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "atethat_oauth_mode", Value: "link", Path: "/api/v1/auth/google/callback",
		MaxAge: 600, HttpOnly: true, Secure: s.Config.CookieSecure, SameSite: http.SameSiteLaxMode})
	s.json(w, 200, map[string]string{"authorization_url": result.AuthorizationUrl})
}

func (s *Server) startGoogleDeletionCancellation(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	result, err := s.Identity.StartGoogleDeletionCancellation(s.rpcContext(ctx, pb.IdentityService_StartGoogleDeletionCancellation_FullMethodName, ""), &pb.Empty{})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "atethat_oauth_mode", Value: "deletion_cancel", Path: "/api/v1/auth/google/callback",
		MaxAge: 600, HttpOnly: true, Secure: s.Config.CookieSecure, SameSite: http.SameSiteLaxMode})
	s.json(w, http.StatusOK, map[string]string{"authorization_url": result.AuthorizationUrl})
}

func (s *Server) startGoogleDeletionRequest(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	result, err := s.Identity.StartGoogleDeletionRequest(s.rpcContext(ctx, pb.IdentityService_StartGoogleDeletionRequest_FullMethodName, user.UserID, user.SessionFamily), &pb.Empty{})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "atethat_oauth_mode", Value: "deletion_request", Path: "/api/v1/auth/google/callback",
		MaxAge: 600, HttpOnly: true, Secure: s.Config.CookieSecure, SameSite: http.SameSiteLaxMode})
	s.json(w, http.StatusOK, map[string]string{"authorization_url": result.AuthorizationUrl})
}

func (s *Server) completeGoogle(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	mode := ""
	if cookie, err := r.Cookie("atethat_oauth_mode"); err == nil {
		mode = cookie.Value
	}
	isLink := mode == "link"
	isDeletionCancellation := mode == "deletion_cancel"
	isDeletionRequest := mode == "deletion_request"
	http.SetCookie(w, &http.Cookie{Name: "atethat_oauth_mode", Path: "/api/v1/auth/google/callback", MaxAge: -1,
		HttpOnly: true, Secure: s.Config.CookieSecure, SameSite: http.SameSiteLaxMode})
	failurePath := "/login?error=google"
	if isLink {
		failurePath = "/app/settings?error=google_link"
	} else if isDeletionCancellation {
		failurePath = "/cancel-deletion?error=google"
	} else if isDeletionRequest {
		failurePath = "/app/settings?error=google_deletion"
	}
	if r.URL.Query().Get("error") != "" {
		http.Redirect(w, r, s.Config.FrontendOrigin+failurePath, http.StatusFound)
		return
	}
	result, err := s.Identity.CompleteGoogleSignIn(s.rpcContext(ctx, pb.IdentityService_CompleteGoogleSignIn_FullMethodName, ""),
		&pb.CompleteGoogleSignInRequest{Code: r.URL.Query().Get("code"), State: r.URL.Query().Get("state")})
	if err != nil {
		http.Redirect(w, r, s.Config.FrontendOrigin+failurePath, http.StatusFound)
		return
	}
	if !isDeletionRequest {
		s.setRefreshCookie(w, result.Session.RefreshToken)
	}
	returnTo := result.ReturnTo
	if result.NewUser {
		returnTo = "/app/settings"
	}
	if returnTo != "/app" && returnTo != "/app/settings" && !(isDeletionRequest && returnTo == "/cancel-deletion") {
		s.problem(w, requestID, 503, "CORE_UNAVAILABLE", "Invalid OAuth return path")
		return
	}
	if isLink {
		returnTo = "/app/settings?linked=google"
	} else if isDeletionCancellation {
		returnTo = "/app/settings?deletion_cancelled=google"
	} else if isDeletionRequest {
		s.clearRefreshCookie(w)
		returnTo = "/cancel-deletion?requested=1"
	}
	http.Redirect(w, r, s.Config.FrontendOrigin+returnTo, http.StatusFound)
}

func (s *Server) requestAccountDeletion(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.Identity.RequestAccountDeletion(s.rpcContext(ctx, pb.IdentityService_RequestAccountDeletion_FullMethodName, user.UserID, user.SessionFamily), &pb.RequestAccountDeletionRequest{Password: body.Password})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.clearRefreshCookie(w)
	s.json(w, http.StatusAccepted, map[string]string{"status": result.Status, "requested_at": result.RequestedAt, "purge_after": result.PurgeAfter})
}

func (s *Server) cancelAccountDeletion(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.Identity.CancelAccountDeletion(s.rpcContext(ctx, pb.IdentityService_CancelAccountDeletion_FullMethodName, ""), &pb.CancelAccountDeletionRequest{Email: body.Email, Password: body.Password})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.setRefreshCookie(w, result.RefreshToken)
	s.sessionJSON(w, result)
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	if !s.allowedOrigin(r) {
		s.problem(w, requestID, 403, "ORIGIN_NOT_ALLOWED", "Origin not allowed")
		return
	}
	cookie, err := r.Cookie("atethat_refresh")
	if err != nil {
		s.problem(w, requestID, 401, "UNAUTHENTICATED", "Session expired")
		return
	}
	result, err := s.Identity.RefreshSession(s.rpcContext(ctx, pb.IdentityService_RefreshSession_FullMethodName, ""),
		&pb.RefreshSessionRequest{RefreshToken: cookie.Value})
	if err != nil {
		s.clearRefreshCookie(w)
		s.rpcProblem(w, requestID, err)
		return
	}
	s.setRefreshCookie(w, result.RefreshToken)
	s.sessionJSON(w, result)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	if !s.allowedOrigin(r) {
		s.problem(w, requestID, 403, "ORIGIN_NOT_ALLOWED", "Origin not allowed")
		return
	}
	value := ""
	if cookie, err := r.Cookie("atethat_refresh"); err == nil {
		value = cookie.Value
	}
	_, err := s.Identity.Logout(s.rpcContext(ctx, pb.IdentityService_Logout_FullMethodName, ""), &pb.LogoutRequest{RefreshToken: value})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	_, err := s.Identity.ResetPassword(s.rpcContext(ctx, pb.IdentityService_ResetPassword_FullMethodName, ""),
		&pb.ResetPasswordRequest{Token: body.Token, NewPassword: body.NewPassword})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) allowedOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	// Non-browser clients may omit Origin; browser cookie requests send it.
	return origin == "" || strings.EqualFold(origin, s.Config.FrontendOrigin)
}

func (s *Server) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{Name: "atethat_refresh", Value: token, Path: "/api/v1/auth", MaxAge: 30 * 24 * 60 * 60,
		HttpOnly: true, Secure: s.Config.CookieSecure, SameSite: http.SameSiteLaxMode})
}

func (s *Server) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "atethat_refresh", Value: "", Path: "/api/v1/auth", MaxAge: -1,
		HttpOnly: true, Secure: s.Config.CookieSecure, SameSite: http.SameSiteLaxMode})
}

func (s *Server) sessionJSON(w http.ResponseWriter, value *pb.SessionResult) {
	s.json(w, 200, map[string]interface{}{
		"access_token": value.AccessToken, "token_type": "Bearer", "expires_in_seconds": value.ExpiresInSeconds,
		"user": map[string]interface{}{"id": value.User.Id, "email": value.User.Email,
			"email_verified": value.User.EmailVerified, "status": value.User.Status},
	})
}
