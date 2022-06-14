import {createRouter} from './lib/router.js'
import {gurd} from "./auth.js";

const r = createRouter()
// r.route('/' , view('home'))
r.route('/' , gurd(view('home') , view('access')) )

r.route(/^\//, view('not-found'))

console.log(r)

r.subscribe(renderInto(document.querySelector('main')))
r.install()

function view(name) {
    return (...args) => import(`/js/components/${name}-page.js`)
        .then(m => m.default(...args))
}


function renderInto(target) {
    return async result => {
        target.innerHTML = ''
        target.appendChild(await result)
    }
}