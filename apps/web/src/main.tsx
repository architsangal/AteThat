import React from 'react'
import ReactDOM from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import App from './App'
import './styles.css'
import './features/phase-2.1.css'
import './features/spacing.css'
import './features/health-ui.css'
import './features/health-analysis.css'
import './scrollbars.css'

const router = createBrowserRouter([{ path: '*', element: <App /> }])
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <RouterProvider router={router} />
  </React.StrictMode>,
)
