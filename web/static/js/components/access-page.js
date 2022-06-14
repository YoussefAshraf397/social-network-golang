import {stringifyJSON} from "../lib/json.js";
import {doPost} from "../http.js";


const template = document.createElement('template')
template.innerHTML = `
 <div class="container">
    <h1>Access</h1>
    <form id="login-form">
    <input type="text" placeholder="Email" autocomplete="email" value="youssef@youssef.com" required></input>
    <button>Login</button>
</form>
</div>
`


export default function renderAccessPage() {
    const page = (template.content.cloneNode(true))
    const loginForm = (page.getElementById('login-form'))
    loginForm.addEventListener('submit' , onLoginFormSubmit)
    return page
}

async function onLoginFormSubmit(ev) {
    ev.preventDefault()
    const form = (ev.currentTarget)
    const input = form.querySelector('input')
    const button = form.querySelector('button')
    const email = input.value
    console.log(email)

    input.disabled = true
    button.disabled = true

    try{
        const out = await  http.login(email)

        // console.log("in access-page: " , http.login(email))

        localStorage.setItem('token' , out.token)
        localStorage.setItem('expires_at' , typeof out.expiresAt === 'string'
            ? out.expiresAt
            : out.expiresAt.toJSON())
        localStorage.setItem('auth_user', stringifyJSON(out.user))
        location.reload()
    } catch (err) {
      console.error(err)
        alert(err.message)
        setTimeout(input.focus)
    } finally {
        input.disabled = false
        button.disabled = false
    }


}

const http = {
    login: email =>doPost('api/login' , {email})
}