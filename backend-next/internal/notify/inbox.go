package notify

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type InboxSettings struct {
	Enabled            bool       `json:"enabled"`
	Host               string     `json:"host"`
	Port               int        `json:"port"`
	Username           string     `json:"username"`
	Address            string     `json:"address"`
	Folder             string     `json:"folder"`
	ImportExisting     bool       `json:"importExisting"`
	PasswordConfigured bool       `json:"passwordConfigured"`
	Version            int64      `json:"version"`
	LastPollAt         *time.Time `json:"lastPollAt"`
	LastError          string     `json:"lastError"`
	AcceptAfter        *time.Time `json:"acceptAfter,omitempty"`
	Initialized        bool       `json:"initialized"`
}

func (s InboxSettings) Validate() error {
	if strings.TrimSpace(s.Folder) != s.Folder || s.Folder == "" || len(s.Folder) > 100 || strings.ContainsAny(s.Folder, "\r\n\x00") || s.Username == "" {
		return errors.New("请填写完整的玩家支持收件箱设置。")
	}
	return (Settings{Host: s.Host, Port: s.Port, Security: "TLS", Username: s.Username, From: s.Address, Recipients: []string{s.Address}}).Validate()
}

// Read-only IMAP with BODY.PEEK: operating the workbench never deletes mail,
// changes the user's Seen flags, or treats sequence numbers as durable IDs.
func PollInbox(ctx context.Context, s InboxSettings, password string, position func(uint32, uint32) (uint32, error), receive func(uint32, []byte, time.Time) error) error {
	return pollInboxWithRoots(ctx, s, password, position, receive, nil)
}
func pollInboxWithRoots(ctx context.Context, s InboxSettings, password string, position func(uint32, uint32) (uint32, error), receive func(uint32, []byte, time.Time) error, roots *x509.CertPool) error {
	if s.Validate() != nil {
		return errors.New("玩家支持收件箱设置不完整。")
	}
	dialer := tls.Dialer{NetDialer: &net.Dialer{Timeout: 8 * time.Second}, Config: &tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12, RootCAs: roots}}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(s.Host, strconv.Itoa(s.Port)))
	if err != nil {
		return errors.New("IMAP TLS 连接失败，请检查主机、端口和证书。")
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	c, err := client.New(conn)
	if err != nil {
		return errors.New("IMAP 服务响应无效。")
	}
	c.ErrorLog = log.New(io.Discard, "", 0)
	c.Timeout = 15 * time.Second
	defer c.Terminate()
	if err = c.Login(s.Username, password); err != nil {
		return errors.New("IMAP 登录失败，请检查账号、授权码及 IMAP 是否开启。")
	}
	box, err := c.Select(s.Folder, true)
	if err != nil {
		return errors.New("无法打开玩家支持邮件文件夹。")
	}
	after, err := position(box.UidValidity, box.UidNext)
	if err != nil {
		return err
	}
	if box.Messages == 0 || box.UidNext <= after+1 {
		return nil
	}
	criteria := imap.NewSearchCriteria()
	if s.AcceptAfter != nil {
		criteria.Since = s.AcceptAfter.Add(-24 * time.Hour)
	}
	criteria.Uid = new(imap.SeqSet)
	criteria.Uid.AddRange(after+1, 0)
	uids, err := c.UidSearch(criteria)
	if err != nil {
		return errors.New("无法读取玩家支持邮件目录。")
	}
	sort.Slice(uids, func(i, j int) bool { return uids[i] < uids[j] })
	count := 0
	for _, uid := range uids {
		if uid <= after {
			continue
		}
		if count >= 25 {
			break
		}
		count++
		set := new(imap.SeqSet)
		set.AddNum(uid)
		section := &imap.BodySectionName{Peek: true, Partial: []int{0, MaxInboundBytes + 1}}
		messages := make(chan *imap.Message, 1)
		done := make(chan error, 1)
		go func() {
			done <- c.UidFetch(set, []imap.FetchItem{imap.FetchUid, imap.FetchInternalDate, imap.FetchRFC822Size, section.FetchItem()}, messages)
		}()
		var data []byte
		var received time.Time
		var readErr error
		for message := range messages {
			if message == nil || message.Uid != uid {
				continue
			}
			received = message.InternalDate
			body := message.GetBody(section)
			if body != nil {
				data, readErr = io.ReadAll(io.LimitReader(body, MaxInboundBytes+1))
			}
		}
		if fetchErr := <-done; fetchErr != nil || readErr != nil {
			return errors.New("邮件读取中断，将在下次同步继续。")
		}
		if data == nil {
			continue
		} // Message was moved or deleted by another mail client.
		if received.IsZero() {
			return errors.New("邮件服务器未提供有效的收件时间。")
		}
		if err = receive(uid, data, received); err != nil {
			return err
		}
	}
	return nil
}
