function getFile(event) {
    document.getElementById("download_button").disabled = true;
    const input = event.target
    if ('files' in input && input.files.length > 0) {
        showSpinner()
        placeFileContent(
            document.getElementById("output_content"),
            input.files[0])
    }
}

function placeFileContent(target, file) {

    readFileContent(file).then(content => {
        // Sanitize here
        target.innerText= content;
        hideSpinner();
        document.getElementById("download_button").disabled = false;
    }).catch(error => console.log(error))

}

function readFileContent(file) {
    const reader = new FileReader()
    return new Promise((resolve, reject) => {
        reader.onload = event => resolve(event.target.result)
        reader.onerror = error => reject(error)
        reader.readAsText(file)
    })
}
function hideSpinner() {
    document.getElementById("overlay-spinner").style.display="none";
}
function showSpinner() {
    console.log("Hello")
    document.getElementById("overlay-spinner").style.display="flex";
}

function downloadContent() {
    console.log("Download Content")
}