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
    let response

    input.disabled = true
    button.disabled = true

    try{
        const out = await http.login(email)

        localStorage.setItem('token' , out.token)
        localStorage.setItem('expires_at' , typeof out.expiresAt === 'string'
            ? out.expiresAt
            : out.expiresAt.toJSON())
        localStorage.setItem('auth_user', stringifyJSON(out.authUser))
        location.reload()
    } catch (err) {
        console.log(err)
      console.error(err)
        if (err.name === 'UserNotFoundError')
        {
            if(confirm('user not found. Do yo want create an account')) {
                runRegistrationProgram(email)
            }
            return
        }
        alert(err.message)
        setTimeout(() => {input.focus()})
    } finally {
        input.disabled = false
        button.disabled = false
    }


}

async function runRegistrationProgram(email , username){
    username = prompt('username: ', username)
    if (username === null){
        return
    }
    username = username.trim()
    if(!rxUsername.test(username)) {
        alert('invalid username')
        runRegistrationProgram(email, username)
        return
    }
    try{
        await http.createUser(email,username)
        saveLogin(await http.login(email))
        location.reload()

    }catch (err){
        console.error(err)
        alert(err.message)
        if(err.name === 'usernameTakenError') {
            runRegistrationProgram
        }
    }
}


const http = {
    login: email =>doPost('api/login' , {email}) ,
    createUser: (email,username) => doPost('/api/users' , {email,username})
}