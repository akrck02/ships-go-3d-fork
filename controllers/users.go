package controllers

import (
	"errors"
	"log"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	dataaccess "ships3d/dataAccess"
	"ships3d/i18n"
	"ships3d/models"
)

// Usernames are shown to every player in game, so they are restricted to
// letters, digits and a little punctuation. (Accounts created by the 2D game
// predate this rule and are left as they are.)
var usernamePattern = regexp.MustCompile(`^[\p{L}\p{N}_.\-]{3,20}$`)

const (
	minPasswordLength = 8
	// bcrypt ignores everything past 72 bytes, so a longer password would
	// silently match any other with the same first 72 bytes.
	maxPasswordLength = 72
)

func (s *Server) registerUserRoutes(router *gin.Engine) {
	router.POST("/register", s.register)
	router.POST("/login", s.login)
	router.POST("/logout", s.logout)
	router.GET("/userInfo", s.userInfo)
	router.POST("/changePassword", s.requireUser, s.changePassword)
}

type registerBody struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Cpassword string `json:"cpassword"`
}

func (s *Server) register(c *gin.Context) {
	var body registerBody
	if err := c.ShouldBindJSON(&body); err != nil {
		errorResponse(c, http.StatusBadRequest, i18n.New("invalidBody"))
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	body.Email = strings.TrimSpace(body.Email)

	var problems []i18n.Message
	if !usernamePattern.MatchString(body.Username) {
		problems = append(problems, i18n.New("usernameRule"))
	}
	if address, err := mail.ParseAddress(body.Email); err != nil || address.Address != body.Email {
		problems = append(problems, i18n.New("emailInvalid"))
	}
	if problem, bad := passwordProblem(body.Password); bad {
		problems = append(problems, problem)
	}
	if body.Password != body.Cpassword {
		problems = append(problems, i18n.New("passwordsDontMatch"))
	}
	if len(problems) > 0 {
		errorResponse(c, http.StatusBadRequest, problems...)
		return
	}

	if _, err := s.store.CreateUser(body.Username, body.Email, body.Password); err != nil {
		if errors.Is(err, dataaccess.ErrUserExists) {
			errorResponse(c, http.StatusConflict, i18n.New("userExists"))
			return
		}
		log.Println("register:", err)
		errorResponse(c, http.StatusInternalServerError, i18n.New("createUserFailed"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func passwordProblem(password string) (i18n.Message, bool) {
	if len(password) < minPasswordLength {
		return i18n.New("passwordTooShort", minPasswordLength), true
	}
	if len(password) > maxPasswordLength {
		return i18n.New("passwordTooLong", maxPasswordLength), true
	}
	return i18n.Message{}, false
}

type loginBody struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe"`
}

func (s *Server) login(c *gin.Context) {
	var body loginBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" || body.Password == "" {
		errorResponse(c, http.StatusBadRequest, i18n.New("credentialsRequired"))
		return
	}
	user, err := s.store.Authenticate(strings.TrimSpace(body.Email), body.Password)
	if errors.Is(err, dataaccess.ErrInvalidCredentials) {
		errorResponse(c, http.StatusUnauthorized, i18n.New("invalidCredentials"))
		return
	}
	if err != nil {
		log.Println("login:", err)
		errorResponse(c, http.StatusInternalServerError, i18n.New("loginFailed"))
		return
	}

	now := time.Now()
	session := &models.Session{
		Admin:            user.Admin,
		UserId:           user.IdAsString(),
		SessionTimeStamp: now.UnixMilli(),
		Persistent:       body.RememberMe,
		Token:            newToken(),
		ExpirationTime:   now.Add(sessionDuration).UnixMilli(),
	}
	if body.RememberMe {
		session.ExpirationTime = now.Add(persistentCookieAge).UnixMilli()
	}
	if err := s.store.InsertSession(session); err != nil {
		log.Println("login: session:", err)
		errorResponse(c, http.StatusInternalServerError, i18n.New("loginFailed"))
		return
	}
	s.setTokenCookie(c, session)
	c.JSON(http.StatusOK, gin.H{"success": true, "user": user})
}

func (s *Server) logout(c *gin.Context) {
	_, session := s.currentUser(c)
	if session != nil {
		session.LoggedOut = true
		session.Persistent = false
		if err := s.store.UpdateSession(session); err != nil {
			log.Println("logout:", err)
		}
	}
	clearTokenCookie(c)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// userInfo answers 200 either way: "not logged in" is a normal state for a
// page asking who is there, not an error.
func (s *Server) userInfo(c *gin.Context) {
	user, _ := s.currentUser(c)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

type changePasswordBody struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// changePassword also ends the user's other sessions, so a password changed
// because it leaked actually locks the other party out.
func (s *Server) changePassword(c *gin.Context) {
	var body changePasswordBody
	if err := c.ShouldBindJSON(&body); err != nil {
		errorResponse(c, http.StatusBadRequest, i18n.New("invalidBody"))
		return
	}
	if problem, bad := passwordProblem(body.NewPassword); bad {
		errorResponse(c, http.StatusBadRequest, problem)
		return
	}
	user, session := contextUser(c), contextSession(c)
	err := s.store.ChangePassword(user.Id, body.CurrentPassword, body.NewPassword)
	if errors.Is(err, dataaccess.ErrInvalidCredentials) {
		errorResponse(c, http.StatusBadRequest, i18n.New("currentPasswordWrong"))
		return
	}
	if err != nil {
		log.Println("changePassword:", err)
		errorResponse(c, http.StatusInternalServerError, i18n.New("changePasswordFailed"))
		return
	}
	if err := s.store.LogoutUserSessions(user.IdAsString(), session.Id); err != nil {
		log.Println("changePassword: sessions:", err)
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
