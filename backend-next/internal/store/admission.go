package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
)

const AdmissionCovenantVersion = "2026-09-12-v1"

var admissionGameName = regexp.MustCompile(`^[A-Za-z0-9_]{3,16}$`)
var admissionQQ = regexp.MustCompile(`^[1-9][0-9]{4,10}$`)
var admissionUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type AdmissionProfile struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type AdmissionSubmission struct {
	ReceiptToken     string   `json:"receiptToken"`
	GameID           string   `json:"gameId"`
	QQ               string   `json:"qq"`
	Interests        []string `json:"interests"`
	Message          string   `json:"message"`
	CovenantVersion  string   `json:"covenantVersion"`
	CovenantAccepted bool     `json:"covenantAccepted"`
	Website          string   `json:"website"`
}

func ValidAdmissionName(name string) bool { return admissionGameName.MatchString(name) }
func ValidAdmissionUUID(uuid string) bool { return admissionUUID.MatchString(uuid) }
func ValidAdmissionReceipt(token string) bool {
	b, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil && len(b) == 32 && len(token) == 43 && base64.RawURLEncoding.EncodeToString(b) == token
}
func (in *AdmissionSubmission) Normalize() {
	in.GameID = strings.TrimSpace(in.GameID)
	in.QQ = strings.TrimSpace(in.QQ)
	in.Message = strings.TrimSpace(in.Message)
	if in.Interests == nil {
		in.Interests = []string{}
	}
}
func (in AdmissionSubmission) Validate() error {
	if !ValidAdmissionReceipt(in.ReceiptToken) || !ValidAdmissionName(in.GameID) || !admissionQQ.MatchString(in.QQ) || !ValidSocialText(in.Message, 0, 200) || in.Website != "" || len(in.Interests) > 4 {
		return catalogError(400, "ADMISSION_INVALID", "请检查正版玩家名、QQ 号码和补充记录。")
	}
	if !in.CovenantAccepted || in.CovenantVersion != AdmissionCovenantVersion {
		return catalogError(400, "COVENANT_REQUIRED", "请阅读并同意当前版本的《文明游戏公约》。")
	}
	seen := map[string]bool{}
	for _, value := range in.Interests {
		if !catalogEnum(value, "建筑创造", "工业自动化", "探索冒险", "日常同行") || seen[value] {
			return ErrSocialInvalid
		}
		seen[value] = true
	}
	return nil
}
func admissionFingerprint(in AdmissionSubmission) string {
	in.GameID = strings.ToLower(in.GameID)
	b, _ := json.Marshal(in)
	return Digest(b)
}

type AdmissionApplication struct {
	ID              string     `json:"applicationId"`
	UUID            string     `json:"uuid"`
	GameID          string     `json:"gameId"`
	QQ              string     `json:"qq"`
	Interests       []string   `json:"interests"`
	Message         string     `json:"message"`
	CovenantVersion string     `json:"covenantVersion"`
	Status          string     `json:"status"`
	Version         int64      `json:"version"`
	CreatedAt       time.Time  `json:"createdAt"`
	ReviewedAt      *time.Time `json:"reviewedAt"`
	Reviewer        string     `json:"reviewer"`
	Reason          string     `json:"reason"`
}

const admissionApplicationColumns = `a.application_id,a.server_uuid,a.game_id,a.qq,a.interests_json,a.message,a.covenant_version,a.status,a.version,a.created_at,a.reviewed_at,COALESCE(i.game_id,''),a.reason`

type admissionScanner interface{ Scan(...any) error }

func scanAdmissionApplication(row admissionScanner) (AdmissionApplication, error) {
	var a AdmissionApplication
	var interests string
	err := row.Scan(&a.ID, &a.UUID, &a.GameID, &a.QQ, &interests, &a.Message, &a.CovenantVersion, &a.Status, &a.Version, &a.CreatedAt, &a.ReviewedAt, &a.Reviewer, &a.Reason)
	if err == nil {
		err = json.Unmarshal([]byte(interests), &a.Interests)
	}
	return a, err
}
func lockAdmission(ctx context.Context, tx *sql.Tx) error {
	var id int
	return tx.QueryRowContext(ctx, "SELECT id FROM admission_guard WHERE id=1 FOR UPDATE").Scan(&id)
}

func (s *Store) ReserveAdmission(ctx context.Context, ip, qq string, statusOnly bool) error {
	limits := map[string]int{Digest([]byte("admission:ip:" + ip)): 12}
	if statusOnly {
		limits = map[string]int{Digest([]byte("admission:status:" + ip)): 120}
	} else if qq != "" {
		limits[Digest([]byte("admission:qq:"+qq))] = 4
	}
	return s.reserveBudgets(ctx, limits)
}

func (s *Store) AdmissionReceipt(ctx context.Context, token string) (AdmissionApplication, error) {
	if !ValidAdmissionReceipt(token) {
		return AdmissionApplication{}, ErrSocialNotFound
	}
	a, err := scanAdmissionApplication(s.DB.QueryRowContext(ctx, "SELECT "+admissionApplicationColumns+" FROM admission_applications a LEFT JOIN identities i ON i.id=a.reviewer_id WHERE a.receipt_hash=?", Digest([]byte(token))))
	return a, socialMissing(err)
}

// A receipt is also the request key: replaying a lost response never creates a new application.
func (s *Store) ExistingAdmission(ctx context.Context, in AdmissionSubmission) (AdmissionApplication, bool, error) {
	var fingerprint string
	err := s.DB.QueryRowContext(ctx, "SELECT fingerprint FROM admission_applications WHERE receipt_hash=?", Digest([]byte(in.ReceiptToken))).Scan(&fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return AdmissionApplication{}, false, nil
	}
	if err != nil {
		return AdmissionApplication{}, false, err
	}
	if fingerprint != admissionFingerprint(in) {
		return AdmissionApplication{}, false, ErrConflict
	}
	a, err := s.AdmissionReceipt(ctx, in.ReceiptToken)
	return a, true, err
}

func (s *Store) SubmitAdmission(ctx context.Context, in AdmissionSubmission, p AdmissionProfile) (AdmissionApplication, error) {
	if err := in.Validate(); err != nil {
		return AdmissionApplication{}, err
	}
	if !ValidAdmissionUUID(p.UUID) || !ValidAdmissionName(p.Name) {
		return AdmissionApplication{}, ErrSocialInvalid
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return AdmissionApplication{}, err
	}
	defer tx.Rollback()
	if err = lockAdmission(ctx, tx); err != nil {
		return AdmissionApplication{}, err
	}
	var existing string
	err = tx.QueryRowContext(ctx, "SELECT fingerprint FROM admission_applications WHERE receipt_hash=?", Digest([]byte(in.ReceiptToken))).Scan(&existing)
	if err == nil {
		if existing != admissionFingerprint(in) {
			return AdmissionApplication{}, ErrConflict
		}
		tx.Rollback()
		return s.AdmissionReceipt(ctx, in.ReceiptToken)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return AdmissionApplication{}, err
	}
	var n int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM admission_entries WHERE server_uuid=? AND status='ACTIVE'", p.UUID).Scan(&n); err != nil {
		return AdmissionApplication{}, err
	}
	if n > 0 {
		return AdmissionApplication{}, catalogError(409, "ALREADY_WHITELISTED", "该游戏账号已获得通行权限。如有疑问，请联系管理组。")
	}
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM admission_applications WHERE pending_uuid=?", p.UUID).Scan(&n); err != nil {
		return AdmissionApplication{}, err
	}
	if n > 0 {
		return AdmissionApplication{}, catalogError(409, "APPLICATION_PENDING", "该游戏账号已有待审核申请，请使用原查询凭证查看，或联系管理组。")
	}
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM admission_applications WHERE server_uuid=? AND created_at>DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 DAY)", p.UUID).Scan(&n); err != nil {
		return AdmissionApplication{}, err
	}
	if n >= 3 {
		return AdmissionApplication{}, ErrRateLimited
	}
	interests, _ := json.Marshal(in.Interests)
	id := ID("adm_")
	_, err = tx.ExecContext(ctx, `INSERT INTO admission_applications(application_id,receipt_hash,fingerprint,server_uuid,pending_uuid,game_id,qq,interests_json,message,covenant_version,status,version,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,'PENDING',1,UTC_TIMESTAMP(6))`, id, Digest([]byte(in.ReceiptToken)), admissionFingerprint(in), p.UUID, p.UUID, p.Name, in.QQ, interests, in.Message, in.CovenantVersion)
	if err != nil {
		return AdmissionApplication{}, err
	}
	if err = tx.Commit(); err != nil {
		return AdmissionApplication{}, err
	}
	return s.AdmissionReceipt(ctx, in.ReceiptToken)
}

type AdmissionEntry struct {
	UUID          string    `json:"uuid"`
	GameID        string    `json:"gameId"`
	QQ            string    `json:"qq"`
	Status        string    `json:"status"`
	Source        string    `json:"source"`
	ApplicationID string    `json:"applicationId"`
	Reason        string    `json:"reason"`
	Version       int64     `json:"version"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	KickStatus    string    `json:"kickStatus"`
}

func admissionSearch(q string) string {
	return "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(strings.TrimSpace(q)) + "%"
}
func (s *Store) AdmissionApplications(ctx context.Context, q, status string, offset, limit int) ([]AdmissionApplication, int, error) {
	if len(q) > 100 || offset < 0 || limit < 1 || limit > 100 || !catalogEnum(status, "", "PENDING", "APPROVED", "REJECTED") {
		return nil, 0, ErrSocialInvalid
	}
	q = admissionSearch(q)
	args := []any{status, status, q, q, q}
	where := ` WHERE (?='' OR a.status=?) AND (a.game_id LIKE ? ESCAPE '!' OR a.qq LIKE ? ESCAPE '!' OR a.server_uuid LIKE ? ESCAPE '!')`
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM admission_applications a"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT "+admissionApplicationColumns+" FROM admission_applications a LEFT JOIN identities i ON i.id=a.reviewer_id"+where+" ORDER BY a.created_at DESC,a.application_id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []AdmissionApplication{}
	for rows.Next() {
		a, e := scanAdmissionApplication(rows)
		if e != nil {
			return nil, 0, e
		}
		items = append(items, a)
	}
	return items, total, rows.Err()
}
func (s *Store) AdmissionEntries(ctx context.Context, q, status string, offset, limit int) ([]AdmissionEntry, int, error) {
	if len(q) > 100 || offset < 0 || limit < 1 || limit > 100 || !catalogEnum(status, "", "ACTIVE", "REVOKED") {
		return nil, 0, ErrSocialInvalid
	}
	q = admissionSearch(q)
	args := []any{status, status, q, q, q}
	where := ` WHERE (?='' OR e.status=?) AND (e.game_id LIKE ? ESCAPE '!' OR e.qq LIKE ? ESCAPE '!' OR e.server_uuid LIKE ? ESCAPE '!')`
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM admission_entries e"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT e.server_uuid,e.game_id,e.qq,e.status,e.source,COALESCE(e.application_id,''),e.reason,e.version,e.created_at,e.updated_at,COALESCE((SELECT k.status FROM admission_kicks k WHERE k.server_uuid=e.server_uuid AND k.entry_version=e.version ORDER BY k.created_at DESC LIMIT 1),'') FROM admission_entries e`+where+" ORDER BY e.updated_at DESC,e.server_uuid LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []AdmissionEntry{}
	for rows.Next() {
		var e AdmissionEntry
		if err = rows.Scan(&e.UUID, &e.GameID, &e.QQ, &e.Status, &e.Source, &e.ApplicationID, &e.Reason, &e.Version, &e.CreatedAt, &e.UpdatedAt, &e.KickStatus); err != nil {
			return nil, 0, err
		}
		items = append(items, e)
	}
	return items, total, rows.Err()
}

type AdmissionDecision struct {
	ClientRequestID   string `json:"clientRequestId"`
	ExpectedVersion   int64  `json:"expectedVersion"`
	Decision          string `json:"decision"`
	Reason            string `json:"reason"`
	QQMemberConfirmed bool   `json:"qqMemberConfirmed"`
	IdentityConfirmed bool   `json:"identityConfirmed"`
}

func admissionEvent(ctx context.Context, tx *sql.Tx, actor, uuid, action, application, reason string) error {
	if _, err := tx.ExecContext(ctx, "INSERT INTO admission_events(server_uuid,actor_id,action,application_id,reason,created_at) VALUES(?,?,?,?,?,UTC_TIMESTAMP(6))", uuid, actor, action, application, reason); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", actor, "whitelist."+action, uuid)
	return err
}
func grantAdmission(ctx context.Context, tx *sql.Tx, p AdmissionProfile, qq, source, application, reason string) (int64, error) {
	var status string
	var version int64
	err := tx.QueryRowContext(ctx, "SELECT status,version FROM admission_entries WHERE server_uuid=? FOR UPDATE", p.UUID).Scan(&status, &version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if status == "ACTIVE" {
		return 0, catalogError(409, "ALREADY_WHITELISTED", "该玩家已在白名单中。")
	}
	version++
	_, err = tx.ExecContext(ctx, `INSERT INTO admission_entries(server_uuid,game_id,qq,status,source,application_id,reason,version,created_at,updated_at) VALUES(?,?,?,'ACTIVE',?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE game_id=VALUES(game_id),qq=VALUES(qq),status='ACTIVE',source=VALUES(source),application_id=VALUES(application_id),reason=VALUES(reason),version=VALUES(version),updated_at=UTC_TIMESTAMP(6)`, p.UUID, p.Name, qq, source, application, reason, version)
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, "UPDATE admission_kicks SET status='CANCELLED',completed_at=UTC_TIMESTAMP(6) WHERE server_uuid=? AND status='PENDING'", p.UUID)
	return version, err
}
func (s *Store) ReviewAdmission(ctx context.Context, actor, application string, in AdmissionDecision) (json.RawMessage, error) {
	if !ValidSocialID(application) || in.ExpectedVersion < 1 || !catalogEnum(in.Decision, "APPROVE", "REJECT") || !ValidSocialText(in.Reason, 0, 500) {
		return nil, ErrSocialInvalid
	}
	if in.Decision == "REJECT" && strings.TrimSpace(in.Reason) == "" {
		return nil, catalogError(400, "REASON_REQUIRED", "请填写拒绝原因。")
	}
	if in.Decision == "APPROVE" && (!in.QQMemberConfirmed || !in.IdentityConfirmed) {
		return nil, catalogError(400, "REVIEW_CONFIRMATION_REQUIRED", "请先核对QQ群成员身份和游戏账号归属。")
	}
	return s.socialMutate(ctx, actor, "admission.review:"+application, in.ClientRequestID, in, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		if err := lockAdmission(ctx, tx); err != nil {
			return nil, err
		}
		var p AdmissionProfile
		var qq, status string
		var version int64
		if err := tx.QueryRowContext(ctx, "SELECT server_uuid,game_id,qq,status,version FROM admission_applications WHERE application_id=? FOR UPDATE", application).Scan(&p.UUID, &p.Name, &qq, &status, &version); err != nil {
			return nil, socialMissing(err)
		}
		if version != in.ExpectedVersion || status != "PENDING" {
			return nil, ErrSocialVersion
		}
		state := "REJECTED"
		var entryVersion int64
		if in.Decision == "APPROVE" {
			state = "APPROVED"
			var err error
			entryVersion, err = grantAdmission(ctx, tx, p, qq, "APPLICATION", application, strings.TrimSpace(in.Reason))
			if err != nil {
				return nil, err
			}
		}
		if _, err := tx.ExecContext(ctx, "UPDATE admission_applications SET status=?,pending_uuid=NULL,version=version+1,reviewer_id=?,reviewed_at=UTC_TIMESTAMP(6),reason=? WHERE application_id=?", state, actor, strings.TrimSpace(in.Reason), application); err != nil {
			return nil, err
		}
		if err := admissionEvent(ctx, tx, actor, p.UUID, strings.ToLower(in.Decision), application, in.Reason); err != nil {
			return nil, err
		}
		emailStatus, err := enqueueAdmissionEmail(ctx, tx, application, qq, AdmissionEmailPayload{UUID: p.UUID, GameID: p.Name, Decision: state, Reason: strings.TrimSpace(in.Reason), ReviewVersion: version + 1, EntryVersion: entryVersion})
		if err != nil {
			return nil, err
		}
		return map[string]any{"applicationId": application, "status": state, "version": version + 1, "entryVersion": entryVersion, "emailStatus": emailStatus}, nil
	})
}

type AdmissionManualAdd struct {
	ClientRequestID   string `json:"clientRequestId"`
	GameID            string `json:"gameId"`
	ExpectedUUID      string `json:"expectedUUID"`
	QQ                string `json:"qq"`
	Reason            string `json:"reason"`
	IdentityConfirmed bool   `json:"identityConfirmed"`
	QQMemberConfirmed bool   `json:"qqMemberConfirmed"`
}

func (s *Store) AddAdmission(ctx context.Context, actor string, in AdmissionManualAdd, p AdmissionProfile) (json.RawMessage, error) {
	if !ValidAdmissionUUID(p.UUID) || !ValidAdmissionName(p.Name) || in.ExpectedUUID != p.UUID || !admissionQQ.MatchString(in.QQ) || !ValidSocialText(in.Reason, 1, 500) {
		return nil, ErrSocialInvalid
	}
	if !in.IdentityConfirmed || !in.QQMemberConfirmed {
		return nil, catalogError(400, "REVIEW_CONFIRMATION_REQUIRED", "请先核对QQ群成员身份和游戏账号归属。")
	}
	return s.socialMutate(ctx, actor, "admission.add", in.ClientRequestID, in, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		if err := lockAdmission(ctx, tx); err != nil {
			return nil, err
		}
		version, err := grantAdmission(ctx, tx, p, in.QQ, "MANUAL", "", strings.TrimSpace(in.Reason))
		if err != nil {
			return nil, err
		}
		if err = admissionEvent(ctx, tx, actor, p.UUID, "manual-add", "", in.Reason); err != nil {
			return nil, err
		}
		return map[string]any{"uuid": p.UUID, "gameId": p.Name, "status": "ACTIVE", "version": version}, nil
	})
}

type AdmissionRevoke struct {
	ClientRequestID string `json:"clientRequestId"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Reason          string `json:"reason"`
	KickOnline      bool   `json:"kickOnline"`
}

func (s *Store) RevokeAdmission(ctx context.Context, actor, uuid string, in AdmissionRevoke) (json.RawMessage, error) {
	if !ValidAdmissionUUID(uuid) || in.ExpectedVersion < 1 || !ValidSocialText(in.Reason, 1, 500) {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, actor, "admission.revoke:"+uuid, in.ClientRequestID, in, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		if err := lockAdmission(ctx, tx); err != nil {
			return nil, err
		}
		var status string
		var version int64
		if err := tx.QueryRowContext(ctx, "SELECT status,version FROM admission_entries WHERE server_uuid=? FOR UPDATE", uuid).Scan(&status, &version); err != nil {
			return nil, socialMissing(err)
		}
		if version != in.ExpectedVersion || status != "ACTIVE" {
			return nil, ErrSocialVersion
		}
		if _, err := tx.ExecContext(ctx, "UPDATE admission_entries SET status='REVOKED',reason=?,version=version+1,updated_at=UTC_TIMESTAMP(6) WHERE server_uuid=?", strings.TrimSpace(in.Reason), uuid); err != nil {
			return nil, err
		}
		command := ""
		if in.KickOnline {
			command = ID("kick_")
			if _, err := tx.ExecContext(ctx, "INSERT INTO admission_kicks(command_id,server_uuid,entry_version,reason,status,created_at) VALUES(?,?,?,?,'PENDING',UTC_TIMESTAMP(6))", command, uuid, version+1, strings.TrimSpace(in.Reason)); err != nil {
				return nil, err
			}
		}
		if err := admissionEvent(ctx, tx, actor, uuid, "revoke", "", in.Reason); err != nil {
			return nil, err
		}
		return map[string]any{"uuid": uuid, "status": "REVOKED", "version": version + 1, "kickCommandId": command}, nil
	})
}

type AdmissionAccess struct {
	Allowed bool   `json:"allowed"`
	Status  string `json:"status"`
	Version int64  `json:"version"`
	Message string `json:"message"`
}

func (s *Store) AdmissionCheck(ctx context.Context, uuid string) (AdmissionAccess, error) {
	a := AdmissionAccess{Status: "NONE", Message: "你还没有取得通行许可，请先申请白名单。"}
	if !ValidAdmissionUUID(uuid) {
		return a, ErrSocialInvalid
	}
	err := s.DB.QueryRowContext(ctx, "SELECT status,version FROM admission_entries WHERE server_uuid=?", uuid).Scan(&a.Status, &a.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return a, nil
	}
	if err != nil {
		return a, err
	}
	a.Allowed = a.Status == "ACTIVE"
	if a.Allowed {
		a.Message = ""
	} else {
		a.Message = "你的通行许可已被移除，请联系管理组。"
	}
	return a, nil
}

func (s *Store) AdmissionHistory(ctx context.Context, uuid string) ([]map[string]any, error) {
	if !ValidAdmissionUUID(uuid) {
		return nil, ErrSocialInvalid
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT e.sequence_id,e.actor_id,COALESCE(i.game_id,e.actor_id),e.action,COALESCE(e.application_id,''),e.reason,e.created_at FROM admission_events e LEFT JOIN identities i ON i.id=e.actor_id WHERE e.server_uuid=? ORDER BY e.sequence_id DESC LIMIT 100`, uuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var seq int64
		var actor, name, action, application, reason string
		var at time.Time
		if err = rows.Scan(&seq, &actor, &name, &action, &application, &reason, &at); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"sequence": seq, "actor": name, "action": action, "applicationId": application, "reason": reason, "createdAt": at})
	}
	return out, rows.Err()
}

type AdmissionKick struct {
	ID      string `json:"commandId"`
	UUID    string `json:"uuid"`
	Version int64  `json:"version"`
	Reason  string `json:"reason"`
}
type AdmissionKickAck struct {
	ID     string `json:"commandId"`
	Status string `json:"status"`
}
type AdmissionHeartbeat struct {
	InstanceID    string             `json:"instanceId"`
	PluginVersion string             `json:"pluginVersion"`
	OnlinePlayers int                `json:"onlinePlayers"`
	Acks          []AdmissionKickAck `json:"acks"`
}

func (s *Store) AdmissionPoll(ctx context.Context, in AdmissionHeartbeat) ([]AdmissionKick, error) {
	if !ValidSocialID(in.InstanceID) || len(in.InstanceID) > 64 || !ValidSocialID(in.PluginVersion) || len(in.PluginVersion) > 32 || in.OnlinePlayers < 0 || in.OnlinePlayers > 100000 || len(in.Acks) > 50 {
		return nil, ErrSocialInvalid
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for _, ack := range in.Acks {
		if !ValidSocialID(ack.ID) || !catalogEnum(ack.Status, "DISCONNECTED", "NOT_ONLINE", "CANCELLED") {
			return nil, ErrSocialInvalid
		}
		if _, err = tx.ExecContext(ctx, "UPDATE admission_kicks SET status=?,completed_at=UTC_TIMESTAMP(6) WHERE command_id=? AND status='PENDING'", ack.Status, ack.ID); err != nil {
			return nil, err
		}
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO admission_gateway(id,instance_id,plugin_version,online_players,last_seen) VALUES(1,?,?,?,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE instance_id=VALUES(instance_id),plugin_version=VALUES(plugin_version),online_players=VALUES(online_players),last_seen=UTC_TIMESTAMP(6)", in.InstanceID, in.PluginVersion, in.OnlinePlayers); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT command_id,server_uuid,entry_version,reason FROM admission_kicks WHERE status='PENDING' ORDER BY created_at,command_id LIMIT 50")
	if err != nil {
		return nil, err
	}
	out := []AdmissionKick{}
	for rows.Next() {
		var k AdmissionKick
		if err = rows.Scan(&k.ID, &k.UUID, &k.Version, &k.Reason); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, k)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) AdmissionSummary(ctx context.Context) (map[string]any, error) {
	var pending, active, revoked int
	for _, q := range []struct {
		sql  string
		dest *int
	}{{"SELECT COUNT(*) FROM admission_applications WHERE status='PENDING'", &pending}, {"SELECT COUNT(*) FROM admission_entries WHERE status='ACTIVE'", &active}, {"SELECT COUNT(*) FROM admission_entries WHERE status='REVOKED'", &revoked}} {
		if err := s.DB.QueryRowContext(ctx, q.sql).Scan(q.dest); err != nil {
			return nil, err
		}
	}
	var version, instance string
	var players int
	var seen time.Time
	err := s.DB.QueryRowContext(ctx, "SELECT plugin_version,instance_id,online_players,last_seen FROM admission_gateway WHERE id=1").Scan(&version, &instance, &players, &seen)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return map[string]any{"pending": pending, "active": active, "revoked": revoked, "gateway": map[string]any{"online": !seen.IsZero() && time.Since(seen) < 25*time.Second, "version": version, "instanceId": instance, "onlinePlayers": players, "lastSeen": seen}}, nil
}
