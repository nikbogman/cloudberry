import './style.css'
import { mountControlUi } from './app'

const serverApiBaseUrl = import.meta.env.VITE_CONTROL_SERVER_API_URL ?? ''
const piApiBaseUrl = import.meta.env.VITE_CONTROL_PI_API_URL ?? ''

mountControlUi(document.querySelector<HTMLDivElement>('#app')!, { serverApiBaseUrl, piApiBaseUrl })
