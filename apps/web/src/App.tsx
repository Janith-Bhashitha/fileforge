import { lazy, Suspense } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ThemeProvider } from './lib/theme'
import { DeveloperModeProvider } from './lib/developerMode'
import { AuthProvider } from './lib/auth'
import { ToastProvider } from './components/Toast'
import { ProtectedRoute } from './components/ProtectedRoute'
import { AppShell } from './components/AppShell'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { ForgotPasswordPage } from './pages/ForgotPasswordPage'
import { ResetPasswordPage } from './pages/ResetPasswordPage'
import { LandingPage } from './pages/LandingPage'
import { DashboardPage } from './pages/DashboardPage'
import { ConvertPage } from './pages/ConvertPage'
import { BatchProcessingPage } from './pages/BatchProcessingPage'
import { FilesPage } from './pages/FilesPage'
import { HistoryPage } from './pages/HistoryPage'
import { OCRPage } from './pages/OCRPage'
import { DocumentInsightsPage } from './pages/DocumentInsightsPage'
import { AIProcessingPage } from './pages/AIProcessingPage'
import { DeveloperPage } from './pages/DeveloperPage'
import { DeveloperOverviewPage } from './pages/developer/DeveloperOverviewPage'
import { DeveloperApiKeysPage } from './pages/developer/DeveloperApiKeysPage'
import { DeveloperWebhooksPage } from './pages/developer/DeveloperWebhooksPage'
import { DeveloperUsagePage } from './pages/developer/DeveloperUsagePage'
import { DeveloperCliPage } from './pages/developer/DeveloperCliPage'
import { SettingsPage } from './pages/SettingsPage'
import { SystemStatusPage } from './pages/SystemStatusPage'

const queryClient = new QueryClient()

// Edit & Sign is the only page that needs pdf.js, and pdf.js is by far the
// heaviest dependency in the app. Loading it with the route rather than with
// the bundle keeps it off the critical path for everyone who never opens it,
// which on a free-tier box serving over the public internet is most visits.
const EditSignPage = lazy(() =>
  import('./pages/EditSignPage').then((m) => ({ default: m.EditSignPage }))
)

function App() {
  return (
    <ThemeProvider>
      <DeveloperModeProvider>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
        <AuthProvider>
          <BrowserRouter>
            <Routes>
              <Route path="/" element={<LandingPage />} />
              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />
              <Route path="/forgot-password" element={<ForgotPasswordPage />} />
              <Route path="/reset-password" element={<ResetPasswordPage />} />

              <Route element={<ProtectedRoute />}>
                <Route element={<AppShell />}>
                  <Route path="/dashboard" element={<DashboardPage />} />
                  <Route path="/convert" element={<ConvertPage />} />
                  <Route path="/batches" element={<BatchProcessingPage />} />
                  <Route
                    path="/edit-sign"
                    element={
                      <Suspense fallback={<p className="empty-hint">Loading editor…</p>}>
                        <EditSignPage />
                      </Suspense>
                    }
                  />
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
                  <Route path="/status" element={<SystemStatusPage />} />
                </Route>
              </Route>

              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </BrowserRouter>
        </AuthProvider>
        </ToastProvider>
      </QueryClientProvider>
      </DeveloperModeProvider>
    </ThemeProvider>
  )
}

export default App
