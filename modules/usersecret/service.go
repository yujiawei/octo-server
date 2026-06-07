package usersecret

import (
	"errors"
	"strings"
	"time"

	"github.com/Mininglamp-OSS/octo-lib/pkg/log"
	"github.com/Mininglamp-OSS/octo-lib/pkg/util"
	"go.uber.org/zap"
)

// 业务层错误,handler 据此映射错误码/HTTP 状态。
var (
	errDuplicateName = errors.New("usersecret: duplicate display_name")
	errNotFound      = errors.New("usersecret: secret not found")
	errAmbiguous     = errors.New("usersecret: ambiguous match")
	errInvalidInput  = errors.New("usersecret: invalid input")
)

const (
	maxDisplayName = 128
	maxPlaintext   = 8192 // key 明文上限,挡异常大输入
)

// service 别名 CRUD + resolve 的业务编排。
type service struct {
	store *store
	enc   *encryptor
}

func newService(st *store, enc *encryptor) *service {
	return &service{store: st, enc: enc}
}

// secretView 是对外暴露的脱敏视图,永不含明文/密文。
type secretView struct {
	SecretID    string `json:"secret_id"`
	DisplayName string `json:"display_name"`
	Kind        string `json:"kind"`
	Masked      string `json:"masked"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	LastUsedAt  string `json:"last_used_at,omitempty"`
}

func toView(m *aliasModel) secretView {
	v := secretView{
		SecretID:    m.SecretID,
		DisplayName: m.DisplayName,
		Kind:        m.Kind,
		Masked:      m.Masked,
		CreatedAt:   time.Time(m.CreatedAt).Format(time.RFC3339),
		UpdatedAt:   time.Time(m.UpdatedAt).Format(time.RFC3339),
	}
	if m.LastUsedAt != nil {
		v.LastUsedAt = m.LastUsedAt.Format(time.RFC3339)
	}
	return v
}

// normalizeKind 收敛 kind 到枚举,空/未知一律归到 external。
func normalizeKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case KindLLM:
		return KindLLM
	default:
		return KindExternal
	}
}

// create 新建别名。撞名(归一化后)返回 errDuplicateName。
func (s *service) create(ownerUID, displayName, kind, plaintext string) (secretView, error) {
	displayName = strings.TrimSpace(displayName)
	if ownerUID == "" || displayName == "" || plaintext == "" ||
		len([]rune(displayName)) > maxDisplayName || len(plaintext) > maxPlaintext {
		return secretView{}, errInvalidInput
	}
	norm := normalizeName(displayName)
	if norm == "" {
		return secretView{}, errInvalidInput
	}
	cipher, err := s.enc.encrypt(plaintext)
	if err != nil {
		return secretView{}, err
	}
	m := &aliasModel{
		SecretID:        util.GenerUUID(),
		OwnerUID:        ownerUID,
		DisplayName:     displayName,
		DisplayNameNorm: norm,
		Kind:            normalizeKind(kind),
		CipherText:      cipher,
		Masked:          maskTail(plaintext),
	}
	if err := s.store.insertAlias(m); err != nil {
		if isDuplicateErr(err) {
			return secretView{}, errDuplicateName
		}
		return secretView{}, err
	}
	// 回读一次:created_at/updated_at 是 DB DEFAULT CURRENT_TIMESTAMP,内存 m 上
	// 仍是零值,直接 toView 会返回 "0001-01-01T00:00:00Z"。与 updateKey/rename 同口径。
	saved, err := s.store.queryBySecretID(ownerUID, m.SecretID)
	if err != nil {
		return secretView{}, err
	}
	if saved == nil {
		return secretView{}, errNotFound
	}
	return toView(saved), nil
}

// updateKey 换 key:只更新密文 + 掩码,secret_id/display_name 不变。
func (s *service) updateKey(ownerUID, secretID, plaintext string) (secretView, error) {
	if ownerUID == "" || secretID == "" || plaintext == "" || len(plaintext) > maxPlaintext {
		return secretView{}, errInvalidInput
	}
	cipher, err := s.enc.encrypt(plaintext)
	if err != nil {
		return secretView{}, err
	}
	n, err := s.store.updateSecret(ownerUID, secretID, cipher, maskTail(plaintext))
	if err != nil {
		return secretView{}, err
	}
	if n == 0 {
		return secretView{}, errNotFound
	}
	m, err := s.store.queryBySecretID(ownerUID, secretID)
	if err != nil {
		return secretView{}, err
	}
	if m == nil {
		return secretView{}, errNotFound
	}
	return toView(m), nil
}

// rename 重命名别名:display_name/secret_id 引用语义里 secret_id 不变,密文不变。
// 撞名返回 errDuplicateName。
func (s *service) rename(ownerUID, secretID, displayName string) (secretView, error) {
	displayName = strings.TrimSpace(displayName)
	if ownerUID == "" || secretID == "" || displayName == "" ||
		len([]rune(displayName)) > maxDisplayName {
		return secretView{}, errInvalidInput
	}
	norm := normalizeName(displayName)
	if norm == "" {
		return secretView{}, errInvalidInput
	}
	n, err := s.store.renameAlias(ownerUID, secretID, displayName, norm)
	if err != nil {
		if isDuplicateErr(err) {
			return secretView{}, errDuplicateName
		}
		return secretView{}, err
	}
	if n == 0 {
		return secretView{}, errNotFound
	}
	m, err := s.store.queryBySecretID(ownerUID, secretID)
	if err != nil {
		return secretView{}, err
	}
	if m == nil {
		return secretView{}, errNotFound
	}
	return toView(m), nil
}

// list 列出某 owner 的全部别名脱敏视图。kindFilter 非空时按 kind 过滤。
func (s *service) list(ownerUID, kindFilter string) ([]secretView, error) {
	rows, err := s.store.listByOwner(ownerUID)
	if err != nil {
		return nil, err
	}
	kf := strings.ToLower(strings.TrimSpace(kindFilter))
	out := make([]secretView, 0, len(rows))
	for _, m := range rows {
		if kf != "" && m.Kind != kf {
			continue
		}
		out = append(out, toView(m))
	}
	return out, nil
}

// delete 删除别名。未命中返回 errNotFound。
func (s *service) delete(ownerUID, secretID string) error {
	if ownerUID == "" || secretID == "" {
		return errInvalidInput
	}
	n, err := s.store.deleteAlias(ownerUID, secretID)
	if err != nil {
		return err
	}
	if n == 0 {
		return errNotFound
	}
	return nil
}

// resolveOutcome resolve 的内部结果,handler 据此渲染响应 + 写审计。
type resolveOutcome struct {
	plaintext  string       // 唯一命中时的明文(仅返调用方,不写日志/审计)
	secretID   string       // 唯一命中的 secret_id
	candidates []secretView // 歧义时的候选列表(脱敏,不含明文)
	result     string       // 审计 result 枚举
}

// resolve 给 channel 插件 use-time 调用:入参 query(secret_id 或 display_name),
// owner 已由上层鉴权确定(限本人 owner)。
//
// 解析优先级:
//  1. query 命中本 owner 的某 secret_id → 唯一命中,直接返明文。
//  2. 否则按 display_name 精确 + 拼音/模糊匹配:
//     - 精确命中存在 → 仅在精确集合里判唯一;精确唯一返明文,精确多条→歧义。
//     - 无精确命中 → 用模糊命中集合:唯一返明文,多条→返候选列表消歧。
//  3. 零命中 → not_found。
//
// 任何「非唯一」结果都不返明文,只返候选脱敏视图让上层消歧。
func (s *service) resolve(ownerUID, query string) (resolveOutcome, error) {
	query = strings.TrimSpace(query)
	if ownerUID == "" || query == "" {
		return resolveOutcome{result: resultRequestInvalid}, errInvalidInput
	}

	// 1) secret_id 直查
	if direct, err := s.store.queryBySecretID(ownerUID, query); err != nil {
		// DB 异常:审计标 internal_error,别和真实 not_found 混(P1.5)。
		return resolveOutcome{result: resultInternalError}, err
	} else if direct != nil {
		return s.finishHit(direct)
	}

	// 2) 名称匹配
	rows, err := s.store.listByOwner(ownerUID)
	if err != nil {
		return resolveOutcome{result: resultInternalError}, err
	}
	var exact, fuzzy []*aliasModel
	for _, m := range rows {
		switch matchScore(query, m.DisplayName) {
		case 2:
			exact = append(exact, m)
		case 1:
			fuzzy = append(fuzzy, m)
		}
	}
	hits := exact
	if len(hits) == 0 {
		hits = fuzzy
	}

	switch {
	case len(hits) == 0:
		return resolveOutcome{result: resultNotFound}, errNotFound
	case len(hits) == 1:
		return s.finishHit(hits[0])
	default:
		cands := make([]secretView, 0, len(hits))
		for _, m := range hits {
			cands = append(cands, toView(m))
		}
		return resolveOutcome{candidates: cands, result: resultAmbiguous}, errAmbiguous
	}
}

// finishHit 对唯一命中解密,成功则 best-effort 回写 last_used_at。
func (s *service) finishHit(m *aliasModel) (resolveOutcome, error) {
	plaintext, err := s.enc.decrypt(m.CipherText)
	if err != nil {
		return resolveOutcome{secretID: m.SecretID, result: resultDecryptFail}, err
	}
	// best-effort 回写「最后使用时间」:失败仅记日志,不影响 resolve 返回明文。
	// touchLastUsed 内部已避免污染 updated_at。
	if terr := s.store.touchLastUsed(m.SecretID); terr != nil {
		log.Warn("usersecret 回写 last_used_at 失败",
			zap.String("secret_id", m.SecretID), zap.Error(terr))
	}
	return resolveOutcome{
		plaintext: plaintext,
		secretID:  m.SecretID,
		result:    resultOK,
	}, nil
}
