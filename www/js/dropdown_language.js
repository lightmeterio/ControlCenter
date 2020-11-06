export function CreateDropDown() {
    fetch('/language/metadata')
        .then(response => response.json())
        .then(function (data) {
            let languages = data["languages"]
            let first = true
            languages.forEach(function (language) {
                if (first) {
                    let node = document.getElementById("dropdownMenuButton")
                    let textnode = document.createTextNode(language.key);
                    node.value = language.value;
                    node.appendChild(textnode);
                    document.getElementById( "app-language-hidden").value = language.value
                    first = false
                }

                let node = document.createElement("button");
                node.value = language.value
                node.classList.add("dropdown-menu-language-item")
                node.classList.add("dropdown-item")
                node.id = "dropdown-item-"+language.value
                node.setAttribute("languageKey", language.key)
                node.value = language.value;
                let textnode = document.createTextNode(language.key);
                node.appendChild(textnode);
                document.getElementById("dropdown-menu-language").appendChild(node);

                $(node).on('click', function (e) {
                    e.preventDefault();
                    let elem = $(this);
                    let v = elem.attr("languageKey")
                    let textnode = document.createTextNode(v);
                    let node = document.getElementById("dropdownMenuButton");
                    node.innerHTML = '';
                    document.getElementById( "app-language-hidden").value = elem.val()
                    node.appendChild(textnode);
                })
            })
        })
        .catch(function(err) {
            alert('error get metadata')
            console.log(err)
        });
}

