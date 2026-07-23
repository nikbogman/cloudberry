import './style.css'
import { mountUi } from './app'

const serverApiBaseUrl = import.meta.env.VITE_SERVER_API_URL ?? ''
const piApiBaseUrl = import.meta.env.VITE_PI_API_URL ?? ''

mountUi(document.querySelector<HTMLDivElement>('#app')!, { serverApiBaseUrl, piApiBaseUrl })
