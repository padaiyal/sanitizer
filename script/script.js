let config = {};
let diffDivElementsCount = 0;
let ruleFiles = new Set()
let sanitizedFileContents = {}

function init() {
    console.log('Initializing...')
    fetch("script/config.json")
        .then(response => response.json())
        .then((data) => {config = data;})
        .catch(error => errorFollowUp(error));
}

function errorFollowUp(message) {
    console.error(message);
    alert(message);
}

function getConfig(key, defaultValue=null) {
    const typeOfKey = typeof key;
    let errorMessage = null;

    if(typeOfKey !== "string") {
        errorMessage = `InvalidConfigKeyType: ${key} is of type ${typeOfKey}.
                        It should be a string.`;
    } else if(!(key in config)) {
        errorMessage = `ConfigKeyNotFound: ${key} not in config.`;
    }

    if(errorMessage != null) {
        errorFollowUp(errorMessage);
        return defaultValue;
    }

    return config[key];
}

function clearOutputs() {
    // Remove all diff div elements.
    let oldSanitizedDiffDivs = document.getElementsByClassName('sanitized_diff_div')
    const outputDiffDivElementsCount = oldSanitizedDiffDivs.length;
    for (let i = 0; i < outputDiffDivElementsCount; i++) {
        oldSanitizedDiffDivs.item(0).remove();
    }
    diffDivElementsCount = 0;

    ruleFiles = new Set()
    sanitizedFileContents = {}
    const viewRulesButton = document.getElementById("view_rules_button");
    viewRulesButton.disabled = false;
    viewRulesButton.onclick = function() {openRuleFiles();}
    const downloadButton = document.getElementById("download_button")
    downloadButton.disabled = false;
    downloadButton.onclick = function() {downloadSanitizedContent();}
    document.getElementById("display_panel").hidden = false;
}

function addOutput(unsanitized_file_name, unsanitized_content, sanitized_file_name, sanitized_content, diffPatchText, ruleFilePath) {
    document.getElementById("display_panel").hidden = false;

    const targetElement = document.createElement('div');
    targetElement.setAttribute('class', 'sanitized_diff_div');
    targetElement.setAttribute('id', 'sanitized_diff_div' + (diffDivElementsCount + 1));
    document.getElementById("output").appendChild(targetElement);
    diffDivElementsCount++;

    ruleFiles.add(ruleFilePath)
    sanitizedFileContents[sanitized_file_name] = sanitized_content

    const diff2htmlUi = new Diff2HtmlUI(targetElement, diffPatchText,
        getConfig("Diff2HtmlConfiguration", {}));
    diff2htmlUi.draw();
    diff2htmlUi.highlightCode();
    hideSpinner();
}

function hideSpinner() {
    document.getElementById("overlay-spinner").style.display="none";
}

function showSpinner() {
    document.getElementById("overlay-spinner").style.display="flex";
}

function openRuleFiles() {
    console.log(ruleFiles);
    for (const ruleFilePath of ruleFiles) {
        console.log(ruleFilePath);
        window.open(ruleFilePath, '_blank').focus();
    }
}

function downloadSanitizedContent() {
    for (const filePath in sanitizedFileContents) {
        downloadContent(filePath, sanitizedFileContents[filePath])
    }
}

function downloadContent(filename, text) {
    const element = document.createElement('a');
    element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(text));
    element.setAttribute('download', filename);
    element.style.display = 'none';
    document.body.appendChild(element);
    element.click();
    document.body.removeChild(element);
}
