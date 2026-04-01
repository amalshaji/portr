import { BrowserRouter, Route, Routes } from "react-router-dom"

import { Toaster } from "@/components/ui/sonner"
import { TooltipProvider } from "@/components/ui/tooltip"
import { HomePage } from "@/pages/home-page"
import { TunnelPage } from "@/pages/tunnel-page"

export function App() {
  return (
    <BrowserRouter>
      <TooltipProvider>
        <Toaster position="top-right" richColors />
        <Routes>
          <Route element={<HomePage />} path="/" />
          <Route element={<TunnelPage />} path="/:id" />
        </Routes>
      </TooltipProvider>
    </BrowserRouter>
  )
}

export default App
