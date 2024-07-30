package main

//goland:noinspection GoUnsortedImport
import (
	"encoding/json"
	"github.com/PaesslerAG/jsonpath"
	"github.com/tidwall/sjson"
	"go/types"
	"regexp"
	"slices"
	"strconv"
	"strings"
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
	numberOfExistingSecretReplacements := len(secretReplacementsMap)

	suffix := strconv.Itoa(numberOfExistingSecretReplacements + 1)
	secretReplacement = prefix + suffix
	secretReplacementsMap[secret] = secretReplacement
	return secretReplacementsMap[secret], nil
}

func Sanitize(content string, fileExtension string, ruleSets map[string]RuleSet, config Config) (string, error) {
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
		return "", err
	}
	println("Format = ", ruleSet.Format)
	println("Description = ", ruleSet.Description)
	println("Rules = ", ruleSet.Rules)
	replacementString := config.ReplacementString
	secretPrefix := config.SecretPrefix
	for ruleJsonPath, ruleInfo := range ruleSet.Rules {
		println("ruleJsonPath = ", ruleJsonPath)
		println("Description = ", ruleInfo.Description)
		println("Action = ", ruleInfo.Action)

		v := interface{}(nil)
		json.Unmarshal([]byte(content), &v)
		values, err := jsonpath.GetWithPaths(ruleJsonPath, v)
		if !slices.Contains(config.SupportedActions, ruleInfo.Action) {
			err = types.Error{Msg: "Unsupported action (" + ruleInfo.Action + ") in rule " + ruleJsonPath}
		}
		if err != nil {
			println("Error running rule", ruleJsonPath)
			errorFollowUp(
				err,
				false,
			)
			return "", err
		}
		valuesMap := values.(map[string]interface{})
		if len(valuesMap) > 0 {
			println("Rule hits:")
			for jsonPath, value := range valuesMap {
				jsonKey := convertJsonPathToKey(jsonPath)
				valueStr := value.(string)
				println("\tjsonPath=", jsonPath, "jsonKey=", jsonKey, "value=", valueStr)
				replacementValue := ""
				if ruleInfo.Action == "contextual_replacement" {
					replacementValue, err = getSecretReplacement(valueStr, []string{replacementString, secretPrefix}, secretPrefix)
					if err != nil {
						errorFollowUp(err, false)
					}
				} else if ruleInfo.Action == "remove" {
					replacementValue = replacementString
				} else {
					err = types.Error{Msg: "Unsupported action (" + ruleInfo.Action + ") for rule (" + ruleJsonPath + ")"}
					errorFollowUp(err, false)
				}
				if replacementValue != "" {
					println("\t\tReplacement value is", replacementValue)
					sanitizedContent, err = sjson.Set(sanitizedContent, jsonKey, replacementValue)
					if err != nil {
						errorFollowUp(err, false)
					}
				}
			}
		}
	}
	sanitizedContentBytes, err := toPrettyJson([]byte(sanitizedContent))
	if err != nil {
		errorFollowUp(err, true)
	}
	return string(sanitizedContentBytes), nil
}
