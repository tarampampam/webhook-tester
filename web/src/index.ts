import {createApp} from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import {library} from '@fortawesome/fontawesome-svg-core'
import {
  faAngleLeft,
  faAngleRight,
  faAnglesLeft,
  faAnglesRight,
  faArrowUpRightFromSquare,
  faAtom,
  faDownload,
  faFont, faPersonRunning,
  faPlus,
  faQuestion
} from '@fortawesome/free-solid-svg-icons'
import {faGithub} from '@fortawesome/free-brands-svg-icons'
import {faCopy} from '@fortawesome/free-regular-svg-icons'
import ClipboardJS from 'clipboard'
import iziToast from 'izitoast'
import index from './views/index.vue'
import hljs from 'highlight.js/lib/core'
import javascript from 'highlight.js/lib/languages/javascript'
import hljsVuePlugin from '@highlightjs/vue-plugin'

library.add( // https://fontawesome.com/icons
  faQuestion,
  faGithub,
  faCopy,
  faPlus,
  faArrowUpRightFromSquare,
  faAngleLeft,
  faAngleRight,
  faAnglesLeft,
  faAnglesRight,
  faFont,
  faAtom,
  faDownload,
  faPersonRunning,
)

hljs.registerLanguage('javascript', javascript)

new ClipboardJS('[data-clipboard]') // <https://clipboardjs.com/#events>
  .on('error', () => {
    iziToast.error({title: 'Copying error!'})
  })
  .on('success', (e) => {
    iziToast.success({title: 'Copied!', message: e.text, timeout: 4000})

    e.clearSelection()
  })

const router = createRouter({
  history: createWebHashHistory(), // https://router.vuejs.org/guide/essentials/history-mode.html#hash-mode
  routes: [
    {path: '/', name: 'index', component: index},
    {path: '/:sessionUUID?/:requestUUID?', name: 'request', props: true, component: index},
  ],
})

createApp(index)
  .use(router)
  .use(hljsVuePlugin)
  .mount('#app')
