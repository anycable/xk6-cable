// Find and return the Turbo stream name
export function turboStreamSource(doc, selector) {
  let el = doc.find(`${ selector ? selector + ' ': ''}turbo-cable-stream-source`);
  if (!el) return;

  return { streamName: el.attr("signed-stream-name"), channelName: el.attr("channel") || "Turbo::StreamsChannel" };
}

// Find and return action-cable-url on the page
export function cableUrl(doc) {
  return readMeta(doc, 'action-cable-url')
}

// Find and return csrf-token on the page
export function csrfToken(doc) {
  return readMeta(doc, 'csrf-token')
}

// Find and return csrf-param on the page
export function csrfParam(doc) {
  return readMeta(doc, 'csrf-param')
}

// Find and return meta attributes' value
export function readMeta(doc, name, attrContent = 'content') {
  let el = doc.find(`meta[name=${name.toString()}]`)
  if (!el) return;

  return el.attr(attrContent.toString())
}
