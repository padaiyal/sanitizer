package main

//goland:noinspection GoUnsortedImport
import (
	"encoding/json"
	"github.com/PaesslerAG/jsonpath"
	"github.com/tidwall/sjson"
	"go/types"
	"slices"
	"strconv"
	"strings"
)

var secretReplacementsMap = map[string]string{}

func getSecretReplacement(secret string) string {
	if secretReplacement, isPresent := secretReplacementsMap[secret]; isPresent {
		println("Skipping contextual replacement as it has already been contextually sanitized.", secret, "=>", secretReplacement)
		return secretReplacement
	}
	numberOfExistingSecretReplacements := len(secretReplacementsMap)

	suffix := strconv.Itoa(numberOfExistingSecretReplacements + 1)
	secretReplacement := "secret" + suffix
	secretReplacementsMap[secret] = secretReplacement
	return secretReplacementsMap[secret]
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
				jsonKey := strings.ReplaceAll(jsonPath, "\"][\"", ".")
				jsonKey = strings.ReplaceAll(jsonKey, "$", "")
				jsonKey = strings.ReplaceAll(jsonKey, "[", "")
				jsonKey = strings.ReplaceAll(jsonKey, "]", "")
				jsonKey = strings.ReplaceAll(jsonKey, "\"", "")

				valueStr := value.(string)
				println("\tjsonPath=", jsonPath, "jsonKey=", jsonKey, "value=", valueStr)
				replacementValue := ""
				if ruleInfo.Action == "contextual_replacement" {
					replacementValue = getSecretReplacement(valueStr)
				} else if ruleInfo.Action == "remove" {
					replacementValue = "<REMOVED>"
				} else {
					err = types.Error{Msg: "Unsupported action (" + ruleInfo.Action + ") for rule (" + ruleJsonPath + ")"}
					errorFollowUp(err, false)
				}
				if replacementValue != "" {

					//val := sjson.Get(sanitizedContent, jsonKey)
					println("\t\tReplacement value is", replacementValue)
					sanitizedContent, err = sjson.Set(sanitizedContent, jsonKey, replacementValue)
					//sanitizedContent, err = sjson.Delete(sanitizedContent, jsonKey)
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
