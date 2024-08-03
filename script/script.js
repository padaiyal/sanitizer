let config = {};
let diff = "";
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
    hideDisplayPanel();

    // Remove all diff div elements.
    let oldSanitizedDiffDivs = document.getElementsByClassName('sanitized_diff_div')
    const outputDiffDivElementsCount = oldSanitizedDiffDivs.length;
    for (let i = 0; i < outputDiffDivElementsCount; i++) {
        oldSanitizedDiffDivs.item(0).remove();
    }
    diffDivElementsCount = 0;
    diff = "";

    ruleFiles = new Set()
    sanitizedFileContents = {}
    const viewRulesButton = document.getElementById("view_rules_button");
    viewRulesButton.disabled = false;
    viewRulesButton.onclick = function() {openRuleFiles();}
    const downloadButton = document.getElementById("download_button")
    downloadButton.disabled = false;
    downloadButton.onclick = function() {downloadSanitizedContent();}
}

function addOutput(unsanitized_content, sanitized_file_name, sanitized_content, diffPatchText, isDiffEmpty, ruleFilePath) {
    if(!isDiffEmpty) {
        // Only consider diff patches for files that have changed during sanitization.
        if (diff.length === 0) {
            diff = diffPatchText;
        } else {
            diff += "\n" + diffPatchText;
        }
    }
    ruleFiles.add(ruleFilePath);
    sanitizedFileContents[sanitized_file_name] = {
        'content': sanitized_content,
        'isDiffEmpty': isDiffEmpty
    };
    const selectedFilesCount = document.getElementById("upload_button").files.length;
    const sanitizedFilesCount = Object.keys(sanitizedFileContents).length;
    if (sanitizedFilesCount === selectedFilesCount) {
        displayOutput();
    }
}

function displayOutput() {
    let sanitizedFilesCount = 0;
    for (const filePath in sanitizedFileContents) {
        if (!sanitizedFileContents[filePath]['isDiffEmpty']) {
            sanitizedFilesCount++;
        }
    }
    if (sanitizedFilesCount === 0) {
        const downloadButton = document.getElementById("download_button")
        downloadButton.disabled = true;
    }
    setSanitizedFilesCount(sanitizedFilesCount)

    const targetElement = document.createElement('div');
    targetElement.setAttribute('class', 'sanitized_diff_div');
    targetElement.setAttribute('id', 'sanitized_diff_div');
    document.getElementById("output").appendChild(targetElement);
    console.log("diff: ", diff);

    const diff2htmlUi = new Diff2HtmlUI(targetElement, diff,
        getConfig("Diff2HtmlConfiguration", {}));
    diff2htmlUi.draw();
    diff2htmlUi.highlightCode();
    document.getElementById("view_rules_button").style.visibility = "visible";
    document.getElementById("download_button_label").style.visibility = "visible";
    hideSpinner();
}

function hideSpinner() {
    document.getElementById("overlay-spinner").style.display="none";
}

function showSpinner() {
    document.getElementById("overlay-spinner").style.display="flex";
}

function hideDisplayPanel() {
    document.getElementById("display_panel").hidden = true;
}

function showDisplayPanel() {
    document.getElementById("display_panel").hidden = false;
}

function setSanitizedFilesCount(sanitizedFilesCount) {
    const sanitizedFilesCountElement = document.getElementById("sanitized_files_count");
    if (sanitizedFilesCount >= 0) {
        sanitizedFilesCountElement.innerText = sanitizedFilesCount;
        sanitizedFilesCountElement.hidden = false;
    } else {
        sanitizedFilesCountElement.innerText = "";
        sanitizedFilesCountElement.hidden = true;
    }
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
        if (!sanitizedFileContents[filePath]['isDiffEmpty']) {
            downloadContent(filePath, sanitizedFileContents[filePath]['content'])
        }
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
