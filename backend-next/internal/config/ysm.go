package config

import (
	"errors"
	"net/url"
	"strconv"
)

// YSM remains a passwordless loopback service; a remote host must use a tunnel.
func (c Config) ValidateYSM() error {
	if c.YSMAdminURL == "" {
		if c.YSMAdminOrigin != "" {
			return errors.New("ysmAdminOrigin requires ysmAdminUrl")
		}
		return nil
	}
	for _, raw := range []string{c.YSMAdminURL, c.YSMOrigin()} {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "http" || (u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") || u.User != nil || raw != "http://"+u.Host {
			return errors.New("YSM admin addresses must be exact literal HTTP loopback origins")
		}
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return errors.New("YSM admin addresses require an explicit valid port")
		}
	}
	return nil
}

func (c Config) YSMOrigin() string {
	if c.YSMAdminOrigin != "" {
		return c.YSMAdminOrigin
	}
	return c.YSMAdminURL
}
