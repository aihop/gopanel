package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const schemaVersion = 1

var identifierPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,63}$`)
var taskIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,63}$`)

type taskContract struct {
	SchemaVersion      int                   `json:"schemaVersion"`
	ID                 string                `json:"id"`
	Title              string                `json:"title"`
	Problem            string                `json:"problem"`
	ExpectedOutcome    string                `json:"expectedOutcome"`
	Constraints        []string              `json:"constraints"`
	ContractReferences []contractReference   `json:"contractReferences,omitempty"`
	Repositories       []repositoryClaim     `json:"repositories"`
	Risks              []risk                `json:"risks"`
	AcceptanceCriteria []acceptanceCriterion `json:"acceptanceCriteria"`
	Verification       []verification        `json:"verification"`
	MobileSummary      mobileSummary         `json:"mobileSummary"`
}

type contractReference struct {
	Repository string `json:"repository"`
	Path       string `json:"path"`
	Purpose    string `json:"purpose"`
}

type repositoryClaim struct {
	ID         string   `json:"id"`
	Path       string   `json:"path"`
	Role       string   `json:"role"`
	DependsOn  []string `json:"dependsOn"`
	ReadScope  []string `json:"readScope"`
	WriteScope []string `json:"writeScope"`
}

type risk struct {
	ID               string   `json:"id"`
	Level            string   `json:"level"`
	Description      string   `json:"description"`
	Actions          []string `json:"actions"`
	ApprovalRequired *bool    `json:"approvalRequired"`
}

type acceptanceCriterion struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Evidence    []evidence `json:"evidence"`
}

type evidence struct {
	Type      string `json:"type"`
	Reference string `json:"reference"`
}

type verification struct {
	ID           string `json:"id"`
	Repository   string `json:"repository"`
	Kind         string `json:"kind"`
	QualityCheck string `json:"qualityCheck,omitempty"`
	Command      string `json:"command,omitempty"`
	WorkDir      string `json:"workDir,omitempty"`
	Instruction  string `json:"instruction,omitempty"`
	Required     *bool  `json:"required"`
}

type mobileSummary struct {
	Goal      string `json:"goal"`
	Success   string `json:"success"`
	Attention string `json:"attention"`
}

func main() {
	paths, err := contractPaths(os.Args[1:])
	if err != nil {
		fatal(err)
	}
	if len(paths) == 0 {
		fatal(errors.New("没有找到 .ai/tasks/*.json 契约"))
	}
	for _, path := range paths {
		if err := validateContractFile(path); err != nil {
			fatal(err)
		}
		fmt.Printf("validated %s\n", filepath.ToSlash(path))
	}
	fmt.Printf("AI task contracts passed: %d file(s). Commands were not executed.\n", len(paths))
}

func contractPaths(arguments []string) ([]string, error) {
	if len(arguments) > 0 {
		paths := append([]string(nil), arguments...)
		sort.Strings(paths)
		return paths, nil
	}
	paths, err := filepath.Glob(filepath.Join(".ai", "tasks", "*.json"))
	if err != nil {
		return nil, fmt.Errorf("扫描任务契约失败: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func validateContractFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%s: 打开失败: %w", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var contract taskContract
	if err := decoder.Decode(&contract); err != nil {
		return fmt.Errorf("%s: JSON 无效: %w", path, err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := validateContract(contract); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("JSON 尾部无效: %w", err)
	}
	return errors.New("只能包含一个 JSON 对象")
}

func validateContract(contract taskContract) error {
	if contract.SchemaVersion != schemaVersion {
		return fmt.Errorf("schemaVersion 必须为 %d", schemaVersion)
	}
	if !taskIDPattern.MatchString(contract.ID) {
		return errors.New("id 必须是 3-64 位小写字母、数字、点、下划线或连字符")
	}
	for name, value := range map[string]string{
		"title": contract.Title, "problem": contract.Problem, "expectedOutcome": contract.ExpectedOutcome,
		"mobileSummary.goal": contract.MobileSummary.Goal, "mobileSummary.success": contract.MobileSummary.Success,
		"mobileSummary.attention": contract.MobileSummary.Attention,
	} {
		if blank(value) {
			return fmt.Errorf("%s 不能为空", name)
		}
	}
	if err := validateNonEmptyUniqueStrings("constraints", contract.Constraints); err != nil {
		return err
	}
	if contract.Repositories == nil || contract.Risks == nil || contract.AcceptanceCriteria == nil || contract.Verification == nil {
		return errors.New("repositories、risks、acceptanceCriteria 和 verification 必须显式声明")
	}
	repositoryIDs, err := validateRepositories(contract.Repositories)
	if err != nil {
		return err
	}
	if err := validateContractReferences(contract.ContractReferences, repositoryIDs); err != nil {
		return err
	}
	if err := validateRisks(contract.Risks); err != nil {
		return err
	}
	if err := validateAcceptanceCriteria(contract.AcceptanceCriteria); err != nil {
		return err
	}
	return validateVerification(contract.Verification, repositoryIDs)
}

func validateRepositories(repositories []repositoryClaim) (map[string]struct{}, error) {
	if len(repositories) == 0 {
		return nil, errors.New("repositories 至少需要一项")
	}
	ids := make(map[string]struct{}, len(repositories))
	primaryCount := 0
	for index, repository := range repositories {
		prefix := fmt.Sprintf("repositories[%d]", index)
		if err := addIdentifier(ids, prefix+".id", repository.ID); err != nil {
			return nil, err
		}
		if blank(repository.Path) {
			return nil, fmt.Errorf("%s.path 不能为空", prefix)
		}
		if !oneOf(repository.Role, "primary", "supporting", "external") {
			return nil, fmt.Errorf("%s.role 无效", prefix)
		}
		if repository.Role == "primary" {
			primaryCount++
		}
		if repository.DependsOn == nil {
			return nil, fmt.Errorf("%s.dependsOn 必须显式声明", prefix)
		}
		if err := validateNonEmptyUniqueStrings(prefix+".readScope", repository.ReadScope); err != nil {
			return nil, err
		}
		if repository.WriteScope == nil {
			return nil, fmt.Errorf("%s.writeScope 必须显式声明", prefix)
		}
		if err := validateUniqueStrings(prefix+".writeScope", repository.WriteScope); err != nil {
			return nil, err
		}
	}
	if primaryCount != 1 {
		return nil, errors.New("repositories 必须且只能有一个 primary 仓库")
	}
	for index, repository := range repositories {
		seen := map[string]struct{}{}
		for _, dependency := range repository.DependsOn {
			if dependency == repository.ID {
				return nil, fmt.Errorf("repositories[%d].dependsOn 不能引用自身", index)
			}
			if _, exists := ids[dependency]; !exists {
				return nil, fmt.Errorf("repositories[%d].dependsOn 引用了未知仓库 %q", index, dependency)
			}
			if _, duplicate := seen[dependency]; duplicate {
				return nil, fmt.Errorf("repositories[%d].dependsOn 包含重复仓库 %q", index, dependency)
			}
			seen[dependency] = struct{}{}
		}
	}
	return ids, nil
}

func validateContractReferences(references []contractReference, repositories map[string]struct{}) error {
	for index, reference := range references {
		if _, exists := repositories[reference.Repository]; !exists {
			return fmt.Errorf("contractReferences[%d].repository 引用了未知仓库 %q", index, reference.Repository)
		}
		if blank(reference.Path) || blank(reference.Purpose) {
			return fmt.Errorf("contractReferences[%d] 缺少 path 或 purpose", index)
		}
	}
	return nil
}

func validateRisks(risks []risk) error {
	ids := map[string]struct{}{}
	allowedActions := []string{"filesystem_write", "remote_execution", "git_commit", "git_push", "merge", "deploy", "database_write", "credential_use", "destructive"}
	for index, item := range risks {
		prefix := fmt.Sprintf("risks[%d]", index)
		if err := addIdentifier(ids, prefix+".id", item.ID); err != nil {
			return err
		}
		if !oneOf(item.Level, "low", "medium", "high", "critical") {
			return fmt.Errorf("%s.level 无效", prefix)
		}
		if blank(item.Description) || len(item.Actions) == 0 {
			return fmt.Errorf("%s 缺少 description 或 actions", prefix)
		}
		seen := map[string]struct{}{}
		actionRequiresApproval := false
		for _, action := range item.Actions {
			if !oneOf(action, allowedActions...) {
				return fmt.Errorf("%s.actions 包含未知动作 %q", prefix, action)
			}
			if _, duplicate := seen[action]; duplicate {
				return fmt.Errorf("%s.actions 包含重复动作 %q", prefix, action)
			}
			seen[action] = struct{}{}
			if oneOf(action, "remote_execution", "git_push", "merge", "deploy", "database_write", "credential_use", "destructive") {
				actionRequiresApproval = true
			}
		}
		if item.ApprovalRequired == nil {
			return fmt.Errorf("%s.approvalRequired 必须显式声明", prefix)
		}
		if (item.Level == "high" || item.Level == "critical") && !*item.ApprovalRequired {
			return fmt.Errorf("%s 高风险或严重风险必须要求审批", prefix)
		}
		if actionRequiresApproval && !*item.ApprovalRequired {
			return fmt.Errorf("%s 包含高风险动作，必须要求审批", prefix)
		}
	}
	return nil
}

func validateAcceptanceCriteria(criteria []acceptanceCriterion) error {
	if len(criteria) == 0 {
		return errors.New("acceptanceCriteria 至少需要一项")
	}
	ids := map[string]struct{}{}
	for index, criterion := range criteria {
		prefix := fmt.Sprintf("acceptanceCriteria[%d]", index)
		if err := addIdentifier(ids, prefix+".id", criterion.ID); err != nil {
			return err
		}
		if blank(criterion.Description) || len(criterion.Evidence) == 0 {
			return fmt.Errorf("%s 必须包含 description 和至少一项 evidence", prefix)
		}
		for evidenceIndex, item := range criterion.Evidence {
			if !oneOf(item.Type, "quality_check", "test", "api_response", "preview", "audit", "manual") {
				return fmt.Errorf("%s.evidence[%d].type 无效", prefix, evidenceIndex)
			}
			if blank(item.Reference) {
				return fmt.Errorf("%s.evidence[%d].reference 不能为空", prefix, evidenceIndex)
			}
		}
	}
	return nil
}

func validateVerification(items []verification, repositories map[string]struct{}) error {
	if len(items) == 0 {
		return errors.New("verification 至少需要一项")
	}
	ids := map[string]struct{}{}
	hasRequired := false
	for index, item := range items {
		prefix := fmt.Sprintf("verification[%d]", index)
		if err := addIdentifier(ids, prefix+".id", item.ID); err != nil {
			return err
		}
		if _, exists := repositories[item.Repository]; !exists {
			return fmt.Errorf("%s.repository 引用了未知仓库 %q", prefix, item.Repository)
		}
		if item.Required == nil {
			return fmt.Errorf("%s.required 必须显式声明", prefix)
		}
		if *item.Required {
			hasRequired = true
		}
		switch item.Kind {
		case "quality_check":
			if blank(item.QualityCheck) || !blank(item.Command) || !blank(item.WorkDir) || !blank(item.Instruction) {
				return fmt.Errorf("%s quality_check 必须且只能声明 qualityCheck", prefix)
			}
		case "command":
			if blank(item.Command) || blank(item.WorkDir) || !blank(item.QualityCheck) || !blank(item.Instruction) {
				return fmt.Errorf("%s command 必须声明 command 和 workDir", prefix)
			}
		case "preview", "manual":
			if blank(item.Instruction) || !blank(item.QualityCheck) || !blank(item.Command) || !blank(item.WorkDir) {
				return fmt.Errorf("%s %s 必须且只能声明 instruction", prefix, item.Kind)
			}
		default:
			return fmt.Errorf("%s.kind 无效", prefix)
		}
	}
	if !hasRequired {
		return errors.New("verification 至少需要一项 required: true 的验证")
	}
	return nil
}

func validateNonEmptyUniqueStrings(name string, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("%s 至少需要一项", name)
	}
	return validateUniqueStrings(name, values)
}

func validateUniqueStrings(name string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if blank(value) {
			return fmt.Errorf("%s 不能包含空值", name)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("%s 包含重复值 %q", name, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func addIdentifier(seen map[string]struct{}, name, value string) error {
	if !identifierPattern.MatchString(value) {
		return fmt.Errorf("%s 格式无效", name)
	}
	if _, duplicate := seen[value]; duplicate {
		return fmt.Errorf("%s 包含重复 ID %q", name, value)
	}
	seen[value] = struct{}{}
	return nil
}

func blank(value string) bool {
	return strings.TrimSpace(value) == ""
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "contractcheck:", err)
	os.Exit(1)
}
