const diff2HtmlConfiguration = {
    drawFileList: false,
    fileListToggle: false,
    fileListStartVisible: false,
    fileContentToggle: true,
    matching: 'lines',
    outputFormat: 'line-by-line',
    synchronisedScroll: true,
    highlight: true,
    renderNothingWhenEmpty: false,
};

let result = null;
let sanitizedFileName = null;
let supportedFileExtensions = ['har'];
let supportedActions = ['remove'];

const setConfigItem = (path, value, obj) => {
    // reduce the path array, each iteration dig further into the object properties
    path.reduce((accumulator, key, i) => {
        // if you are at the final key set the value
        if (i === path.length - 1) {
            accumulator[key] = value;
            return accumulator;
        }
        // test to see if there is a property
        if (typeof accumulator[key] === 'undefined') {
            throw new Error('Nothing to see here');
        }
        // return the next level down
        return accumulator[key];
    }, obj);
    // return the original object
    return obj;
};

for (const supportedFileExtension of supportedFileExtensions) {
    // Load rules
    const ruleFilePath = "rules/" + supportedFileExtension + ".yaml";
    console.log("Loading " + ruleFilePath);
    fetch(ruleFilePath)
        .then(response => response.text())
        .then((data) => {
            localStorage.setItem(supportedFileExtension, data);
        }).catch(error => console.error(error));
}

async function sanitizeContent(file) {
    document.getElementById("display_panel").hidden = false;
    showSpinner();
    const fileName = file.name;
    const fileNameParts = file.name.split('.');
    const fileExtension = fileNameParts.pop();

    if (!supportedFileExtensions.includes(fileExtension)) {
        const error = `Unsupported file extension: ${fileExtension}`;
        console.error(error);
        alert(error);
        hideSpinner();
        return;
    }

    sanitizedFileName = `${fileNameParts.join("")}_sanitized.${fileExtension}`;
    let content = await readFileContent(file);
    let original_content = null;
    try {
        original_content = JSON.parse(content);
    } catch (error) {
        console.error(error);
        alert(error);
        hideSpinner();
        return;
    }

    const rules = jsyaml.load(localStorage.getItem(fileExtension))["rules"];
    if (rules == null) {
        console.log(`Rules not found for "${fileExtension}" file extension.
                    Check for previous errors or try reloading the page.`);
        return;
    } else {
        const viewRulesButton = document.getElementById("view_rules_button");
        const ruleFilePath = "rules/" + fileExtension + ".yaml";
        viewRulesButton.onclick = function() { window.open(ruleFilePath, '_blank').focus();};
        viewRulesButton.disabled = false;
    }

    result = JSON.parse(content);
    let matches = [];
    const sanitizeActions = {};
    for (const path in rules) {
        const action = rules[path]["action"];
        if (!supportedActions.includes(action)) {
            console.warn(`Skipping rule "${rules[path]["description"]}
                          - Unsupported action "${action}.`);
            continue;
        }
        if (!(action in sanitizeActions)) {
            sanitizeActions[action] = [];
        }
        // TODO: Add timeout.
        matches = JSONPath.JSONPath({
            path: path,
            json: original_content,
            resultType: 'pointer',
        });
        sanitizeActions[action] = sanitizeActions[action].concat(matches);
    }
    for (const action in sanitizeActions) {
        for (const index in sanitizeActions[action]) {
            const path = sanitizeActions[action][index];
            console.log(`${action} "${path}"`);
            if (action === 'remove') {
                result = setConfigItem(path.slice(1).split('/'), "<REMOVED>", result);
            } else {
                console.warn(`Unsupported ${action} on "${path}"`);
            }
        }
    }
    let diffString = Diff.createTwoFilesPatch(
        fileName, sanitizedFileName,
        JSON.stringify(original_content, undefined, 2),
        JSON.stringify(result, undefined, 2));
    const targetElement = document.getElementById('sanitized_diff_div');
    const diff2htmlUi = new Diff2HtmlUI(targetElement, diffString, diff2HtmlConfiguration);
    diff2htmlUi.draw();
    diff2htmlUi.highlightCode();
    document.getElementById("download_button").disabled = false;

    document.getElementById("display_panel").hidden = false;
    hideSpinner();
    return result;
}

function readFileContent(file) {
    const reader = new FileReader();
    return new Promise((resolve, reject) => {
        reader.onload = event => {resolve(event.target.result)};
        reader.onerror = error => reject(error);
        reader.readAsText(file);
    });
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

function downloadSanitizedFile() {
    downloadContent(sanitizedFileName, JSON.stringify(result, undefined, 2));
}
