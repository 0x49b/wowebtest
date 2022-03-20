function toHtml(json) {
    const htmlElem = JSON.parse(json);
    const target = htmlElem.target === null
        ? document.querySelector("body")
        : document.getElementById(htmlElem.target);

    let e = document.getElementById(htmlElem.id);
    if (e === null) {
        e = document.createElement(htmlElem.type.toLowerCase());
        e.id = htmlElem.id;
        target.appendChild(e);
    }
    (htmlElem.cssclass) ? e.classList.add(htmlElem.cssclass) : "";
    e.textContent = htmlElem.text;
    if (htmlElem.children !== null) {
        htmlElem.children.forEach(toHtml)
    }
}
