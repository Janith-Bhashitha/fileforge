import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ThemeProvider } from './lib/theme'
import { DeveloperModeProvider } from './lib/developerMode'
import { AuthProvider } from './lib/auth'
import { ProtectedRoute } from './components/ProtectedRoute'
import { AppShell } from './components/AppShell'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { DashboardPage } from './pages/DashboardPage'
import { ConvertPage } from './pages/ConvertPage'
import { BatchProcessingPage } from './pages/BatchProcessingPage'
import { FilesPage } from './pages/FilesPage'
import { HistoryPage } from './pages/HistoryPage'
import { AIProcessingPage } from './pages/AIProcessingPage'
import { OCRPage } from './pages/OCRPage'
import { DocumentInsightsPage } from './pages/DocumentInsightsPage'
import { DeveloperPage } from './pages/DeveloperPage'
import { DeveloperOverviewPage } from './pages/developer/DeveloperOverviewPage'
import { DeveloperApiKeysPage } from './pages/developer/DeveloperApiKeysPage'
import { DeveloperWebhooksPage } from './pages/developer/DeveloperWebhooksPage'
import { DeveloperUsagePage } from './pages/developer/DeveloperUsagePage'
import { DeveloperCliPage } from './pages/developer/DeveloperCliPage'
import { SettingsPage } from './pages/SettingsPage'
import { HelpPage } from './pages/HelpPage'
import { SystemStatusPage } from './pages/SystemStatusPage'

const queryClient = new QueryClient()

function App() {
  return (
    <ThemeProvider>
      <DeveloperModeProvider>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <BrowserRouter>
            <Routes>
              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />

              <Route element={<ProtectedRoute />}>
                <Route element={<AppShell />}>
                  <Route path="/" element={<DashboardPage />} />
                  <Route path="/convert" element={<ConvertPage />} />
                  <Route path="/batches" element={<BatchProcessingPage />} />
                  <Route path="/files" element={<FilesPage />} />
                  <Route path="/history" element={<HistoryPage />} />

                  <Route path="/ai" element={<AIProcessingPage />} />
                  <Route path="/ocr" element={<OCRPage />} />
                  <Route path="/insights" element={<DocumentInsightsPage />} />

                  <Route path="/developer" element={<DeveloperPage />}>
                    <Route index element={<Navigate to="overview" replace />} />
                    <Route path="overview" element={<DeveloperOverviewPage />} />
                    <Route path="api-keys" element={<DeveloperApiKeysPage />} />
                    <Route path="webhooks" element={<DeveloperWebhooksPage />} />
                    <Route path="usage" element={<DeveloperUsagePage />} />
                    <Route path="cli" element={<DeveloperCliPage />} />
                  </Route>

                  <Route path="/settings" element={<SettingsPage />} />
                  <Route path="/help" element={<HelpPage />} />
                  <Route path="/status" element={<SystemStatusPage />} />
                </Route>
              </Route>

              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </BrowserRouter>
        </AuthProvider>
      </QueryClientProvider>
      </DeveloperModeProvider>
    </ThemeProvider>
  )
}

export default App
