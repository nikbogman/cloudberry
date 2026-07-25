import './style.css'
import { mountUi } from './app'

const computeApiBaseUrl = import.meta.env.VITE_COMPUTE_API_URL ?? ''
const gatewayApiBaseUrl = import.meta.env.VITE_GATEWAY_API_URL ?? ''

mountUi(document.querySelector<HTMLDivElement>('#app')!, { computeApiBaseUrl, gatewayApiBaseUrl })
