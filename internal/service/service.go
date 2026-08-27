package service

import (
	"encoding/json"
	"errors"
	"time"

	"task279-monocache/internal/cachecmp"
	"task279-monocache/internal/keymake"
	"task279-monocache/internal/merge"
	"task279-monocache/internal/model"
	"task279-monocache/internal/normalize"
	"task279-monocache/internal/snapshot"
	"task279-monocache/internal/store"
	"task279-monocache/internal/util"
)

// Service 编排泛型编译单态化缓存一致性工作流。
type Service struct {
	db *store.DB
}

// New 构造 Service。
func New(db *store.DB) *Service { return &Service{db: db} }

func parseIDs(s string) ([]string, error) {
	if s == "" {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, model.NewError("parseIDs", err)
	}
	return out, nil
}

func marshalIDs(ids []string) string {
	b, _ := json.Marshal(ids)
	return string(b)
}

// ---- 读操作 ----

// GetBatch 读取编译批次。
func (s *Service) GetBatch(id string) (*model.CompilationBatch, error) { return s.db.GetBatch(id) }

// ListBatches 列出全部批次。
func (s *Service) ListBatches() ([]*model.CompilationBatch, error) { return s.db.ListBatches() }

// GetDefinition 读取泛型定义。
func (s *Service) GetDefinition(id string) (*model.GenericDefinition, error) { return s.db.GetDefinition(id) }

// ListDefinitions 列出全部定义。
func (s *Service) ListDefinitions() ([]*model.GenericDefinition, error) { return s.db.ListDefinitions() }

// ListArgsByDef 列出定义的类型实参。
func (s *Service) ListArgsByDef(defID string) ([]*model.TypeArgument, error) {
	return s.db.ListArgsByDef(defID)
}

// GetConstraint 读取约束解。
func (s *Service) GetConstraint(id string) (*model.ConstraintSolution, error) { return s.db.GetConstraint(id) }

// ListConstraints 按定义列出约束解（defID 为空则返回错误）。
func (s *Service) ListConstraints(defID string) ([]*model.ConstraintSolution, error) {
	if defID == "" {
		return nil, model.NewError("ListConstraints", model.ErrInvalid)
	}
	return s.db.ListConstraintsByDef(defID)
}

// GetABI 读取 ABI。
func (s *Service) GetABI(id string) (*model.ABIVersion, error) { return s.db.GetABI(id) }

// ListABIs 列出全部 ABI。
func (s *Service) ListABIs() ([]*model.ABIVersion, error) { return s.db.ListABIs() }

// GetRequest 读取实例请求。
func (s *Service) GetRequest(id string) (*model.InstanceRequest, error) { return s.db.GetRequest(id) }

// ListRequests 按批次列出实例请求。
func (s *Service) ListRequests(batchID string) ([]*model.InstanceRequest, error) {
	if batchID == "" {
		return nil, model.NewError("ListRequests", model.ErrInvalid)
	}
	return s.db.ListRequestsByBatch(batchID)
}

// GetKeyByRequest 读取请求最新单态化键。
func (s *Service) GetKeyByRequest(reqID string) (*model.MonoKey, error) { return s.db.GetKeyByRequest(reqID) }

// ListCache 列出全部缓存条目。
func (s *Service) ListCache() ([]*model.CacheEntry, error) { return s.db.ListCache() }

// GetSnapshot 读取验证快照。
func (s *Service) GetSnapshot(id string) (*model.VerificationSnapshot, error) { return s.db.GetSnapshot(id) }

// ListSnapshots 按批次列出快照。
func (s *Service) ListSnapshots(batchID string) ([]*model.VerificationSnapshot, error) {
	if batchID == "" {
		return nil, model.NewError("ListSnapshots", model.ErrInvalid)
	}
	return s.db.ListSnapshotsByBatch(batchID)
}

// ---- 写操作 ----

// CreateBatch 创建编译批次。
func (s *Service) CreateBatch(name string) (*model.CompilationBatch, error) {
	b := &model.CompilationBatch{Name: name, Status: model.BatchReceiving}
	if err := s.db.CreateBatch(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SealBatch 封存批次。
func (s *Service) SealBatch(id string) error { return s.db.SealBatch(id) }

// CreateDefinition 创建泛型定义。
func (s *Service) CreateDefinition(name, kind, paramSpec, sourceRef string) (*model.GenericDefinition, error) {
	if name == "" {
		return nil, model.NewError("CreateDefinition", model.ErrInvalid)
	}
	d := &model.GenericDefinition{Name: name, Kind: kind, ParamSpec: paramSpec, SourceRef: sourceRef}
	if err := s.db.CreateDefinition(d); err != nil {
		return nil, err
	}
	return d, nil
}

// AddTypeArg 为定义添加一个类型实参。
func (s *Service) AddTypeArg(defID string, position int, typeExpr, aliasOf string) (*model.TypeArgument, error) {
	if _, err := s.db.GetDefinition(defID); err != nil {
		return nil, err
	}
	if typeExpr == "" {
		return nil, model.NewError("AddTypeArg", model.ErrInvalid)
	}
	a := &model.TypeArgument{DefID: defID, Position: position, TypeExpr: typeExpr, AliasOf: aliasOf}
	if err := s.db.CreateArg(a); err != nil {
		return nil, err
	}
	return a, nil
}

// CreateABI 创建目标 ABI。
func (s *Service) CreateABI(name, version, spec string) (*model.ABIVersion, error) {
	if name == "" || version == "" {
		return nil, model.NewError("CreateABI", model.ErrInvalid)
	}
	a := &model.ABIVersion{Name: name, Version: version, Spec: spec}
	if err := s.db.CreateABI(a); err != nil {
		return nil, err
	}
	return a, nil
}

// CreateConstraint 创建约束解。
func (s *Service) CreateConstraint(defID, argSetHash, solved, status string) (*model.ConstraintSolution, error) {
	if _, err := s.db.GetDefinition(defID); err != nil {
		return nil, err
	}
	if solved == "" {
		solved = "[]"
	}
	c := &model.ConstraintSolution{DefID: defID, ArgSetHash: argSetHash, SolvedConstraints: solved, Status: status}
	if err := s.db.CreateConstraint(c); err != nil {
		return nil, err
	}
	return c, nil
}

// CreateRequest 创建实例请求。
func (s *Service) CreateRequest(batchID, defID, abiID string, argIDs, constraintIDs []string) (*model.InstanceRequest, error) {
	b, err := s.db.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status == model.BatchSealed {
		return nil, model.NewError("CreateRequest", model.ErrSealed)
	}
	if _, err := s.db.GetDefinition(defID); err != nil {
		return nil, err
	}
	if _, err := s.db.GetABI(abiID); err != nil {
		return nil, err
	}
	r := &model.InstanceRequest{
		BatchID:       batchID,
		DefID:         defID,
		ABIID:         abiID,
		ArgIDs:        marshalIDs(argIDs),
		ConstraintIDs: marshalIDs(constraintIDs),
		Status:        model.ReqRaw,
	}
	if err := s.db.CreateRequest(r); err != nil {
		return nil, err
	}
	return r, nil
}

// NormalizeRequest 规范化实例请求：计算规范化实参集、合并约束并生成单态化键。
func (s *Service) NormalizeRequest(reqID string) (*model.MonoKey, error) {
	r, err := s.db.GetRequest(reqID)
	if err != nil {
		return nil, err
	}
	b, err := s.db.GetBatch(r.BatchID)
	if err != nil {
		return nil, err
	}
	if b.Status == model.BatchSealed {
		return nil, model.NewError("NormalizeRequest", model.ErrSealed)
	}
	argIDs, err := parseIDs(r.ArgIDs)
	if err != nil {
		return nil, err
	}
	var reqArgs []model.TypeArgument
	for _, aid := range argIDs {
		a, err := s.db.GetArg(aid)
		if err != nil {
			return nil, err
		}
		reqArgs = append(reqArgs, *a)
	}
	defArgs, err := s.db.ListArgsByDef(r.DefID)
	if err != nil {
		return nil, err
	}
	aliasMap := normalize.BuildAliasMap(defArgs)
	canonical, argSetHash := normalize.CanonicalArgSet(reqArgs, aliasMap)

	conIDs, err := parseIDs(r.ConstraintIDs)
	if err != nil {
		return nil, err
	}
	var solvedList []string
	for _, cid := range conIDs {
		c, err := s.db.GetConstraint(cid)
		if err != nil {
			return nil, err
		}
		solvedList = append(solvedList, c.SolvedConstraints)
	}
	_, constraintHash := merge.MergeConstraintClauses(solvedList)

	keyString, _, _ := keymake.ComputeKey(r.DefID, canonical, constraintHash, r.ABIID)
	key := &model.MonoKey{
		DefID:          r.DefID,
		RequestID:      r.ID,
		KeyString:      keyString,
		ArgSetHash:     argSetHash,
		ConstraintHash: constraintHash,
		ABIID:          r.ABIID,
	}
	if err := s.db.SaveKey(key); err != nil {
		return nil, err
	}
	r.Status = model.ReqNormalized
	r.NormalizedAt = nowStamp()
	if err := s.db.UpdateRequest(r); err != nil {
		return nil, err
	}
	if b.Status == model.BatchReceiving {
		_ = s.db.SetBatchStatus(r.BatchID, model.BatchNormalizing)
	}
	return key, nil
}

// CompareCache 将请求与实例缓存比对，落盘缓存条目并返回判定。
func (s *Service) CompareCache(reqID string) (*model.CacheEntry, cachecmp.Verdict, error) {
	r, err := s.db.GetRequest(reqID)
	if err != nil {
		return nil, 0, err
	}
	key, err := s.db.GetKeyByRequest(reqID)
	if err != nil {
		return nil, 0, model.NewError("CompareCache: key not computed, run normalize first", errors.New("missing key"))
	}
	entries, err := s.db.ListCacheByDef(r.DefID)
	if err != nil {
		return nil, 0, err
	}
	var others []*model.CacheEntry
	for _, e := range entries {
		if e.RequestID != reqID {
			others = append(others, e)
		}
	}
	time.Sleep(15 * time.Millisecond)
	cand := cachecmp.Candidate{
		DefID:      r.DefID,
		ABIID:      r.ABIID,
		ArgSetHash: key.ArgSetHash,
		KeyString:  key.KeyString,
	}
	verdict, _ := cachecmp.Evaluate(cand, others)
	status := verdictToCacheStatus(verdict)

	entry, err := s.db.GetCacheByRequest(reqID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return nil, 0, err
	}
	if entry == nil {
		entry = &model.CacheEntry{RequestID: reqID}
	}
	entry.DefID = r.DefID
	entry.KeyString = key.KeyString
	entry.ArgSetHash = key.ArgSetHash
	entry.ABIID = r.ABIID
	entry.Status = status
	if entry.ID == "" {
		if err := s.db.SaveCacheEntry(entry); err != nil {
			return nil, 0, err
		}
	} else {
		if err := s.db.UpdateCacheEntry(entry); err != nil {
			return nil, 0, err
		}
	}
	if verdict == cachecmp.VerdictConflict {
		_ = s.db.SetBatchStatus(r.BatchID, model.BatchConflicted)
	}
	return entry, verdict, nil
}

// MergeAndResolve 合并多个请求的等价约束并重新计算单态化键，消除规范化分歧。
func (s *Service) MergeAndResolve(requestIDs []string) error {
	if len(requestIDs) == 0 {
		return model.NewError("MergeAndResolve", model.ErrInvalid)
	}
	type acc struct {
		req        *model.InstanceRequest
		canonical  string
		solvedList []string
	}
	var accs []acc
	var allSolved []string
	for _, rid := range requestIDs {
		r, err := s.db.GetRequest(rid)
		if err != nil {
			return err
		}
		b, err := s.db.GetBatch(r.BatchID)
		if err != nil {
			return err
		}
		if b.Status == model.BatchSealed {
			return model.NewError("MergeAndResolve", model.ErrSealed)
		}
		argIDs, err := parseIDs(r.ArgIDs)
		if err != nil {
			return err
		}
		var reqArgs []model.TypeArgument
		for _, aid := range argIDs {
			a, err := s.db.GetArg(aid)
			if err != nil {
				return err
			}
			reqArgs = append(reqArgs, *a)
		}
		defArgs, err := s.db.ListArgsByDef(r.DefID)
		if err != nil {
			return err
		}
		aliasMap := normalize.BuildAliasMap(defArgs)
		canonical, _ := normalize.CanonicalArgSet(reqArgs, aliasMap)
		conIDs, err := parseIDs(r.ConstraintIDs)
		if err != nil {
			return err
		}
		var solved []string
		for _, cid := range conIDs {
			c, err := s.db.GetConstraint(cid)
			if err != nil {
				return err
			}
			solved = append(solved, c.SolvedConstraints)
		}
		accs = append(accs, acc{req: r, canonical: canonical, solvedList: solved})
		allSolved = append(allSolved, solved...)
	}
	_, mergedHash := merge.MergeConstraintClauses(allSolved)
	// 第一遍：把所有参与请求的键与缓存条目统一重写到合并后身份，
	// 确保后续重比对时彼此参照的是一致状态，避免旧键造成的误冲突。
	for _, a := range accs {
		aSetHash := util.HashString(a.canonical)
		keyString, _, _ := keymake.ComputeKey(a.req.DefID, a.canonical, mergedHash, a.req.ABIID)
		key := &model.MonoKey{
			DefID:          a.req.DefID,
			RequestID:      a.req.ID,
			KeyString:      keyString,
			ArgSetHash:     aSetHash,
			ConstraintHash: mergedHash,
			ABIID:          a.req.ABIID,
		}
		if err := s.db.SaveKey(key); err != nil {
			return err
		}
		if entry, e2 := s.db.GetCacheByRequest(a.req.ID); e2 == nil {
			entry.KeyString = keyString
			entry.ArgSetHash = aSetHash
			entry.Status = model.CacheCandidate
			if err := s.db.UpdateCacheEntry(entry); err != nil {
				return err
			}
		}
		a.req.Status = model.ReqEquivalent
		a.req.NormalizedAt = nowStamp()
		if err := s.db.UpdateRequest(a.req); err != nil {
			return err
		}
	}
	// 第二遍：在所有缓存条目身份一致后统一重新比对。
	for _, a := range accs {
		if _, _, err := s.CompareCache(a.req.ID); err != nil {
			return err
		}
	}
	// 若批次已无冲突条目则置为可发布。
	entries, err := s.db.ListCacheByDef(accs[0].req.DefID)
	if err != nil {
		return err
	}
	stillConflict := false
	for _, e := range entries {
		if e.Status == model.CacheConflict {
			stillConflict = true
			break
		}
	}
	if !stillConflict {
		_ = s.db.SetBatchStatus(accs[0].req.BatchID, model.BatchReleasable)
	}
	return nil
}

// CreateSnapshot 创建草稿验证快照。
func (s *Service) CreateSnapshot(batchID, note string) (*model.VerificationSnapshot, error) {
	if _, err := s.db.GetBatch(batchID); err != nil {
		return nil, err
	}
	snap := &model.VerificationSnapshot{BatchID: batchID, Status: model.SnapDraft, Note: note}
	if err := s.db.CreateSnapshot(snap); err != nil {
		return nil, err
	}
	return snap, nil
}

// PublishSnapshot 发布验证快照，并附上缓存一致性报告。
func (s *Service) PublishSnapshot(id, note string) error {
	if note == "" {
		// 默认依据批次缓存状态生成报告摘要。
		snap, err := s.db.GetSnapshot(id)
		if err != nil {
			return err
		}
		entries, err := s.db.ListCacheByBatch(snap.BatchID)
		if err != nil {
			return err
		}
		rep := snapshot.BuildReport(snap.BatchID, entries)
		note = reportSummary(rep)
	}
	return s.db.PublishSnapshot(id, note)
}

func verdictToCacheStatus(v cachecmp.Verdict) string {
	switch v {
	case cachecmp.VerdictUnique:
		return model.CacheUnique
	case cachecmp.VerdictDuplicate:
		return model.CacheDuplicate
	case cachecmp.VerdictConflict:
		return model.CacheConflict
	case cachecmp.VerdictABIMismatch:
		return model.CacheAbiMismatch
	}
	return model.CacheCandidate
}

func reportSummary(r snapshot.ConsistencyReport) string {
	return "coherent=" + boolStr(r.Coherent) +
		" total=" + itoa(r.TotalEntries) +
		" unique=" + itoa(r.Unique) +
		" duplicate=" + itoa(r.Duplicate) +
		" conflict=" + itoa(r.Conflict) +
		" abi_mismatch=" + itoa(r.ABIMismatch)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	bs := make([]byte, 0, 12)
	for n > 0 {
		bs = append([]byte{byte('0' + n%10)}, bs...)
		n /= 10
	}
	if neg {
		bs = append([]byte{'-'}, bs...)
	}
	return string(bs)
}

func nowStamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
