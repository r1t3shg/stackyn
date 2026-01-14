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

// SendTrialStartedEmail sends a welcome email when a trial starts
func (s *EmailService) SendTrialStartedEmail(email string, trialEndsAt time.Time) error {
	subject := "Your Stackyn 7-day trial has started"
	trialEndDate := trialEndsAt.Format("January 2, 2006")
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; font-size: 28px;">Welcome to Stackyn! 🎉</h1>
			</div>
			<div style="background: #ffffff; padding: 40px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
				<h2 style="color: #333; margin-top: 0;">Your 7-Day Free Trial Has Started</h2>
				<p style="color: #666; font-size: 16px;">Welcome to Stackyn! You now have full access to Pro features during your 7-day free trial.</p>
				
				<div style="background: #f5f5f5; border-left: 4px solid #667eea; padding: 20px; margin: 30px 0;">
					<h3 style="color: #333; margin-top: 0;">Your Trial Details</h3>
					<p style="color: #666; margin: 10px 0;"><strong>Trial End Date:</strong> %s</p>
					<p style="color: #666; margin: 10px 0;"><strong>Resource Limits:</strong> 2GB RAM / 20GB Disk</p>
					<p style="color: #666; margin: 10px 0;"><strong>Apps:</strong> Up to 3 apps</p>
				</div>

				<p style="color: #666; font-size: 16px;">No credit card required. Your trial expires automatically after 7 days.</p>
				
				<div style="text-align: center; margin: 30px 0;">
					<a href="https://stackyn.com/apps/new" style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 15px 30px; text-decoration: none; border-radius: 5px; font-weight: bold; display: inline-block;">Deploy Your First App</a>
				</div>

				<p style="color: #999; font-size: 12px; margin-top: 30px; border-top: 1px solid #e0e0e0; padding-top: 20px;">If you have any questions, feel free to reach out to our support team.</p>
			</div>
		</body>
		</html>
	`, trialEndDate)

	return s.sendEmail(email, subject, htmlBody)
}

// SendTrialEndingEmail sends a reminder email when a trial is about to end
func (s *EmailService) SendTrialEndingEmail(email string, trialEndsAt time.Time) error {
	subject := "Your Stackyn trial ends tomorrow"
	trialEndDate := trialEndsAt.Format("January 2, 2006")
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; font-size: 28px;">Trial Ending Soon</h1>
			</div>
			<div style="background: #ffffff; padding: 40px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
				<h2 style="color: #333; margin-top: 0;">Your Stackyn trial ends tomorrow</h2>
				<p style="color: #666; font-size: 16px;">Your 7-day free trial ends on <strong>%s</strong>. Continue enjoying Stackyn by upgrading to a paid plan.</p>
				
				<div style="background: #f5f5f5; padding: 20px; margin: 30px 0; border-radius: 8px;">
					<h3 style="color: #333; margin-top: 0;">Pricing Plans</h3>
					
					<div style="margin: 20px 0; padding: 15px; background: white; border-left: 4px solid #667eea; border-radius: 4px;">
						<h4 style="margin: 0; color: #333;">Starter — $19/month</h4>
						<ul style="color: #666; margin: 10px 0; padding-left: 20px;">
							<li>1 app, 1 VPS</li>
							<li>512 MB RAM</li>
							<li>5 GB Disk</li>
						</ul>
					</div>
					
					<div style="margin: 20px 0; padding: 15px; background: white; border-left: 4px solid #764ba2; border-radius: 4px;">
						<h4 style="margin: 0; color: #333;">Pro — $49/month</h4>
						<ul style="color: #666; margin: 10px 0; padding-left: 20px;">
							<li>Up to 3 apps, 1 VPS</li>
							<li>2 GB RAM (shared)</li>
							<li>20 GB Disk</li>
						</ul>
					</div>
				</div>

				<div style="text-align: center; margin: 30px 0;">
					<a href="https://stackyn.com/upgrade" style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 15px 30px; text-decoration: none; border-radius: 5px; font-weight: bold; display: inline-block;">Upgrade Now</a>
				</div>

				<p style="color: #999; font-size: 12px; margin-top: 30px; border-top: 1px solid #e0e0e0; padding-top: 20px;">After your trial expires, existing apps will keep running, but new deployments will be blocked until you upgrade.</p>
			</div>
		</body>
		</html>
	`, trialEndDate)

	return s.sendEmail(email, subject, htmlBody)
}

// SendTrialExpiredEmail sends an email when a trial expires
func (s *EmailService) SendTrialExpiredEmail(email string) error {
	subject := "Your Stackyn trial has ended"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; font-size: 28px;">Trial Expired</h1>
			</div>
			<div style="background: #ffffff; padding: 40px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
				<h2 style="color: #333; margin-top: 0;">Your Stackyn trial has ended</h2>
				<p style="color: #666; font-size: 16px;">Your 7-day free trial has expired. Here's what happens next:</p>
				
				<div style="background: #f5f5f5; border-left: 4px solid #667eea; padding: 20px; margin: 30px 0;">
					<ul style="color: #666; margin: 10px 0; padding-left: 20px;">
						<li><strong>Existing apps</strong> will continue running</li>
						<li><strong>New deploys</strong> will be blocked until you upgrade</li>
						<li><strong>No data loss</strong> - all your apps and data are safe</li>
					</ul>
				</div>

				<p style="color: #666; font-size: 16px;">Upgrade to a paid plan to continue deploying new apps and unlock all features.</p>
				
				<div style="text-align: center; margin: 30px 0;">
					<a href="https://stackyn.com/upgrade" style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 15px 30px; text-decoration: none; border-radius: 5px; font-weight: bold; display: inline-block;">Upgrade Now</a>
				</div>

				<p style="color: #999; font-size: 12px; margin-top: 30px; border-top: 1px solid #e0e0e0; padding-top: 20px;">If you have any questions, feel free to reach out to our support team.</p>
			</div>
		</body>
		</html>
	`)

	return s.sendEmail(email, subject, htmlBody)
}

// SendSubscriptionActivatedEmail sends a welcome email when a subscription is activated
func (s *EmailService) SendSubscriptionActivatedEmail(email, planName string, ramLimitMB, diskLimitGB int) error {
	subject := "Welcome to Stackyn 🎉"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; font-size: 28px;">Welcome to Stackyn! 🎉</h1>
			</div>
			<div style="background: #ffffff; padding: 40px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
				<h2 style="color: #333; margin-top: 0;">Your subscription is active</h2>
				<p style="color: #666; font-size: 16px;">Thank you for upgrading to <strong>%s</strong>!</p>
				
				<div style="background: #f5f5f5; border-left: 4px solid #667eea; padding: 20px; margin: 30px 0;">
					<h3 style="color: #333; margin-top: 0;">Your Plan Details</h3>
					<p style="color: #666; margin: 10px 0;"><strong>Plan:</strong> %s</p>
					<p style="color: #666; margin: 10px 0;"><strong>RAM Limit:</strong> %d MB</p>
					<p style="color: #666; margin: 10px 0;"><strong>Disk Limit:</strong> %d GB</p>
				</div>

				<p style="color: #666; font-size: 16px;">You now have full access to all features. Start deploying your apps!</p>
				
				<div style="text-align: center; margin: 30px 0;">
					<a href="https://stackyn.com/apps" style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 15px 30px; text-decoration: none; border-radius: 5px; font-weight: bold; display: inline-block;">View Your Apps</a>
				</div>

				<p style="color: #999; font-size: 12px; margin-top: 30px; border-top: 1px solid #e0e0e0; padding-top: 20px;">If you need any help, feel free to reach out to our support team. We're here to help!</p>
			</div>
		</body>
		</html>
	`, planName, planName, ramLimitMB, diskLimitGB)

	return s.sendEmail(email, subject, htmlBody)
}

// SendPaymentFailedEmail sends an email when payment fails
func (s *EmailService) SendPaymentFailedEmail(email string) error {
	subject := "Payment Failed - Action Required"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; font-size: 28px;">Payment Failed</h1>
			</div>
			<div style="background: #ffffff; padding: 40px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
				<h2 style="color: #333; margin-top: 0;">We couldn't process your payment</h2>
				<p style="color: #666; font-size: 16px;">Your recent payment attempt failed. Here's what happens next:</p>
				
				<div style="background: #fff3cd; border-left: 4px solid #ffc107; padding: 20px; margin: 30px 0;">
					<h3 style="color: #333; margin-top: 0;">What happens now:</h3>
					<ul style="color: #666; margin: 10px 0; padding-left: 20px;">
						<li><strong>Your apps will be stopped</strong> until payment is resolved</li>
						<li><strong>No data loss</strong> - all your apps and data are safe</li>
						<li><strong>Update your payment method</strong> to restore service</li>
					</ul>
				</div>

				<p style="color: #666; font-size: 16px;">Please update your payment method to continue using Stackyn.</p>
				
				<div style="text-align: center; margin: 30px 0;">
					<a href="https://stackyn.com/billing" style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 15px 30px; text-decoration: none; border-radius: 5px; font-weight: bold; display: inline-block;">Update Payment Method</a>
				</div>

				<p style="color: #999; font-size: 12px; margin-top: 30px; border-top: 1px solid #e0e0e0; padding-top: 20px;">If you have any questions, feel free to reach out to our support team.</p>
			</div>
		</body>
		</html>
	`)

	return s.sendEmail(email, subject, htmlBody)
}

// SendSubscriptionExpiredEmail sends an email when subscription expires
func (s *EmailService) SendSubscriptionExpiredEmail(email string) error {
	subject := "Your Stackyn subscription has expired"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; font-size: 28px;">Subscription Expired</h1>
			</div>
			<div style="background: #ffffff; padding: 40px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
				<h2 style="color: #333; margin-top: 0;">Your subscription has expired</h2>
				<p style="color: #666; font-size: 16px;">Your Stackyn subscription has expired. Here's what happens next:</p>
				
				<div style="background: #f5f5f5; border-left: 4px solid #667eea; padding: 20px; margin: 30px 0;">
					<h3 style="color: #333; margin-top: 0;">What happens now:</h3>
					<ul style="color: #666; margin: 10px 0; padding-left: 20px;">
						<li><strong>All apps are stopped</strong> until you resubscribe</li>
						<li><strong>No data loss</strong> - all your apps and data are safe</li>
						<li><strong>Resubscribe anytime</strong> to restore service</li>
					</ul>
				</div>

				<p style="color: #666; font-size: 16px;">Resubscribe to continue using Stackyn and restore your apps.</p>
				
				<div style="text-align: center; margin: 30px 0;">
					<a href="https://stackyn.com/pricing" style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 15px 30px; text-decoration: none; border-radius: 5px; font-weight: bold; display: inline-block;">Resubscribe Now</a>
				</div>

				<p style="color: #999; font-size: 12px; margin-top: 30px; border-top: 1px solid #e0e0e0; padding-top: 20px;">If you have any questions, feel free to reach out to our support team.</p>
			</div>
		</body>
		</html>
	`)

	return s.sendEmail(email, subject, htmlBody)
}

