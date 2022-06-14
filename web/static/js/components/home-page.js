const template = document.createElement('template')
template.innerHTML = `
 <div class="container">
    <h1>Home Page</h1>
</div>
`


export default function renderHomePage() {
    const page = (template.content.cloneNode(true))
    return page
}