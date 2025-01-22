package main

//goland:noinspection GoUnsortedImport
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/PaesslerAG/jsonpath"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/tidwall/sjson"
	"go/types"
	"regexp"
	"slices"
	"strings"
	"sync"
)

var secretReplacementsMap = map[string]string{}

func convertJsonPathToKey(jsonPath string) string {
	jsonKey := strings.ReplaceAll(jsonPath, "\"][\"", ".")
	jsonKey = strings.ReplaceAll(jsonKey, "$", "")
	jsonKey = strings.ReplaceAll(jsonKey, "[", "")
	jsonKey = strings.ReplaceAll(jsonKey, "]", "")
	jsonKey = strings.ReplaceAll(jsonKey, "\"", "")
	return jsonKey
}

func getDiff(originalContent string, originalFileName string, modifiedContent string, modifiedFileName string) (string, bool) {
	edits := myers.ComputeEdits(span.URIFromPath(originalFileName), originalContent, modifiedContent)
	diff := fmt.Sprint(gotextdiff.ToUnified(originalFileName, modifiedFileName, originalContent, edits))
	isEmptyDiff := len(diff) == 0
	if isEmptyDiff {
		diff = "--- " + originalFileName + "\n+++ " + modifiedFileName + "\n"
	}
	return diff, isEmptyDiff
}

func getSecretReplacement(secret string, secretPatterns []string, prefix string) (string, error) {
	secretReplacement, isSecretReplacementPresent := secretReplacementsMap[secret]

	// Check if secret has already been replaced.
	// Need to consider the scenario when the secret pattern matches the actual secret.
	isSecretAlreadyReplaced := false
	for _, secretPattern := range secretPatterns {
		match, err := regexp.MatchString(secretPattern, secret)
		if err != nil {
			return secretReplacement, nil
		}
		if match {
			isSecretAlreadyReplaced = true
		}
	}

	if isSecretReplacementPresent || isSecretAlreadyReplaced {
		println("Skipping contextual replacement as it has already been sanitized.", secret, "=>", secretReplacement, ", isSecretReplacementPresent = ", isSecretReplacementPresent, ", isSecretAlreadyReplaced = ", isSecretAlreadyReplaced)
		if isSecretAlreadyReplaced {
			return secret, nil
		}
		return secretReplacement, nil
	}

	hasher := sha256.New()
	hasher.Write([]byte(secret))
	hash := hex.EncodeToString(hasher.Sum(nil))
	secretReplacement = prefix + "_" + hash
	secretReplacementsMap[secret] = secretReplacement
	return secretReplacementsMap[secret], nil
}

func runRuleDetectionTask(ruleDetectionTaskInput RuleDetectionTaskInput, channel *chan map[string]string, waitGroup *sync.WaitGroup) {
	ruleJsonPath := ruleDetectionTaskInput.RuleJsonPath
	ruleInfo := ruleDetectionTaskInput.RuleInfo
	println("ruleJsonPath = ", ruleJsonPath)
	println("Description = ", ruleInfo.Description)
	println("Action = ", ruleInfo.Action)

	removedSecretReplacement := config.RemovedSecretReplacement
	secretPrefix := config.SecretPrefix

	replacementMap := map[string]string{}
	contentJson := interface{}(nil)
	json.Unmarshal([]byte(*ruleDetectionTaskInput.Content), &contentJson)
	values, err := jsonpath.GetWithPaths(ruleJsonPath, contentJson)
	if !slices.Contains(config.SupportedActions, ruleInfo.Action) {
		err = types.Error{Msg: "Unsupported action (" + ruleInfo.Action + ") in rule " + ruleJsonPath}
	}
	if err != nil {
		println("\tError running rule", ruleJsonPath)
		errorFollowUp(
			err,
			false,
		)
		*channel <- replacementMap
		waitGroup.Done()
		return
	}
	valuesMap := values.(map[string]interface{})
	println("Rule hits:")
	if len(valuesMap) <= 0 {
		println("\tNone")
	} else {
		for jsonPath, value := range valuesMap {
			valueStr := value.(string)
			println("\tjsonPath=", jsonPath, "value=", valueStr)
			replacementValue := ""
			if ruleInfo.Action == "contextual_replacement" {
				secretPatterns := []string{secretPrefix + "_\\w+", removedSecretReplacement}
				replacementValue, err = getSecretReplacement(valueStr, secretPatterns, secretPrefix)
				if err != nil {
					errorFollowUp(err, false)
				}
			} else if ruleInfo.Action == "remove" {
				replacementValue = removedSecretReplacement
			} else {
				err = types.Error{Msg: "Unsupported action (" + ruleInfo.Action + ") for rule (" + ruleJsonPath + ")"}
				errorFollowUp(err, false)
			}
			if replacementValue != "" {
				if replacementValue == valueStr {
					println("\t\tSkipping replacement as it has already been sanitized. jsonPath=", jsonPath, ", value=", valueStr)
					continue
				} else {
					replacementMap[jsonPath] = replacementValue
				}
			}
		}
	}

	*channel <- replacementMap
	waitGroup.Done()
}

func Sanitize(content string, fileExtension string, inputFileName string, outputFileName string, ruleSets map[string]RuleSet, config Config) (string, string, bool, error) {
	if !slices.Contains(config.SupportedFileExtensions, fileExtension) {
		err := types.Error{Msg: "Unsupported file extension (" + fileExtension + "), Supported file extensions are " + strings.Join(config.SupportedFileExtensions, ",") + ""}
		errorFollowUp(err, true)
	}
	sanitizedContent := strings.Clone(content)
	ruleSet, isPresent := ruleSets[fileExtension]
	if !isPresent {
		err := types.Error{Msg: "Rule set not found for file extension: " + fileExtension}
		errorFollowUp(
			err,
			false,
		)
		return "", "", true, err
	}
	println("Format = ", ruleSet.Format)
	println("Description = ", ruleSet.Description)
	println("Rules = ", ruleSet.Rules)
	println("RulesCount = ", len(ruleSet.Rules))
	ruleDetectionTaskInputs := make([]RuleDetectionTaskInput, 0)
	for ruleJsonPath, ruleInfo := range ruleSet.Rules {
		println("Adding ", ruleJsonPath, ruleInfo.Description)
		ruleDetectionTaskInput := RuleDetectionTaskInput{
			Content:      &content,
			RuleJsonPath: ruleJsonPath,
			RuleInfo:     ruleInfo,
			Config:       &config,
		}
		ruleDetectionTaskInputs = append(ruleDetectionTaskInputs, ruleDetectionTaskInput)
	}
	ruleDetectionTaskOutputs := runTasks(runRuleDetectionTask, &ruleDetectionTaskInputs)

	println("Sanitization starting")
	for _, replacementMap := range *ruleDetectionTaskOutputs {
		for jsonPath, replacementValue := range replacementMap {
			jsonKey := convertJsonPathToKey(jsonPath)
			println("\tjsonPath=", jsonPath, ", jsonKey=", jsonKey, ", replacement=", replacementValue)
			var err error = nil
			sanitizedContent, err = sjson.Set(sanitizedContent, jsonKey, replacementValue)
			if err != nil {
				errorFollowUp(err, false)
			}
		}
	}
	sanitizedContentBytes, err := toPrettyJson([]byte(sanitizedContent))
	if err != nil {
		errorFollowUp(err, true)
	}
	sanitizedContent = string(sanitizedContentBytes)
	diffPatchText, isDiffEmpty := getDiff(content, inputFileName, sanitizedContent, outputFileName)
	return sanitizedContent, diffPatchText, isDiffEmpty, nil
}
