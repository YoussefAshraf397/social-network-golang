import {doGet, doPost} from "../http.js";
import renderPost from "./post.js";
import {getAuthUser} from "../auth.js";

const PAGE_SIZE = 3
const template = document.createElement('template')
template.innerHTML = `
 <div class="container">
    <h1>Timeline</h1>
    <form class="post-form" id="post-form" >
        <textarea placeholder="Write something..." required maxlength="480"></textarea>
        <button class="post-form-button">Publish</button>
    </form>
    <ol class="post-list" id="timeline-list"></ol>
     <button id="load-more-button">Load More</button>
    
    

   
</div>
`


export default async function renderHomePage() {
    const timeline = await http.timeline()
    console.log(timeline)

    const page = (template.content.cloneNode(true))
    const postForm = (page.getElementById('post-form'))
    const postFormTextArea = postForm.querySelector('textarea')
    const postFormButton = postForm.querySelector('button')

    const timelineList = (page.getElementById('timeline-list'))
    const loadMoreButton = (page.getElementById('load-more-button'))

    const onPostFormSubmit = async ev => {
        ev.preventDefault()
        const content = postFormTextArea.value
        console.log("post content: " , content)
        postFormTextArea.disabled = true
        postFormButton.disabled = true
        try{
            const timelineItem = await http.publishPost({content})
            timeline.unshift(timelineItem)
            timelineList.insertAdjacentElement('afterbegin' , renderPost(timelineItem.post))
            postForm.reset()
        } catch (err){
            console.log(err)
            console.error(err)
            alert(err.message)
            setTimeout(()=> {
                postFormTextArea.focus()
            })

        }finally {
            postFormTextArea.disabled = false
            postFormButton.disabled = false

        }

    }

    const onLoadMoreButtonClick = async ()=>{
        const lastTimelineItem = timeline[timeline.length - 1]
        console.log(lastTimelineItem)
        const newTimelineItems = await http.timeline(lastTimelineItem.id)
        timeline.push(...newTimelineItems)
        for (const timelineItem of timeline) {
            console.log("post is: " , timelineItem.post)
            timelineList.appendChild(renderPost(timelineItem.post))
        }

    }

    for (const timelineItem of timeline) {
        timelineList.appendChild(renderPost(timelineItem.post))
    }

    postForm.addEventListener('submit' ,onPostFormSubmit )
    loadMoreButton.addEventListener('click' ,onLoadMoreButtonClick )

    return page


}

const http = {
    publishPost: input => doPost('/api/posts' , input).then(timelineItem => {
        timelineItem.post.user = getAuthUser()
        return timelineItem
    }),
    timeline: (before = 0n) => doGet(`/api/timeline?&last=${PAGE_SIZE}&before=${before}`) ,
}