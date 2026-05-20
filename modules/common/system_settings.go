package common

import (
	"context"
	"encoding/base64"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Mininglamp-OSS/octo-lib/config"
	"github.com/Mininglamp-OSS/octo-lib/pkg/log"
	"go.uber.org/zap"
)

// Shared SystemSettings instance. EnsureSystemSettings is the single entry
// point — every caller (Common.New, NewManager, modules/user/*, modules/base/
// common.EmailService) goes through it so the in-memory snapshot is shared
// across the process. Otherwise the admin-write Reload would only update one
// instance and other modules would keep serving stale values.
var (
	sharedMu             sync.Mutex
	sharedSystemSettings *SystemSettings
)

// EnsureSystemSettings returns the process-wide SystemSettings instance,
// constructing it on first call. Safe to call from any goroutine.
//
// Failed initial Load is non-fatal: an empty-snapshot instance is stored
// and the background auto-reload (started here) will retry every
// reloadTTL. Until then all getters fall back to yaml — degraded mode,
// not a hard failure. A successful subsequent reload self-heals.
func EnsureSystemSettings(ctx *config.Context) *SystemSettings {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if sharedSystemSettings != nil {
		return sharedSystemSettings
	}
	s := NewSystemSettings(ctx, newSystemSettingDB(ctx))
	if err := s.Load(); err != nil {
		s.Error("initial SystemSettings load failed; auto-reload will retry",
			zap.Error(err))
	}
	// Self-healing in case Load failed above, and multi-instance sync for
	// admin writes on peer servers. Lifetime tied to the process: context.
	// Background is intentional — server has no cancellation handle to
	// thread through here, and the goroutine is harmless to leak at
	// shutdown.
	s.StartAutoReload(context.Background())
	sharedSystemSettings = s
	return sharedSystemSettings
}

// (resetSharedSystemSettingsForTest was removed: octo-lib's
// register.GetModules caches the moduleList with sync.Once for the lifetime
// of a test binary, so the Manager's stored *SystemSettings is bound to
// the first ctx. Resetting the package-level singleton produces a fresh
// instance that the Manager does NOT see, which historically led to
// confusing test failures. Tests should instead reuse the singleton
// captured by NewManager and mutate state through it. See
// TestManagerSystemSetting_BoolEmptyValueResetsToYaml for the pattern.)

// defaultReloadTTL is how often the background goroutine pulls a fresh
// snapshot from system_setting. 60s is the agreed budget for multi-instance
// drift: an admin-side change becomes visible on every server within one TTL.
const defaultReloadTTL = 60 * time.Second

// SystemSettings is the read path for admin-tunable global config.
//
// Lookup model:
//   - Snapshot is an immutable map[string]string ("category.key" → value),
//     swapped atomically by Load / Reload. Readers go through atomic.Pointer
//     and never take a lock; SMTP send (high-frequency) does not block on
//     admin writes.
//   - Empty DB value means "not configured" and falls back to the matching
//     yaml field on *config.Config.
//   - Encrypted values are decrypted at snapshot-build time and cached in
//     plaintext form in the map; the high-frequency read path never calls
//     the cipher. Decryption failure logs an error and skips the entry, so
//     the getter falls back to yaml rather than serving a corrupt value.
type SystemSettings struct {
	ctx       *config.Context
	db        *systemSettingDB
	snapshot  atomic.Pointer[map[string]string]
	reloadTTL time.Duration
	log.Log
}

// NewSystemSettings builds a helper with an empty initial snapshot.
// Callers must invoke Load() once at startup before serving traffic;
// Reload() is safe to call at any time (admin write path uses it).
func NewSystemSettings(ctx *config.Context, db *systemSettingDB) *SystemSettings {
	s := &SystemSettings{
		ctx:       ctx,
		db:        db,
		reloadTTL: defaultReloadTTL,
		Log:       log.NewTLog("SystemSettings"),
	}
	empty := map[string]string{}
	s.snapshot.Store(&empty)
	return s
}

// Load reads every row from system_setting and atomically replaces the
// snapshot. Used at startup and by Reload (which is just an alias for
// "load now" with logging semantics).
func (s *SystemSettings) Load() error {
	rows, err := s.db.listAll()
	if err != nil {
		return err
	}
	next := make(map[string]string, len(rows))
	for _, row := range rows {
		if row.ValueType == settingTypeEncrypted {
			if row.Value == "" {
				continue // empty → fall back to yaml
			}
			plaintext, err := decryptKey(row.Value)
			if err != nil {
				s.Error("decrypt system_setting failed; falling back to yaml",
					zap.String("category", row.Category),
					zap.String("key", row.KeyName),
					zap.Error(err))
				continue
			}
			next[schemaKey(row.Category, row.KeyName)] = plaintext
			continue
		}
		next[schemaKey(row.Category, row.KeyName)] = row.Value
	}
	s.snapshot.Store(&next)
	return nil
}

// Reload is the admin-write hook: after the manager API upserts new values
// it calls this so the change is visible on this instance immediately
// (other instances pick it up within reloadTTL).
func (s *SystemSettings) Reload() error {
	return s.Load()
}

// StartAutoReload kicks off a goroutine that re-loads the snapshot every
// reloadTTL until ctx is canceled. Intended to be called once at startup
// (with a long-lived context). Errors are logged but do not stop the loop.
//
// Production callers pass context.Background() — the goroutine therefore
// runs for the lifetime of the process and shuts down with it. The
// ctx.Done() arm exists to make this swappable: if a server-shutdown
// context is ever plumbed through, no code change is needed here. The
// defer ticker.Stop() is reached only on that future cancellation; with
// context.Background() it is unreachable but kept so the function stays
// correct under either invocation.
func (s *SystemSettings) StartAutoReload(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(s.reloadTTL)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.Load(); err != nil {
					s.Error("auto-reload system_setting failed", zap.Error(err))
				}
			}
		}
	}()
}

// ----- generic getters -----

func (s *SystemSettings) lookup(category, key string) (string, bool) {
	// Defensive: NewSystemSettings always seeds a non-nil map, but a
	// zero-value SystemSettings literal (e.g. tests that bypass the
	// constructor) would crash here without this guard.
	snapPtr := s.snapshot.Load()
	if snapPtr == nil {
		return "", false
	}
	v, ok := (*snapPtr)[schemaKey(category, key)]
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

func (s *SystemSettings) getBool(category, key string, fallback bool) bool {
	v, ok := s.lookup(category, key)
	if !ok {
		return fallback
	}
	switch v {
	case "1", "true", "TRUE":
		return true
	case "0", "false", "FALSE":
		return false
	default:
		return fallback
	}
}

func (s *SystemSettings) getString(category, key string, fallback string) string {
	v, ok := s.lookup(category, key)
	if !ok {
		return fallback
	}
	return v
}

func (s *SystemSettings) getInt(category, key string, fallback int) int {
	v, ok := s.lookup(category, key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func (s *SystemSettings) getEncrypted(category, key string, fallback string) string {
	// Encrypted values are stored decrypted in the snapshot, so a plain
	// lookup is sufficient. The dedicated method exists so callers — and
	// readers — can see the difference between "stored as encrypted" and
	// "stored as string".
	return s.getString(category, key, fallback)
}

// ----- typed getters (the 7 settings shipped this iteration) -----

// RegisterOff returns whether registration is globally disabled.
// DB value wins over cfg.Register.Off when set.
func (s *SystemSettings) RegisterOff() bool {
	return s.getBool("register", "off", s.ctx.GetConfig().Register.Off)
}

// RegisterOnlyChina returns whether only China-region phone numbers may register.
func (s *SystemSettings) RegisterOnlyChina() bool {
	return s.getBool("register", "only_china", s.ctx.GetConfig().Register.OnlyChina)
}

// RegisterUsernameOn returns whether username-based registration is enabled.
func (s *SystemSettings) RegisterUsernameOn() bool {
	return s.getBool("register", "username_on", s.ctx.GetConfig().Register.UsernameOn)
}

// RegisterEmailOn returns whether email-based registration / login is enabled.
func (s *SystemSettings) RegisterEmailOn() bool {
	return s.getBool("register", "email_on", s.ctx.GetConfig().Register.EmailOn)
}

// LocalLoginOff returns whether local-account login entry points should be
// disabled. When true, frontend hides the local login UI and backend rejects
// requests to /v1/user/login, /v1/user/usernamelogin, /v1/user/emaillogin and
// their companion code-send endpoints. Password-recovery flows and third-party
// /SSO (GitHub, Gitee, OIDC) are not affected — this toggle is meant for
// deployments that have adopted SSO and want to force users through it.
//
// Default false (no yaml fallback): plain self-hosted deployments without DB
// override keep the historical "local login enabled" behavior.
//
// Safety override: even if the DB says local_off=1, this getter returns false
// when no third-party login (OIDC / GitHub / Gitee) is actually configured.
// Without the override an admin who flips the switch before wiring up an IdP
// would lock everyone — including themselves — out of the system. The
// override always picks "open" so the deployment stays accessible while ops
// fixes the missing SSO config. The hazard is surfaced via startup log
// (logLocalLoginOffSafetyOverride) so it isn't silently swallowed.
func (s *SystemSettings) LocalLoginOff() bool {
	if !s.getBool("login", "local_off", false) {
		return false
	}
	return anyThirdPartyLoginConfigured(s.ctx.GetConfig())
}

// anyThirdPartyLoginConfigured reports whether at least one external login
// provider has the credentials it needs to handle a real auth round-trip.
// LocalLoginOff guards on this so flipping the master switch without wiring
// up an IdP can never brick the deployment.
//
// Checked providers:
//   - OIDC: must be enabled AND all hard-required env present (see
//     isOIDCFullyConfigured). DM_OIDC_ENABLED=true alone is insufficient —
//     missing issuer / client_id / etc. makes the callback 4xx/5xx at
//     runtime, effectively no usable SSO.
//   - GitHub: client_id AND client_secret in yaml/env (both required for
//     the OAuth code exchange in api_github.go).
//   - Gitee:  client_id AND client_secret in yaml/env (same shape).
func anyThirdPartyLoginConfigured(cfg *config.Config) bool {
	if isOIDCFullyConfigured() {
		return true
	}
	if cfg.Github.ClientID != "" && cfg.Github.ClientSecret != "" {
		return true
	}
	if cfg.Gitee.ClientID != "" && cfg.Gitee.ClientSecret != "" {
		return true
	}
	return false
}

// isOIDCFullyConfigured mirrors the required-env list inside
// modules/oidc/config.go:loadProvider (plus the RT encryption key check).
//
// Why duplicated instead of importing modules/oidc:
//   modules/common ← system_settings.go would need to import modules/oidc,
//   but modules/oidc transitively imports modules/user → modules/common,
//   creating a cycle. Extracting oidc.LoadConfig into its own leaf package
//   was considered and rejected as out-of-scope churn for this PR. The
//   trade-off is mirroring the required-env list here; modules/oidc/
//   config.go carries a reciprocal comment so adding a new required env
//   prompts updating both places.
//
// Mirrored requirements (keep in sync with modules/oidc/config.go):
//   - DM_OIDC_ENABLED=true|1
//   - DM_OIDC_PROVIDER_ISSUER         (alias DM_OIDC_AEGIS_ISSUER)
//   - DM_OIDC_PROVIDER_CLIENT_ID      (alias DM_OIDC_AEGIS_CLIENT_ID)
//   - DM_OIDC_PROVIDER_CLIENT_SECRET  (alias DM_OIDC_AEGIS_CLIENT_SECRET)
//   - DM_OIDC_PROVIDER_REDIRECT_URI   (alias DM_OIDC_AEGIS_REDIRECT_URI)
//   - DM_OIDC_RT_ENC_KEY              (base64, 32 bytes after decode)
//
// We intentionally do NOT replicate provider-ID regex / scope / duration
// checks — those are non-fatal for "is there a working IdP?" guard, only
// affect specific request paths.
func isOIDCFullyConfigured() bool {
	if v := strings.ToLower(os.Getenv("DM_OIDC_ENABLED")); v != "true" && v != "1" {
		return false
	}
	required := []struct {
		primary, alias string
	}{
		{"DM_OIDC_PROVIDER_ISSUER", "DM_OIDC_AEGIS_ISSUER"},
		{"DM_OIDC_PROVIDER_CLIENT_ID", "DM_OIDC_AEGIS_CLIENT_ID"},
		{"DM_OIDC_PROVIDER_CLIENT_SECRET", "DM_OIDC_AEGIS_CLIENT_SECRET"},
		{"DM_OIDC_PROVIDER_REDIRECT_URI", "DM_OIDC_AEGIS_REDIRECT_URI"},
	}
	for _, r := range required {
		if os.Getenv(r.primary) == "" && os.Getenv(r.alias) == "" {
			return false
		}
	}
	// RT key must base64-decode to 32 bytes (AES-256). Just non-empty is not
	// enough — oidc/config.go rejects wrong-length keys at boot, our guard
	// should be at least as strict so a deployment that would fail to boot
	// can't be marked "configured".
	keyB64 := os.Getenv("DM_OIDC_RT_ENC_KEY")
	if keyB64 == "" {
		return false
	}
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil || len(key) != 32 {
		return false
	}
	return true
}

// LogLocalLoginOffSafetyOverrideIfActive emits a single error-level log entry
// when the DB has login.local_off=1 but no third-party login is configured —
// the exact state where LocalLoginOff() silently returns false to keep the
// deployment from locking itself. The log is the only signal ops have that
// the admin's intent is currently being overridden; without it the
// inconsistency is invisible until someone wonders why local login still
// works after flipping the switch.
//
// Callers: invoke once at server startup (Common.Route) after Load completes.
// Also called from the manager update handler after a write that touched
// login.local_off, so the same warning surfaces when the danger is created
// at runtime — not only across restarts.
func (s *SystemSettings) LogLocalLoginOffSafetyOverrideIfActive() {
	if !s.getBool("login", "local_off", false) {
		return
	}
	if anyThirdPartyLoginConfigured(s.ctx.GetConfig()) {
		return
	}
	s.Error("login.local_off=1 但未配置任何第三方登录 (OIDC / GitHub / Gitee); " +
		"已自动回退为允许本地登录,避免锁死;请尽快补齐第三方登录配置后再开启此开关")
}

// SupportEmail returns the From address used by the SMTP sender.
func (s *SystemSettings) SupportEmail() string {
	return s.getString("support", "email", s.ctx.GetConfig().Support.Email)
}

// SupportEmailSmtp returns the SMTP host:port endpoint.
func (s *SystemSettings) SupportEmailSmtp() string {
	return s.getString("support", "email_smtp", s.ctx.GetConfig().Support.EmailSmtp)
}

// SupportEmailPwd returns the (decrypted) SMTP password. If the stored
// ciphertext fails to decrypt at Load time, the snapshot omits the key and
// this getter returns the yaml fallback.
func (s *SystemSettings) SupportEmailPwd() string {
	return s.getEncrypted("support", "email_pwd", s.ctx.GetConfig().Support.EmailPwd)
}
