function toHtml(json) {
    json.ElementType = undefined;
    json.class = undefined;
    json.targetToInsertInto = undefined;

    const newElement = document.createElement(json.ElementType);
    newElement.className = json.class;
    newElement.id = json.id;
    appendTo(json.targetToInsertInto)(newElement);
}

let appendTo = appendToId => newElement =>
    appendToId === undefined || appendToId === null
    ? document.body.appendChild(newElement) : document.getElementById(appendToId).appendChild(newElement);
