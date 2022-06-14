const template = document.createElement('template')
template.innerHTML = `
 <div class="container">
    <h1>Home Page</h1>
    <a href="\login">Login</a>
</div>
`


export default function renderHomePage() {
    const page = (template.content.cloneNode(true))
    return page
}