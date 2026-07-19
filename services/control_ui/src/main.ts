import './style.css'
import { mountControlUi } from './app'

const baseUrl = import.meta.env.VITE_CONTROL_SERVER_API_URL ?? ''

mountControlUi(document.querySelector<HTMLDivElement>('#app')!, { baseUrl })
