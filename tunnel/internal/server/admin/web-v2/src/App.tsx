import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Toaster } from 'sonner'
import HomePage from './pages/home/HomePage'
import AppPage from './pages/app/AppPage'
import InstanceSettings from './pages/instance-settings/InstanceSettings'
import NotFound from './pages/NotFound'

function App() {
  return (
    <BrowserRouter>
      <Toaster position="top-right" />
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/instance-settings/*" element={<InstanceSettings />} />
        <Route path="/:team/*" element={<AppPage />} />
        <Route path="*" element={<NotFound />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
