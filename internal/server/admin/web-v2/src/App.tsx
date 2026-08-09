import { useEffect } from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Toaster } from 'sonner'
import { startThemeSync, useThemeStore } from '@/lib/theme'
import HomePage from './pages/home/HomePage'
import AppPage from './pages/app/AppPage'
import NotFound from './pages/NotFound'

function App() {
  const resolved = useThemeStore((state) => state.resolved)

  useEffect(() => startThemeSync(), [])

  return (
    <BrowserRouter>
      <Toaster position="top-right" theme={resolved} />
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/:team/*" element={<AppPage />} />
        <Route path="*" element={<NotFound />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
