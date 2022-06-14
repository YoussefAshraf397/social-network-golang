import {getAuthUser, isAuthenticated} from "./auth.js";

const authUser = getAuthUser()
const authnticated = authUser !== null
const header = document.querySelector('header')

header.innerHTML = `
<nav>
<a href="/">Home</a>
${authnticated ?`<a href="/users/${authUser.username}">Profile</a>
<button id="logout-button">Logout</button>` : ''
}

    `


if (authnticated) {
    const logoutButton = (header.querySelector('#logout-button'))
    logoutButton.addEventListener('click' , onLogoutButtonClick)
}

function onLogoutButtonClick(ev) {
    const button = (ev.currentTarget)
    button.disabled = true
    localStorage.clear()
    location.reload()

}