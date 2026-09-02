import type { LangCode } from './lang-codes'

/**
 * Translation message keys - each maps to a localized string in every supported locale.
 */
export const KEY = {
  __testKey: '__testKey', // for testing purposes
  justNow: 'justNow',
  notFoundMessage: 'notFoundMessage',
  goToHome: 'goToHome',
  reportIssueToDevelopers: 'reportIssueToDevelopers',
  moreDetailsInConsole: 'moreDetailsInConsole',
  appUpdateAvailable: 'appUpdateAvailable',
  selectLanguage: 'selectLanguage',
  delete: 'delete',
  somethingWentWrong: 'somethingWentWrong',
  firstRequest: 'firstRequest',
  newerRequest: 'newerRequest', // 5 symbols max (markup constraint)
  olderRequest: 'olderRequest', // 5 symbols max (markup constraint)
  lastRequest: 'lastRequest',
  deleteAllRequests: 'deleteAllRequests',
  noRequestsCapturedYet: 'noRequestsCapturedYet',
  sendFirstRequestToSee: 'sendFirstRequestToSee',
  pleaseWait: 'pleaseWait',
  loadingSessions: 'loadingSessions',
  switchingSession: 'switchingSession',
  creatingSession: 'creatingSession',
  copy: 'copy',
  copied: 'copied',
  sendTestRequest: 'sendTestRequest',
  webhookUrl: 'webhookUrl',
  withSpecificStatusCode: 'withSpecificStatusCode',
  openInNewTab: 'openInNewTab',
  localhost: 'localhost',
  expandCode: 'expandCode',
  collapseCode: 'collapseCode',
  yourUniqueWebhookUrl: 'yourUniqueWebhookUrl',
  sendSimpleRequestInShell: 'sendSimpleRequestInShell',
  codeSnippets: 'codeSnippets',
  webhookOptions: 'webhookOptions',
  statusCode: 'statusCode',
  responseHeaders: 'responseHeaders',
  responseDelay: 'responseDelay',
  responseBody: 'responseBody',
  plusMore: 'plusMore',
  timeSec: 'timeSec',
  noDelay: 'noDelay',
  noHeaders: 'noHeaders',
  webhookTesterUsedFor: 'webhookTesterUsedFor',
  noNeedToRefresh: 'noNeedToRefresh',
  webhookWithCustomStatus: 'webhookWithCustomStatus',
  sessionWillExpireAfter: 'sessionWillExpireAfter',
  maxRequestBodySizeLimit: 'maxRequestBodySizeLimit',
  maxRequestsLimit: 'maxRequestsLimit',
} as const

/**
 * Type of translation message keys.
 */
export type KEY = (typeof KEY)[keyof typeof KEY]

const __TEST_KEY_VALUE =
  '{text}, {number}, {repeated}, {repeated}, {nullable}, {undefined}, {missing}, { spaces }, {123}'

/**
 * All supported locales and their translations.
 *
 * List of the top 10 most spoken languages:
 *
 * | Language         | BCP 47 Tag | % of World Population (~8.2B) |
 * |------------------|------------|-------------------------------|
 * | English          | `en`       | ~18.6%                        |
 * | Mandarin Chinese | `zh`       | ~14.4%                        |
 * | Hindi            | `hi`       | ~7.4%                         |
 * | Spanish          | `es`       | ~6.7%                         |
 * | Arabic           | `ar`       | ~4.9%                         |
 * | French           | `fr`       | ~3.8%                         |
 * | Bengali          | `bn`       | ~3.5%                         |
 * | Portuguese       | `pt`       | ~3.3%                         |
 * | Russian          | `ru`       | ~3.1%                         |
 * | Indonesian       | `id`       | ~3.1%                         |
 */
const locales = {
  // TODO: add more languages
  en: {
    [KEY.__testKey]: `Test ${__TEST_KEY_VALUE}`, // for testing purposes
    [KEY.justNow]: 'Just now',
    [KEY.notFoundMessage]: 'The requested page was not found',
    [KEY.goToHome]: 'Go to Home',
    [KEY.reportIssueToDevelopers]: 'Please, report this issue to the developers',
    [KEY.moreDetailsInConsole]: 'You can find more details in the console log',
    [KEY.appUpdateAvailable]: 'An update is available',
    [KEY.selectLanguage]: 'Select language',
    [KEY.delete]: 'Delete',
    [KEY.somethingWentWrong]: 'Something went wrong',
    [KEY.firstRequest]: 'Newest request',
    [KEY.newerRequest]: 'Newer', // 5 symbols max (markup constraint)
    [KEY.olderRequest]: 'Older', // 5 symbols max (markup constraint)
    [KEY.lastRequest]: 'Oldest request',
    [KEY.deleteAllRequests]: 'Delete all requests',
    [KEY.noRequestsCapturedYet]: 'No requests have been captured yet',
    [KEY.sendFirstRequestToSee]: 'Send your first request to see it here',
    [KEY.pleaseWait]: 'Please wait',
    [KEY.loadingSessions]: 'Loading sessions',
    [KEY.switchingSession]: 'Switching session',
    [KEY.creatingSession]: 'Creating session',
    [KEY.copy]: 'Copy',
    [KEY.copied]: 'Copied',
    [KEY.sendTestRequest]: 'Send test request',
    [KEY.webhookUrl]: 'Webhook URL',
    [KEY.withSpecificStatusCode]: 'With specific status code',
    [KEY.openInNewTab]: 'Open in a new tab',
    [KEY.localhost]: 'localhost',
    [KEY.expandCode]: 'Expand code',
    [KEY.collapseCode]: 'Collapse code',
    [KEY.yourUniqueWebhookUrl]: "Here's your unique webhook URL",
    [KEY.sendSimpleRequestInShell]:
      'Send simple request (execute next command in your terminal without leaving this page)',
    [KEY.codeSnippets]: 'Code snippets in various programming languages',
    [KEY.webhookOptions]: 'Webhook options',
    [KEY.statusCode]: 'Status code',
    [KEY.responseHeaders]: 'Response headers',
    [KEY.responseDelay]: 'Response delay',
    [KEY.responseBody]: 'Response body',
    [KEY.plusMore]: 'more', // e.g. `+5 more`
    [KEY.timeSec]: 'sec', // short for "seconds"
    [KEY.noDelay]: 'none',
    [KEY.noHeaders]: 'none',
    [KEY.webhookTesterUsedFor]: 'Webhook Tester lets you easily test webhooks and other HTTP requests',
    [KEY.noNeedToRefresh]: 'Any requests sent to this URL are instantly logged here - no need to refresh!',
    [KEY.webhookWithCustomStatus]:
      'To specify a status code in the response, append it to the URL, like {url} - this way, the URL will respond with a {code} status',
    [KEY.sessionWillExpireAfter]: 'Requests and tokens for this URL expire after {duration} of inactivity',
    [KEY.maxRequestBodySizeLimit]: 'The maximum size for incoming requests is {size}',
    [KEY.maxRequestsLimit]: 'The maximum number of requests per session is {limit}',
  },
  ru: {
    [KEY.__testKey]: `Тест ${__TEST_KEY_VALUE}`, // for testing purposes
    [KEY.justNow]: 'Только что',
    [KEY.notFoundMessage]: 'Страница не найдена',
    [KEY.goToHome]: 'На главную',
    [KEY.reportIssueToDevelopers]: 'Пожалуйста, сообщите об этой проблеме разработчикам',
    [KEY.moreDetailsInConsole]: 'Больше деталей вы сможете найти в консоли',
    [KEY.appUpdateAvailable]: 'Доступно обновление',
    [KEY.selectLanguage]: 'Выберите язык',
    [KEY.delete]: 'Удалить',
    [KEY.somethingWentWrong]: 'Что-то пошло явно не так',
    [KEY.firstRequest]: 'Самый новый запрос',
    [KEY.newerRequest]: 'Сюда',
    [KEY.olderRequest]: 'Туда',
    [KEY.lastRequest]: 'Самый старый запрос',
    [KEY.deleteAllRequests]: 'Удалить все запросы',
    [KEY.noRequestsCapturedYet]: 'Пока что нет пойманных запросов',
    [KEY.sendFirstRequestToSee]: 'Отправьте ваш первый запрос, чтобы увидеть его здесь',
    [KEY.pleaseWait]: 'Пожалуйста, подождите',
    [KEY.loadingSessions]: 'Загрузка сессий',
    [KEY.switchingSession]: 'Переключение сессии',
    [KEY.creatingSession]: 'Создание сессии',
    [KEY.copy]: 'Копировать',
    [KEY.copied]: 'Скопировано',
    [KEY.sendTestRequest]: 'Отправить тестовый запрос',
    [KEY.webhookUrl]: 'URL вебхука',
    [KEY.withSpecificStatusCode]: 'С конкретным кодом ответа',
    [KEY.openInNewTab]: 'Открыть в новой вкладке',
    [KEY.localhost]: 'localhost', // better to keep it in English as it's a technical term
    [KEY.expandCode]: 'Развернуть код',
    [KEY.collapseCode]: 'Свернуть код',
    [KEY.yourUniqueWebhookUrl]: 'Вот ваш уникальный URL вебхука',
    [KEY.sendSimpleRequestInShell]:
      'Отправьте простой запрос (выполните следующую команду в вашем терминале, не покидая эту страницу)',
    [KEY.codeSnippets]: 'Примеры кода на различных языках программирования',
    [KEY.webhookOptions]: 'Настройки вебхука',
    [KEY.statusCode]: 'Код ответа',
    [KEY.responseHeaders]: 'Заголовки ответа',
    [KEY.responseDelay]: 'Задержка ответа',
    [KEY.responseBody]: 'Тело ответа',
    [KEY.plusMore]: 'ещё',
    [KEY.timeSec]: 'сек',
    [KEY.noDelay]: 'нет',
    [KEY.noHeaders]: 'нет',
    [KEY.webhookTesterUsedFor]:
      'При помощи Webhook Tester вы можете легко и удобно тестировать вебхуки и любые другие HTTP-запросы',
    [KEY.noNeedToRefresh]:
      'Любые запросы, отправленные на этот URL - мгновенно отобразятся здесь, даже без обновления страницы!',
    [KEY.webhookWithCustomStatus]:
      'Чтобы указать код ответа в ответе, добавьте его к URL, например {url} - таким образом, URL будет возвращать {code} код ответа',
    [KEY.sessionWillExpireAfter]: 'Все сохраненные запросы, как и сам URL будут удалены через {duration} бездействия',
    [KEY.maxRequestBodySizeLimit]: 'Максимальный размер входящих запросов составляет {size}',
    [KEY.maxRequestsLimit]: 'Хранимое количество запросов в одной сессии ограничено {limit}',
  },
} satisfies Partial<Readonly<Record<LangCode, Readonly<Record<KEY, string>>>>>

/**
 * Language codes that have registered translations.
 */
export type SupportedLangCode = keyof typeof locales & LangCode

/**
 * All supported language codes.
 */
export const SUPPORTED_LANG_CODES = Object.keys(locales) as ReadonlyArray<SupportedLangCode>

/**
 * Returns true if v is a recognized supported language code.
 */
export const isSupportedLangCode = (v: string): v is SupportedLangCode => v in locales

/**
 * Returns the localized string for key in the given language, replacing named placeholders with own values.
 *
 * Missing placeholder values are preserved. Provided values are converted using String().
 *
 * @example
 * ```ts
 * // [KEY.foo]: 'URL: {url}', code: {code}'
 *
 * translate(KEY.foo, 'en', { url: 'https://example.test/200', code: 200 }) // 'URL: https://example.test/200, code: 200'
 * ```
 */
export const translate = (key: KEY, lang: SupportedLangCode, values?: Readonly<Record<string, unknown>>): string => {
  if (!values) {
    return locales[lang][key] // fast path for no placeholder values
  }

  return locales[lang][key].replace(/\{([A-Za-z][A-Za-z0-9_]*)}/g, (placeholder, name: string) => {
    if (Object.prototype.hasOwnProperty.call(values, name)) {
      return String(values[name])
    }

    return placeholder
  })
}
