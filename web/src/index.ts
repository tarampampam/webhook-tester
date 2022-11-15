import {createApp} from 'vue'
import {createRouter, createWebHashHistory} from 'vue-router'
import {library} from '@fortawesome/fontawesome-svg-core'
import {
  faAngleLeft,
  faAngleRight,
  faAnglesLeft,
  faAnglesRight,
  faArrowUpRightFromSquare,
  faAtom,
  faDownload,
  faFont,
  faPersonRunning,
  faPlus,
  faQuestion
} from '@fortawesome/free-solid-svg-icons'
import {
  faGithub,
  faGolang,
  faJava,
  faJs,
  faNodeJs,
  faPhp,
  faPython
} from '@fortawesome/free-brands-svg-icons'
import {faCopy} from '@fortawesome/free-regular-svg-icons'
import ClipboardJS from 'clipboard'
import iziToast from 'izitoast'
import mainApp from './views/main-app.vue'
import hljs from 'highlight.js/lib/core'
import hlJavascript from 'highlight.js/lib/languages/javascript'
import hlGo from 'highlight.js/lib/languages/go'
import hlPython from 'highlight.js/lib/languages/python'
import hlJava from 'highlight.js/lib/languages/java'
import hlPhp from 'highlight.js/lib/languages/php'
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
  faPhp,
  faGolang,
  faPython,
  faJava,
  faNodeJs,
  faJs,
)

hljs.registerLanguage('javascript', hlJavascript)
hljs.registerLanguage('go', hlGo)
hljs.registerLanguage('python', hlPython)
hljs.registerLanguage('java', hlJava)
hljs.registerLanguage('php', hlPhp)

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
    {path: '/', name: 'index', component: {}},
    {path: '/:sessionUUID?/:requestUUID?', name: 'request', component: {}},
  ],
})

createApp(mainApp)
  .use(router)
  .use(hljsVuePlugin)
  .use(() => {
    document.getElementById('main-loader')?.remove() // hide main loading spinner
  })
  .mount('#app')
