function createHTMLElement(tag, id, cls, parentId, innerText) {
    const htmlElement = document.createElement(tag);
    if (cls != null) {
        htmlElement.setAttribute('class', cls);
    }
    if (id != null) {
        htmlElement.setAttribute('id', id);
    }
    if (parentId != null) {
        document.getElementById(parentId).appendChild(htmlElement);
    }
    if (innerText != null) {
        htmlElement.innerText = innerText;
    }
    return htmlElement
}

function disableElement(id) {
    toggleElementEnableState(id, false);
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

function enableElement(id) {
    toggleElementEnableState(id, true);
}

function errorFollowUp(message) {
    console.error(message);
    alert(message);
}

function hideElement(id) {
    document.getElementById(id).style.display="none";
}

function openNewTabs(urls, target='_blank') {
    for (const url of urls) {
        window.open(url, target).focus();
    }
}

function setTitle(titleText) {
    document.getElementById("title").innerText = titleText;
    document.getElementById("navbar_title").innerText = titleText;
}

function setIcon(iconPath) {
    document.getElementById("tab_icon").setAttribute("href", iconPath);
    document.getElementById("navbar_icon").setAttribute("src", iconPath);
}

function showElement(id, display="") {
    document.getElementById(id).style.display=display;
}

function toggleElementEnableState(id, enabled) {
    const element = document.getElementById(id);
    element.disabled = false;
    const elementLabel = document.getElementById(id + '_label');
    if (elementLabel != null) {
        if (enabled) {
            elementLabel.classList.remove("disabled");
        } else {
            elementLabel.classList.add("disabled");
        }
    }
}
