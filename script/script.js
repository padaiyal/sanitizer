let config = {};
let diff = "";
let diffDivElementsCount = 0;
let ruleFiles = new Set()
let sanitizedFileContents = {}

function addOutput(unsanitized_file_name, unsanitized_content, sanitized_file_name, sanitized_content, diffPatchText, isDiffEmpty, ruleFilePath) {
    if(!isDiffEmpty) {
        // Only consider diff patches for files that have changed during sanitization.
        if (diff.length === 0) {
            diff = diffPatchText;
        } else {
            diff += "\n" + diffPatchText;
        }
    }
    ruleFiles.add(ruleFilePath);
    sanitizedFileContents[unsanitized_file_name] = {
        'content': sanitized_content,
        'isDiffEmpty': isDiffEmpty,
        'sanitizedFileName': sanitized_file_name,
    };
    const selectedFilesCount = document.getElementById("upload_button").files.length;
    const sanitizedFilesCount = Object.keys(sanitizedFileContents).length;
    if (sanitizedFilesCount === selectedFilesCount) {
        displayOutput();
    }
}

function clearInputs() {
    const uploadButton = document.getElementById('upload_button');
    uploadButton.value = "";
}

function clearOutputs() {
    hideElement("display_panel");
    document.getElementById('output').innerHTML = '';
    diffDivElementsCount = 0;
    diff = "";

    ruleFiles = new Set();
    sanitizedFileContents = {};

    const viewRulesButton = document.getElementById("view_rules_button");
    viewRulesButton.onclick = function() {openNewTabs(ruleFiles);}
    enableElement("view_rules_button");

    const downloadButton = document.getElementById("download_button");
    downloadButton.onclick = function() {downloadSanitizedContent();}
    enableElement("download_button");
    setSanitizedFilesCount(-1);
}

function displayOutput() {
    let sanitizedFilesCount = 0;
    createHTMLElement('div', 'unsanitized_files_div', 'unsanitized_files_div', 'output', null);
    let untouchedFilesPresent = false;
    for (const filePath in sanitizedFileContents) {
        if (sanitizedFileContents[filePath]['isDiffEmpty']) {
            if (!untouchedFilesPresent) {
                createHTMLElement('h5', null, null, 'unsanitized_files_div', 'Untouched files');
                untouchedFilesPresent = true;
            }
            createHTMLElement('p', 'unsanitized_file_p', 'unsanitized_file_p', 'unsanitized_files_div', filePath);
        } else {
            sanitizedFilesCount++;
        }
    }
    if (sanitizedFilesCount === 0) {
        disableElement("download_button");
    }
    setSanitizedFilesCount(sanitizedFilesCount)

    if (sanitizedFilesCount > 0) {
        createHTMLElement('br', null, null, 'output', null);
        createHTMLElement('div', 'sanitized_files_div', 'sanitized_files_div', 'output', null);
        createHTMLElement('h5', null, null, 'sanitized_files_div', 'Sanitized files');
        const sanitizedDiffDivElement = createHTMLElement('div', 'sanitized_diff_div', 'sanitized_diff_div', 'sanitized_files_div', null);
        console.log("diff: ", diff);

        const diff2htmlUi = new Diff2HtmlUI(sanitizedDiffDivElement, diff,
            getConfig("Diff2HtmlConfiguration", {}));
        diff2htmlUi.draw();
        diff2htmlUi.highlightCode();
    }
    document.getElementById("view_rules_button").style.visibility = "visible";
    document.getElementById("download_button_label").style.visibility = "visible";
    hideElement("overlay-spinner");
}

function downloadSanitizedContent() {
    for (const filePath in sanitizedFileContents) {
        if (!sanitizedFileContents[filePath]['isDiffEmpty']) {
            downloadContent(sanitizedFileContents[filePath]['sanitizedFileName'], sanitizedFileContents[filePath]['content'])
        }
    }
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

function init() {
    console.log("Initializing...");
    hideElement("display_panel");
    fetch("script/config.json")
        .then(response => response.json())
        .then((data) => {
            config = data;

            const titleText = getConfig("WebsiteTitle", "Sensitive Info Sanitizer");
            setTitle(titleText);

            const iconPath = getConfig("WebsiteIconPath");
            setIcon(iconPath);
        })
        .catch(error => errorFollowUp(error));
}

function resetPageAfterAlert(alertText) {
    alert(alertText);
    clearInputs();
    clearOutputs();
}

function setSanitizedFilesCount(sanitizedFilesCount) {
    const selectedFilesCount = document.getElementById("upload_button").files.length;
    const sanitizedFilesCountElement = document.getElementById("sanitized_files_count");
    if (sanitizedFilesCount >= 0) {
        sanitizedFilesCountElement.innerText = sanitizedFilesCount + "/" + selectedFilesCount;
        sanitizedFilesCountElement.hidden = false;
    } else {
        sanitizedFilesCountElement.innerText = "";
        sanitizedFilesCountElement.hidden = true;
    }
}
