import {createApp} from 'vue'
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
  faPlus,
  faQuestion
} from '@fortawesome/free-solid-svg-icons'
import {faGithub} from '@fortawesome/free-brands-svg-icons'
import {faCopy} from '@fortawesome/free-regular-svg-icons'
import ClipboardJS from 'clipboard'
import iziToast from 'izitoast'
import index from './views/index.vue'

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
)

createApp(index)
  .use((app): void => {
    const clipboard = new ClipboardJS('[data-clipboard]')

    clipboard // <https://clipboardjs.com/#events>
      .on('error', () => {
        iziToast.error({title: 'Copying error!'})
      })
      .on('success', (e) => {
        iziToast.success({title: 'Copied!', message: e.text, timeout: 4000})
        e.clearSelection()
      })

    app.config.globalProperties.$clipboard = clipboard
  })
  .mount('#app')
