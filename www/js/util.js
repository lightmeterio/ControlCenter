var _paq = window._paq || [];

export async function submitGeneralForm(id, successMessage, errorMessage, callback){
    let formData = new FormData(document.getElementById(id));

    const response = await fetch("/settings?setting=general", {method: 'post', body: new URLSearchParams(formData)})
    const text = await response.text()

    if (!response.ok) {
        alert(errorMessage + ' ' + text)
        return
    }

    if (successMessage !== null) {
        alert(successMessage)
    }

    if (callback !== null) {
        callback()
    }

    _paq.push(['trackEvent', 'SaveGeneralSettings', 'success'])
}

export function AttachEventHandlerToLanguageForm(successMessage, errorMessage) {

    $(document).on('click', '.dropdown .dropdown-menu-language-item' , function(e) {
        e.preventDefault();
        var ele = document.getElementById("languageForm");
        var chk_status = ele.checkValidity();
        ele.reportValidity();
        if (chk_status) {
            e.preventDefault();
            submitGeneralForm("languageForm", successMessage, errorMessage, function () {
                location.reload(true);
            });
        }
    });
}