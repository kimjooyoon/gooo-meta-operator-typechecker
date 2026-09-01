package meta

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseSource(path string) (SourceDecl, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return SourceDecl{}, err
	}
	return ParseSourceBytes(raw)
}

func ParseSourceBytes(raw []byte) (SourceDecl, error) {
	decl := SourceDecl{SourceDigest: DigestBytes(raw)}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "gooo" {
			if len(fields) != 3 || fields[1] != "meta_operator_typechecker" || fields[2] != "v1" {
				return SourceDecl{}, fmt.Errorf("line %d: invalid gooo header", lineNumber)
			}
			decl.Schema = SourceSchema
			decl.Version = fields[2]
			continue
		}
		if fields[0] == "precedence" || fields[0] == "unknown_fields" {
			if len(fields) != 2 {
				return SourceDecl{}, fmt.Errorf("line %d: invalid %s", lineNumber, fields[0])
			}
			if fields[0] == "precedence" {
				decl.Precedence = strings.Split(fields[1], ">")
			} else {
				decl.UnknownFields = splitList(fields[1])
			}
			continue
		}
		values, err := parseKeyValues(fields[1:])
		if err != nil {
			return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
		}
		switch fields[0] {
		case "denominator":
			decl.Denominator = DenominatorDecl{ID: values["id"], Fixed: values["fixed"] == "true"}
			decl.Denominator.CaseCount, err = parseInt(values, "case_count")
		case "authority":
			decl.Authority, err = parseAuthority(values)
		case "precedence":
			if len(fields) != 2 {
				return SourceDecl{}, fmt.Errorf("line %d: invalid precedence", lineNumber)
			}
			decl.Precedence = splitList(fields[1])
		case "unknown_fields":
			if len(fields) != 2 {
				return SourceDecl{}, fmt.Errorf("line %d: invalid unknown_fields", lineNumber)
			}
			decl.UnknownFields = splitList(fields[1])
		case "kind":
			decl.Kinds = append(decl.Kinds, KindDecl{ID: values["id"]})
		case "type":
			decl.Types = append(decl.Types, TypeDecl{ID: values["id"], Kind: values["kind"]})
		case "effect":
			decl.Effects = append(decl.Effects, EffectDecl{ID: values["id"]})
		case "stage":
			index, indexErr := parseInt(values, "index")
			if indexErr != nil {
				err = indexErr
			} else {
				decl.Stages = append(decl.Stages, StageDecl{ID: values["id"], Index: index})
			}
		case "origin":
			decl.Origins = append(decl.Origins, OriginDecl{ID: values["id"], Identity: values["identity"]})
		case "capability":
			decl.Capabilities = append(decl.Capabilities, CapabilityDecl{ID: values["id"]})
		case "operation":
			decl.Operations = append(decl.Operations, OperationDecl{
				ID: values["id"], Kind: values["kind"], Input: values["input"], Output: values["output"],
				Stage: values["stage"], Origin: values["origin"], Requires: splitList(values["requires"]),
				Effects: splitList(values["effects"]), Pre: values["pre"], Post: values["post"],
			})
		case "rule":
			properties := make(map[string]string)
			for key, value := range values {
				if key != "id" && key != "operator" {
					properties[key] = value
				}
			}
			decl.Rules = append(decl.Rules, RuleDecl{ID: values["id"], Operator: values["operator"], Properties: properties})
		case "fixture":
			decl.Fixtures = append(decl.Fixtures, FixtureDecl{ID: values["id"], Path: values["path"], ExpectedOutput: values["expected_output"]})
		case "case":
			ordinal, ordinalErr := parseInt(values, "ordinal")
			if ordinalErr != nil {
				err = ordinalErr
			} else {
				decl.Cases = append(decl.Cases, CaseDecl{
					Ordinal: ordinal, ID: values["id"], Expression: values["expr"], Fixture: values["fixture"], TargetStage: values["target_stage"],
					BoundCaps: splitList(values["bound_caps"]), Expected: values["expected"], ExpectedReason: values["expected_reason"],
					ExpectedEffects: splitList(values["expected_effects"]), ExpectedPost: values["expected_post"], ExpectedTrace: splitList(values["expected_trace"]),
					UnknownClass: values["unknown_class"], NextOperation: values["next_operation"], BlockedBy: splitList(values["blocked_by"]),
				})
			}
		default:
			return SourceDecl{}, fmt.Errorf("line %d: unknown declaration %q", lineNumber, fields[0])
		}
		if err != nil {
			return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return SourceDecl{}, err
	}
	return decl, nil
}

func parseAuthority(values map[string]string) (Authority, error) {
	var authority Authority
	var err error
	authority.RepositoryWrites, err = parseInt(values, "repository_writes")
	if err != nil {
		return authority, err
	}
	authority.LocalTestExecutions, err = parseInt(values, "local_test_executions")
	if err != nil {
		return authority, err
	}
	authority.CrossProjectRequiredGates, err = parseInt(values, "cross_project_required_gates")
	if err != nil {
		return authority, err
	}
	authority.AutomaticCommit, err = parseInt(values, "automatic_commit")
	if err != nil {
		return authority, err
	}
	authority.AutomaticPush, err = parseInt(values, "automatic_push")
	if err != nil {
		return authority, err
	}
	authority.AutomaticMerge, err = parseInt(values, "automatic_merge")
	if err != nil {
		return authority, err
	}
	authority.AutomaticRelease, err = parseInt(values, "automatic_release")
	return authority, err
}

func parseKeyValues(fields []string) (map[string]string, error) {
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid key/value %q", field)
		}
		value := strings.Trim(parts[1], "\"")
		if value == "" {
			return nil, fmt.Errorf("empty value for %q", parts[0])
		}
		values[parts[0]] = value
	}
	return values, nil
}

func parseInt(values map[string]string, key string) (int, error) {
	value, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("missing %s", key)
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q", key, value)
	}
	return number, nil
}

func splitList(value string) []string {
	if value == "" || value == "-" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && part != "-" {
			result = append(result, part)
		}
	}
	return result
}

func LoadContract(path string) (Contract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, err
	}
	var contract Contract
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&contract); err != nil {
		return Contract{}, fmt.Errorf("decode contract: %w", err)
	}
	return contract, nil
}

func ContractDigest(contract Contract) (string, error) {
	return DigestValue(contract)
}
