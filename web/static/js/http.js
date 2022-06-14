import {isAuthenticated} from "./auth.js";
import {parseJSON, stringifyJSON} from "./lib/json.js";
import {isObject} from "./utils.js";

export function doGet(url, headers) {
  return fetch (url, {
    headers: object.assign(defaultHeaders(), headers),
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

  // return fetch('http://localhost:3000/api/login', {
  //   method: 'POST',
  //   headers: {
  //     'content-type': 'application/json',
  //   },
  //   body: {
  //     email: 'Royoussef@youssef.com'
  //   }
  // }).then(response => {
  //       console.log(response)
  //     })
  //     .catch(err => {
  //       console.log(err)
  //     })

      // const urls = "http://localhost:3000/api/login"
      // let postObj = {
      //   email: "youssef@youssef.com"
      // }
     // let post = JSON.stringify(postObj)

  // let xhr = new XMLHttpRequest()
  // xhr.open('POST', url, true)
  // // console.log(init)
  // xhr.setRequestHeader('Content-type', 'application/json; charset=UTF-8')
  // xhr.send(init['body']);
  //
  // let test;
  // xhr.onload = function () {
  //   test =   xhr.response
  //   body = parseJSON( test)
  //   if(xhr.status === 200) {
  //     console.log( body.token)
  //
  //     console.log("Post successfully created!")
  //     return body
  //   }
  // }
  //
  // return xhr.responseText
  return fetch(url, init).then(parseResponse => {
    console.log("sdscds" , parseResponse.token)
  })
  // return fetch(url, {
  //   Method: 'POST',
  //   Headers: {'Content-type': 'application/json; charset=UTF-8'} ,
  //   Body:  init['body']
  // }).then(parseResponse)
  // return fetch(url, init).then(parseResponse)

}

function defaultHeaders() {
  return isAuthenticated()
      ? {authorization: "Bearer " + localStorage.getItem('token')}
      : {}
}


async function parseResponse(res) {
  const body = parseJSON(await res.text())
  if(!res.ok) {
    const msg = String(body)
    const err = new Error(msg)
    err.name = msg.toLocaleLowerCase().split(' ').map(word => {
      return word.charAt(0).toUpperCase() + word.slice(1)
    }).join('')+ 'Error'
    err['statusCode'] = err.status
    err['statusText'] = err.statusText
    err['url'] = err.url
    throw err
  }
}