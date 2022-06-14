import renderAvatarHTML from "./avatar.js";
import {escapeHTML} from "../utils.js";


const template = document.createElement('template')

template.innerHTML = `
    <div class="container">
        <h1>404 Not Found</h1>
    </div>
`


export default function renderPost(post) {
    const {user} = post
    const ago = new Date(post.createdAt).toString()
    const li = document.createElement('li')
    li.className = 'post-item'

    li.innerHTML =`
    <article class="post">
        <div class="post-header">
            <a href="/users/${user.username}">
                ${renderAvatarHTML(user)}
                <span>${user.username}</span>
            </a>
            <a href="/posts/${post.id}">
                <time datetime="${post.createdAt}">${ago}</time>
            </a>
        </div>
        <div class="post-content">${escapeHTML(post.content)}</div>
        <div class="post-controls">
            <button class="like-button">${post.likesCount}</button>
            <a class="comments-link" href="/posts/${post.id}">${post.CommentsCount}</a>
        </div>
</article>`

    return li
}







