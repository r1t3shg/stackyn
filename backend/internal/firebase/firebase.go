// Package firebase provides Firebase Admin SDK integration for authentication.
package firebase

import (
	"context"
	"fmt"
	"log"
	"os"

	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type Service struct {
	app  *firebase.App
	auth *auth.Client
}

func NewService() (*Service, error) {
	// Get Firebase credentials from environment variable
	credentialsPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")
	if credentialsPath == "" {
		// Try to use GOOGLE_APPLICATION_CREDENTIALS (standard Firebase env var)
		credentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}

	var opts []option.ClientOption
	if credentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsPath))
		log.Printf("[FIREBASE] Using credentials from: %s", credentialsPath)
	} else {
		log.Println("[FIREBASE] WARNING - No credentials file specified. Using default credentials.")
	}

	// Initialize Firebase app
	app, err := firebase.NewApp(context.Background(), nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app: %w", err)
	}

	// Get Auth client
	authClient, err := app.Auth(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get Auth client: %w", err)
	}

	log.Println("[FIREBASE] Firebase Auth service initialized successfully")
	return &Service{
		app:  app,
		auth: authClient,
	}, nil
}

// VerifyIDToken verifies a Firebase ID token and returns the user's UID and email
func (s *Service) VerifyIDToken(ctx context.Context, idToken string) (uid, email string, err error) {
	token, err := s.auth.VerifyIDToken(ctx, idToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to verify ID token: %w", err)
	}

	uid = token.UID
	email, _ = token.Claims["email"].(string)

	return uid, email, nil
}

// GetUserByEmail retrieves a Firebase user by email
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*auth.UserRecord, error) {
	user, err := s.auth.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

// CreateUser creates a new Firebase user
func (s *Service) CreateUser(ctx context.Context, email, password string) (*auth.UserRecord, error) {
	params := (&auth.UserToCreate{}).
		Email(email).
		Password(password).
		EmailVerified(false) // User needs to verify email

	user, err := s.auth.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	log.Printf("[FIREBASE] Created user: %s (UID: %s)", email, user.UID)
	return user, nil
}

// SendEmailVerification sends an email verification link to the user
func (s *Service) SendEmailVerification(ctx context.Context, uid string) error {
	emailVerificationLink, err := s.auth.EmailVerificationLink(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to generate email verification link: %w", err)
	}

	// Note: Firebase Admin SDK doesn't send emails directly
	// You need to use Firebase Auth REST API or configure email templates in Firebase Console
	// For now, we'll log the link (in production, Firebase will send it automatically)
	log.Printf("[FIREBASE] Email verification link for UID %s: %s", uid, emailVerificationLink)
	
	// In production, Firebase automatically sends the email when you call
	// the sendEmailVerification method on the client side
	return nil
}

// UpdateUser updates user information
func (s *Service) UpdateUser(ctx context.Context, uid string, displayName *string, emailVerified bool) error {
	update := (&auth.UserToUpdate{}).
		EmailVerified(emailVerified)
	
	if displayName != nil {
		update = update.DisplayName(*displayName)
	}

	_, err := s.auth.UpdateUser(ctx, uid, update)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

