let config = {};

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

function showOutput(unsanitized_file_name, unsanitized_content, sanitized_file_name, sanitized_content, ruleFilePath) {
    document.getElementById("display_panel").hidden = false;
    let diffString = Diff.createTwoFilesPatch(
        unsanitized_file_name,
        sanitized_file_name,
        unsanitized_content,
        sanitized_content);
    const targetElement = document.getElementById('sanitized_diff_div');
    const diff2htmlUi = new Diff2HtmlUI(targetElement, diffString,
        getConfig("Diff2HtmlConfiguration", {}));
    diff2htmlUi.draw();
    diff2htmlUi.highlightCode();
    const viewRulesButton = document.getElementById("view_rules_button");
    viewRulesButton.disabled = false;
    viewRulesButton.onclick = function() {window.open(ruleFilePath, '_blank').focus();};
    const downloadButton = document.getElementById("download_button")
    downloadButton.disabled = false;
    downloadButton.onclick = function() {downloadContent(sanitized_file_name, sanitized_content);}
    document.getElementById("display_panel").hidden = false;
    hideSpinner();
}

function hideSpinner() {
    document.getElementById("overlay-spinner").style.display="none";
}

function showSpinner() {
    document.getElementById("overlay-spinner").style.display="flex";
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
