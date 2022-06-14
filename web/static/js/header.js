// import {createRouter} from './lib/router.js'
//
// const main = document.querySelector('main')
// const r = createRouter()
// r.route('/' , view('home'))
// r.route(/^\//, view('not-found'))
//
// r.subscribe(renderInto(document.querySelector('main')))
// r.install()
//
// function view(name) {
//     return (...args) => import(`/js/components/${name}-page.js`)
//         .then(m => m.default(...args))
// }
//
//
// function renderInto(target) {
//     return async result => {
//         target.innerHTML = ''
//         target.appendChild(await result)
//     }
// }


// const main = document.querySelector('main')
// const r = createRouter()
//
// r.route('/aa' , () => {return 'Home Page'})
// r.subscribe(renderInto(main))
// r.install()
// // r.install(renderInto(main))
//
// function renderInto(target) {
//     return  result => {
//         target.innerHTML = result
//     }
// }

// import {createRouter} from './lib/router.js'
//
// const main = document.querySelector('main')
// const router = createRouter()
//
// router.route('/', homePage)
// router.route(/^\/users\/(?<username>[^/]+)$/, userPage)
// router.route(/^\//, notFound)
// router.subscribe(render)
// router.install()
//
// function homePage() {
//     return 'Home Page'
// }

// function userPage(params) {
//     return `${params.username}'s Profile Page`
// }
//
// function notFound() {
//     return '404 not found'
// }
//
// function notFoundPage() {
//     return '404 Not Found'
// }
//
// function render(result) {
//     main.innerHTML = result
// }