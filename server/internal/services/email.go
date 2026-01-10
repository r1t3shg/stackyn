package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type EmailService struct {
	logger    *zap.Logger
	apiKey    string
	fromEmail string
	baseURL   string
	client    *http.Client
}

type ResendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

type ResendEmailResponse struct {
	ID string `json:"id"`
}

type ResendErrorResponse struct {
	Message string `json:"message"`
}

// NewEmailService creates a new email service using Resend API
func NewEmailService(logger *zap.Logger, apiKey, fromEmail string) *EmailService {
	return &EmailService{
		logger:    logger,
		apiKey:    apiKey,
		fromEmail: fromEmail,
		baseURL:   "https://api.resend.com",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendOTPEmail sends an OTP email to the user
func (s *EmailService) SendOTPEmail(email, otp string) error {
	subject := "Your Stackyn Verification Code"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; font-size: 28px;">Stackyn</h1>
			</div>
			<div style="background: #ffffff; padding: 40px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
				<h2 style="color: #333; margin-top: 0;">Verify Your Email</h2>
				<p style="color: #666; font-size: 16px;">Your verification code is:</p>
				<div style="background: #f5f5f5; border: 2px dashed #667eea; border-radius: 8px; padding: 20px; text-align: center; margin: 30px 0;">
					<code style="font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #667eea; font-family: 'Courier New', monospace;">%s</code>
				</div>
				<p style="color: #666; font-size: 14px;">This code will expire in 10 minutes.</p>
				<p style="color: #999; font-size: 12px; margin-top: 30px; border-top: 1px solid #e0e0e0; padding-top: 20px;">If you didn't request this code, you can safely ignore this email.</p>
			</div>
		</body>
		</html>
	`, otp)

	return s.sendEmail(email, subject, htmlBody)
}

// SendPasswordResetOTPEmail sends a password reset OTP email to the user
func (s *EmailService) SendPasswordResetOTPEmail(email, otp string) error {
	subject := "Reset Your Stackyn Password"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; font-size: 28px;">Stackyn</h1>
			</div>
			<div style="background: #ffffff; padding: 40px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
				<h2 style="color: #333; margin-top: 0;">Reset Your Password</h2>
				<p style="color: #666; font-size: 16px;">You requested to reset your password. Use the following code to verify your identity:</p>
				<div style="background: #f5f5f5; border: 2px dashed #667eea; border-radius: 8px; padding: 20px; text-align: center; margin: 30px 0;">
					<code style="font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #667eea; font-family: 'Courier New', monospace;">%s</code>
				</div>
				<p style="color: #666; font-size: 14px;">This code will expire in 10 minutes.</p>
				<p style="color: #999; font-size: 12px; margin-top: 30px; border-top: 1px solid #e0e0e0; padding-top: 20px;">If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.</p>
			</div>
		</body>
		</html>
	`, otp)

	return s.sendEmail(email, subject, htmlBody)
}

// sendEmail sends an email using Resend API
func (s *EmailService) sendEmail(to, subject, htmlBody string) error {
	if s.apiKey == "" {
		s.logger.Warn("Resend API key not configured, skipping email send", zap.String("to", to))
		return fmt.Errorf("email service not configured")
	}

	reqBody := ResendEmailRequest{
		From:    s.fromEmail,
		To:      []string{to},
		Subject: subject,
		HTML:    htmlBody,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal email request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/emails", s.baseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp ResendErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			return fmt.Errorf("resend API error: %s", errorResp.Message)
		}
		return fmt.Errorf("resend API returned status %d", resp.StatusCode)
	}

	var emailResp ResendEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&emailResp); err != nil {
		s.logger.Warn("Failed to decode email response", zap.Error(err))
	}

	s.logger.Info("Email sent successfully", zap.String("to", to), zap.String("email_id", emailResp.ID))
	return nil
}

