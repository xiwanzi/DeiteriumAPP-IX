package config

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

var NodeID = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

type Node struct {
	ID                   string `json:"id"`
	TokenSHA256          string `json:"tokenSha256"`
	Chat                 bool   `json:"chat"`
	ItemPrefix           string `json:"itemPrefix"`
	ClaimEnabled         bool   `json:"claimEnabled"`
	InventoryDomain      string `json:"inventoryDomain"`
	CompatibilityProfile string `json:"compatibilityProfile,omitempty"`
	MailCluster          string `json:"mailCluster,omitempty"`
	Economy              bool   `json:"economy,omitempty"`
}

type Config struct {
	SMTPKey           []byte   `json:"-"`
	Listen            string   `json:"listen"`
	PublicOrigin      string   `json:"publicOrigin"`
	Development       bool     `json:"development"`
	TrustedProxyCIDRs []string `json:"trustedProxyCidrs"`
	Nodes             []Node   `json:"nodes"`
}

func Load(path string) (Config, error) {
	var c Config
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 65537))
	d.DisallowUnknownFields()
	if err = d.Decode(&c); err != nil {
		return c, errors.New("invalid service configuration")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return c, errors.New("trailing service configuration")
	}
	if raw := os.Getenv("DEUTERIUM_SMTP_KEY"); raw != "" {
		c.SMTPKey, err = base64.StdEncoding.DecodeString(raw)
		if err != nil || len(c.SMTPKey) != 32 {
			return c, errors.New("DEUTERIUM_SMTP_KEY must be base64 encoding of 32 random bytes")
		}
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	u, err := url.Parse(c.PublicOrigin)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || u.Opaque != "" {
		return errors.New("publicOrigin must be an exact origin without path")
	}
	if u.Scheme != "https" {
		if !c.Development || u.Scheme != "http" || !loopback(u.Hostname()) {
			return errors.New("HTTPS publicOrigin required")
		}
	}
	host, _, err := net.SplitHostPort(c.Listen)
	if err != nil {
		return errors.New("listen must be host:port")
	}
	if c.Development && !loopback(host) {
		return errors.New("development listener must use loopback")
	}
	for _, cidr := range c.TrustedProxyCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return errors.New("invalid trusted proxy network")
		}
		ones, _ := network.Mask.Size()
		if ones == 0 {
			return errors.New("wildcard trusted proxy forbidden")
		}
	}
	seen := map[string]bool{}
	hashes := map[string]bool{}
	if len(c.Nodes) > 32 {
		return errors.New("too many core nodes")
	}
	economyNodes := 0
	for _, n := range c.Nodes {
		hash, err := hex.DecodeString(n.TokenSHA256)
		if !NodeID.MatchString(n.ID) || seen[n.ID] || err != nil || len(hash) != 32 || strings.ToLower(n.TokenSHA256) != n.TokenSHA256 || hashes[n.TokenSHA256] {
			return errors.New("invalid or duplicate core node credential")
		}
		if n.ItemPrefix != "" && !regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}:$`).MatchString(n.ItemPrefix) {
			return errors.New("invalid item namespace")
		}
		if !NodeID.MatchString(n.InventoryDomain) {
			return errors.New("inventoryDomain is required for each node")
		}
		if n.CompatibilityProfile != "" && !NodeID.MatchString(n.CompatibilityProfile) {
			return errors.New("invalid compatibility profile")
		}
		if n.MailCluster != "" && !NodeID.MatchString(n.MailCluster) {
			return errors.New("invalid mailbox cluster")
		}
		if n.Economy {
			economyNodes++
		}
		seen[n.ID] = true
		hashes[n.TokenSHA256] = true
	}
	if economyNodes > 1 {
		return errors.New("only one Core economy authority is allowed")
	}
	return nil
}

func loopback(host string) bool { ip := net.ParseIP(host); return ip != nil && ip.IsLoopback() }

func (c Config) AuthenticateNode(id, token string) (Node, bool) {
	if len(token) < 32 || len(token) > 256 {
		return Node{}, false
	}
	hash := store.Digest([]byte(token))
	for _, n := range c.Nodes {
		if n.ID == id && subtle.ConstantTimeCompare([]byte(n.TokenSHA256), []byte(hash)) == 1 {
			return n, true
		}
	}
	return Node{}, false
}

func (c Config) ChatNodes() []string {
	ids := []string{}
	for _, n := range c.Nodes {
		if n.Chat {
			ids = append(ids, n.ID)
		}
	}
	return ids
}

func (c Config) HasNode(id string) bool {
	for _, n := range c.Nodes {
		if n.ID == id {
			return true
		}
	}
	return false
}

func (c Config) TrustedProxy(ip net.IP) bool {
	for _, cidr := range c.TrustedProxyCIDRs {
		_, n, err := net.ParseCIDR(cidr)
		if err == nil && n.Contains(ip) {
			return true
		}
	}
	return false
}
