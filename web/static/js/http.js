import {isAuthenticated} from "./auth.js";
import {parseJSON, stringifyJSON} from "./lib/json.js";
import {isObject} from "./utils.js";

export function doGet(url, headers) {
  return fetch (url, {
    headers: Object.assign(defaultHeaders(), headers),
  }).then(parseResponse)

}


export function doPost(url, body , headers) {
  const init = {
    method: 'POST' ,
    headers: defaultHeaders(),
  }
  if( isObject(body)) {
    init['body'] = stringifyJSON(body)
    init.headers['content-type'] = "application/json; charset=utf-8"
  }
  Object.assign(init.headers , headers)
  return fetch(url, init).then(parseResponse)

}

function defaultHeaders() {
  return isAuthenticated()
      ? {authorization: "Bearer " + localStorage.getItem('token')}
      : {}
}


async function parseResponse(res) {
  const body = parseJSON(await res.text())
  console.log("ssddsdsdsds")
  return body
  if(!res.ok) {
    console.log("ssddsdsdsds")
    const msg = String(body)
    const err = new Error(msg)
    console.log(err)
    err.name = msg.toLocaleLowerCase().split(' ').map(word => {
      return word.charAt(0).toUpperCase() + word.slice(1)
    }).join('')+ 'Error'
    err['statusCode'] = err.status
    err['statusText'] = err.statusText
    err['url'] = err.url
    console.log(err)
    throw err
  }
}