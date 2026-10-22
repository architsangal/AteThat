import { AppDatePicker, DatePickerProvider } from './features/AppDatePicker'
import { ProfileAttention, ProfileFieldHint } from './features/ProfileAttention'
import { missingProfileDetails, profileAttentionHref, focusProfileField, type MissingProfileDetail } from './features/profile-completion'
import { TargetSettings, TargetCalculations } from './features/TargetSettings'
import { TextSummary } from './features/TextSummary'
import { Welcome } from './features/Welcome'
import { useDialogs } from './features/ui'
import { healthStatusLabel } from './features/health-model'
import { Form } from './features/ui'
import { createContext, useContext, useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { Link, NavLink, Route, Routes, useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import {
  Activity, ArrowLeft, ArrowRight, CalendarDays, Check,
  CircleHelp, Clock3, HeartPulse, Home, Info, LockKeyhole, Menu,
  Moon, Plus, Search, Settings2, ShieldCheck, Sun, UtensilsCrossed, X,
} from 'lucide-react'
import { apiBaseUrl, deleteAuthenticatedJson, getAuthenticatedJson, getJson, patchAuthenticatedJson, postAuthenticatedIdempotentJson, postAuthenticatedJson, postJson, putAuthenticatedJson, refreshAccessToken } from './api'
import { clearAccessToken, getAccessToken, setAccessToken } from './session'
import { ThemeProvider, useTheme } from './theme'
import { NutritionAnalysisPage } from './features/nutrition/NutritionAnalysisPage'
import { display, label as foodLabel, nutrientFields, type Page } from './features/nutrition/model'
import { NutrientInputs, NutrientOverrideInputs } from './features/nutrition/NutrientInputs'
import { RecipesPage } from './features/recipes/RecipesPage'
import { WellnessPage, WellnessOverview } from './features/wellness/WellnessPage'
import { ScheduleEditor, type SchedulableItem } from './features/wellness/ScheduleEditor'
import { HealthAdherence } from './features/wellness/HealthAdherence'
import { HealthOccurrences } from './features/HealthOccurrences'
import { calendarDay, type HealthOccurrence } from './features/health-occurrences'
import { AppTimeInput } from './features/AppTimeInput'
import { DialogProvider, Loading, Modal, MonthDays, NutritionSummary, ScheduleTimes } from './features/ui'
import { rankFoods, type SearchUsage } from './features/food-search'
import './features/phase-two.css'

type Icon = typeof Home

const navigation: { label: string; to: string; icon: Icon }[] = [
  { label: 'Dashboard', to: '/app', icon: Home },
  { label: 'Log food', to: '/app/log-food', icon: Plus },
  { label: 'History', to: '/app/history', icon: CalendarDays },
  { label: 'Analysis', to: '/app/analysis', icon: Activity },
  { label: 'Foods & meals', to: '/app/meals', icon: UtensilsCrossed },
  { label: 'Recipes', to: '/app/recipes', icon: UtensilsCrossed },
  { label: 'Water, weight & activity', to: '/app/wellness', icon: Activity },
  { label: 'Health items', to: '/app/health', icon: HeartPulse },
  { label: 'Nutrition targets', to: '/app/targets', icon: Activity },
  { label: 'Settings', to: '/app/settings', icon: Settings2 },
]

function Brand({ compact = false }: { compact?: boolean }) {
  return <Link className="brand" to="/" aria-label="AteThat home">
    <span className="brand-mark" aria-hidden="true"><span /></span>
    {!compact && <span className="brand-text"><strong>AteThat</strong><small>Know what nourishes you</small></span>}
  </Link>
}

function ThemeButton() {
  const { theme, setTheme, resolvedTheme } = useTheme()
  const next = resolvedTheme === 'light' ? 'dark' : 'light'
  return <button className="icon-button" type="button" onClick={() => setTheme(next)} aria-label={`Switch to ${next} mode`} title={`Switch to ${next} mode`}>
    {resolvedTheme === 'light' ? <Moon size={19} /> : <Sun size={19} />}
    <span className="sr-only">Current preference: {theme}</span>
  </button>
}

function PublicHeader() {
  return <header className="public-header"><div className="public-header-inner">
    <Brand />
    <nav aria-label="Public navigation"><ThemeButton /><Link className="text-link" to="/login">Log in</Link><Link className="button button-primary button-small" to="/signup">Get started <ArrowRight size={16} /></Link></nav>
  </div></header>
}

// Access tokens intentionally live only in memory. On a fresh browser load, the
// HttpOnly refresh cookie is the authority for restoring a valid session.
function RedirectIfAuthenticated({ children }: { children: ReactNode }) {
  const navigate = useNavigate()
  const [checking, setChecking] = useState(true)
  useEffect(() => {
    let active = true
    if (getAccessToken()) {
      navigate('/app', { replace: true })
      return () => { active = false }
    }
    refreshAccessToken().then(session => {
      if (!active) return
      setAccessToken(session.access_token)
      navigate('/app', { replace: true })
    }).catch(() => { if (active) setChecking(false) })
    return () => { active = false }
  }, [navigate])
  if (checking) return <div className="not-found"><Brand /><p>Checking your session…</p></div>
  return <>{children}</>
}

function Landing() {
  return <div className="public-page"><PublicHeader />
    <main>
      <section className="hero layout-width"><div className="hero-copy">
        <span className="eyebrow"><span className="eyebrow-dot" /> A calmer way to keep track</span>
        <h1 tabIndex={-1}>Know what <em>nourishes</em> you.</h1>
        <p>Keep food, nutrition, medication, and supplements in one clear place. Understand your day without losing the details that matter to you.</p>
        <div className="hero-actions"><Link className="button button-primary" to="/signup">Create your account <ArrowRight size={18} /></Link><Link className="button button-outline" to="/login">Log in</Link></div>
        <div className="hero-note"><ShieldCheck size={18} /><span>Your records are private to your account.</span></div>
      </div><div className="hero-visual" aria-label="Illustration of a calm daily overview">
        <div className="hero-visual-top"><span>Daily overview</span><span className="visual-date">Today</span></div>
        <div className="visual-energy"><span>Energy</span><strong>See the whole day<br />at a glance.</strong><div className="visual-track"><span /></div></div>
        <div className="visual-nutrients"><div><span className="nutrient-dot protein" />Protein</div><div><span className="nutrient-dot carbs" />Carbs</div><div><span className="nutrient-dot fat" />Fat</div><div><span className="nutrient-dot fibre" />Fibre</div></div>
        <div className="visual-bottom"><span className="visual-icon"><UtensilsCrossed size={18} /></span><div><strong>Food and routines, together</strong><small>Open a section to see the details.</small></div><ArrowRight size={18} /></div>
      </div></section>
      <section className="overview-section"><div className="layout-width">
        <div className="section-heading"><span className="eyebrow">Made for everyday clarity</span><h2>A thoughtful space for your routine.</h2><p>A focused overview when you want it, with more detail available when you need it.</p></div>
        <div className="feature-grid"><Feature icon={UtensilsCrossed} title="Food, in context" text="Review one or several foods together before adding them to your day." /><Feature icon={Activity} title="Nutrition, made readable" text="See energy first, then your routine nutrients and source details." /><Feature icon={HeartPulse} title="Your health routine" text="Keep medication and supplement records alongside your daily view." /></div>
      </div></section>
    </main><PublicFooter />
  </div>
}

function PublicFooter() {
  return <footer className="public-footer layout-width"><Brand /><div className="public-footer-links"><Link to="/terms">Terms and conditions</Link><span>© AteThat</span></div></footer>
}

function TermsPage() {
  return <div className="public-page"><PublicHeader /><main className="terms-page layout-width">
    <div className="terms-heading"><span className="eyebrow">AteThat · Legal</span><h1 tabIndex={-1}>Terms and conditions</h1><p className="muted">Draft for local testing · Last updated 8 October 2026</p></div>
    <div className="terms-content">
      <section><h2>About AteThat</h2><p>AteThat helps you record food, nutrition, and health routines. Features shown in the current local version are still under development. Availability, accuracy, and these terms may change before a public launch.</p></section>
      <section><h2>Your account and information</h2><p>Keep your sign-in credentials secure and provide information you have the right to use. You control the records you add to your account. Nutrition information, including USDA-derived values, can be incomplete or vary by product, preparation, and serving size.</p></section>
      <section><h2>Health information</h2><p>AteThat is a tracking tool, not a medical professional or a substitute for medical advice, diagnosis, or treatment. Check important nutrition or medication decisions with a qualified professional.</p></section>
      <section><h2>Account deletion and permanent removal</h2><p>A verified account-deletion request starts a <strong>30-day deletion-pending period</strong> and ends account access immediately. You may cancel the request during that period only by freshly verifying your sign-in method. At the end of the 30 days, user-owned account data is scheduled for permanent removal, except information we must retain to meet applicable legal obligations. Removed data may not be recoverable.</p><p>The 30-day period does not start merely by stopping use of AteThat or deleting a browser session.</p></section>
      <section><h2>Service changes</h2><p>During development, the service may be interrupted, changed, or reset. Please do not rely on this local version as the only copy of important records. We will review and complete these terms, including operator and contact details, before public launch.</p></section>
      <section><h2>Data storage and recovery</h2><p>AteThat currently uses MongoDB Atlas Free (M0) without application-managed backups, snapshots, or point-in-time recovery. We cannot guarantee restoration of records lost through accidental deletion, application errors, unauthorized access, or a storage-service failure. Keep a separate copy of information that is important to you. We do not currently promise a recovery time or a maximum amount of data that could be lost. This wording will be updated if verified backup and recovery capabilities are introduced.</p></section>
    </div>
    <Link className="quiet-link" to="/"><ArrowLeft size={16} /> Back to home</Link>
  </main><PublicFooter /></div>
}

function Feature({ icon: IconComponent, title, text }: { icon: Icon; title: string; text: string }) {
  return <article className="feature-card"><span className="feature-icon"><IconComponent size={21} /></span><h3>{title}</h3><p>{text}</p></article>
}

type SessionResponse = { access_token: string; user?: { email?: string } }

type PasswordCredentialConstructor = new (data: { id: string; password: string }) => Credential

async function offerBrowserPasswordSave(email: string, password: string) {
  if (!email || !password || typeof window === 'undefined' || !navigator.credentials?.store) return
  const PasswordCredential = (window as Window & { PasswordCredential?: PasswordCredentialConstructor }).PasswordCredential
  if (!PasswordCredential) return
  try {
    await navigator.credentials.store(new PasswordCredential({ id: email, password }))
  } catch {
    // Password-manager availability and any save prompt are browser settings.
  }
}

function AuthPage({ mode }: { mode: 'login' | 'signup' | 'forgot' | 'resend' }) {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [error, setError] = useState(params.get('error') === 'google' ? 'Google sign-in could not be completed. If this email already has an AteThat account, log in with your password; account linking will be available separately.' : '')
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const isLogin = mode === 'login'
  const isSignup = mode === 'signup'

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setMessage('')
    const values = Object.fromEntries(new FormData(event.currentTarget))
    if (isSignup && String(values.password || '').length < 12) {
      setError('Use at least 12 characters for your password.')
      return
    }
    setBusy(true)
    try {
      if (isLogin) {
        const session = await postJson<SessionResponse>('/auth/login', { email: values.email, password: values.password })
        await offerBrowserPasswordSave(String(values.email || ''), String(values.password || ''))
        setAccessToken(session.access_token)
        navigate('/app')
      } else if (isSignup) {
        await postJson<{ message: string }>('/auth/register', { display_name: values.display_name, email: values.email, password: values.password, timezone: values.timezone })
        await offerBrowserPasswordSave(String(values.email || ''), String(values.password || ''))
        setMessage('Check your email for a verification link before logging in.')
      } else if (mode === 'forgot') {
        await postJson<{ message: string }>('/auth/forgot-password', { email: values.email })
        setMessage('If an eligible account exists, a secure sign-in or reset link has been sent.')
      } else {
        await postJson<{ message: string }>('/auth/resend-verification', { email: values.email })
        setMessage('If verification is still needed, a new email has been sent.')
      }
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Something went wrong. Please try again.')
    } finally { setBusy(false) }
  }

  return <div className="auth-page"><PublicHeader /><main className="auth-main layout-width">
    <aside className="auth-intro"><span className="eyebrow"><span className="eyebrow-dot" /> Your space, your pace</span><h1 tabIndex={-1}>Make room for a clearer daily picture.</h1><p>AteThat brings the essentials together in a calm, readable experience. Your data remains yours.</p><div className="auth-benefits"><div><UtensilsCrossed size={19} /> Food and nutrition in one place</div><div><Clock3 size={19} /> Timezone-aware daily view</div><div><LockKeyhole size={19} /> Private account access</div></div></aside>
    <section className="auth-card" aria-labelledby="auth-heading"><div className="auth-card-top"><span className="auth-icon"><LockKeyhole size={22} /></span><Link className="quiet-link" to="/"><ArrowLeft size={15} /> Back to home</Link></div>
      <h2 id="auth-heading">{isLogin ? 'Welcome back' : isSignup ? 'Create your account' : mode === 'forgot' ? 'Reset your password' : 'Resend verification'}</h2>
      <p className="muted">{isLogin ? 'Sign in to continue to your daily overview.' : isSignup ? 'Start with the basics. You can complete your profile later.' : mode === 'forgot' ? 'Enter your email and we’ll send the right secure link for your account.' : 'Enter the email you used to sign up.'}</p>
      {(isLogin || isSignup) && <button className="button button-google" type="button" disabled={import.meta.env.VITE_GOOGLE_SIGNIN_ENABLED !== 'true'} title={import.meta.env.VITE_GOOGLE_SIGNIN_ENABLED === 'true' ? 'Continue with Google' : 'Configure Google OAuth to enable'} onClick={() => { window.location.assign(`${apiBaseUrl}/auth/google/start?return_to=%2Fapp`) }}><GoogleMark /> Continue with Google</button>}
      {(isLogin || isSignup) && <div className="divider"><span>or continue with email</span></div>}
      <Form className="auth-form" onSubmit={submit}>
        {isSignup && <label>Display name<input name="display_name" autoComplete="name" required maxLength={80} placeholder="Your name" /></label>}
        <label>Email address<input name="email" type="email" autoComplete={isLogin ? 'username' : 'email'} required placeholder="name@example.com" /></label>
        {(isLogin || isSignup) && <label>Password<span className="password-field"><input name="password" type={showPassword ? 'text' : 'password'} autoComplete={isSignup ? 'new-password' : 'current-password'} required minLength={isSignup ? 12 : 1} placeholder="Enter your password" /><button type="button" onClick={() => setShowPassword(!showPassword)} aria-label={showPassword ? 'Hide password' : 'Show password'}>{showPassword ? 'Hide' : 'Show'}</button></span>{isSignup && <small>Use 12 to 128 characters.</small>}</label>}
        {isSignup && <label>Default timezone<select name="timezone" defaultValue="Asia/Kolkata">{['Asia/Kolkata', 'UTC', ...Intl.supportedValuesOf('timeZone').filter(zone => zone !== 'Asia/Kolkata')].map(zone => <option key={zone} value={zone}>{zone}</option>)}</select><small>Change this manually in settings when you travel.</small></label>}
        {isLogin && <Link className="form-side-link" to="/forgot-password">Forgot password?</Link>}
        {error && <p className="form-message form-error" role="alert">{error}</p>}
        {message && <p className="form-message form-success" role="status">{message}</p>}
        <button className="button button-primary auth-submit" type="submit" disabled={busy}>{busy ? 'Please wait…' : isLogin ? 'Log in' : isSignup ? 'Create account' : mode === 'forgot' ? 'Email me a secure link' : 'Resend email'} <ArrowRight size={17} /></button>
      </Form>
      {isSignup && <p className="auth-terms">Read the <Link to="/terms">Terms and conditions</Link> before creating an account.</p>}
      <p className="auth-switch">{isLogin ? <>New to AteThat? <Link to="/signup">Create an account</Link> · <Link to="/resend-verification">Resend verification</Link></> : isSignup ? <>Already have an account? <Link to="/login">Log in</Link> · <Link to="/resend-verification">Resend verification</Link></> : <Link to="/login">Return to login</Link>}</p>
    </section>
  </main></div>
}

function GoogleMark() {
  return <svg aria-hidden="true" viewBox="0 0 24 24" width="19" height="19"><path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09Z"/><path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23Z"/><path fill="#FBBC05" d="M5.84 14.09A6.6 6.6 0 0 1 5.49 12c0-.73.13-1.43.35-2.09V7.06H2.18A10.95 10.95 0 0 0 1 12c0 1.78.43 3.45 1.18 4.94l3.66-2.85Z"/><path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52Z"/></svg>
}

function TokenPage({ mode }: { mode: 'verify' | 'reset' }) {
  const [params] = useSearchParams()
  const token = params.get('token') || ''
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token) { setError('This link is missing its one-time token. Request a new email.'); return }
    const values = Object.fromEntries(new FormData(event.currentTarget))
    if (mode === 'reset' && String(values.new_password || '').length < 12) {
      setError('Use at least 12 characters for your new password.')
      return
    }
    setBusy(true); setError(''); setMessage('')
    try {
      if (mode === 'verify') {
        await postJson<{ email_verified: boolean }>('/auth/verify-email', { token })
        setMessage('Your email is verified. You can now log in.')
      } else {
        await postJson<void>('/auth/reset-password', { token, new_password: values.new_password })
        clearAccessToken()
        setMessage('Your password has been changed. Please log in again.')
      }
    } catch (cause) {
      const detail = cause instanceof Error ? cause.message : 'Please try again.'
      if (mode === 'reset' && /reset token expired or already used/i.test(detail)) {
        setError('This password-reset link can no longer be used. Request a new email, then open the newest link.')
      } else if (mode === 'verify' && /verification token expired or already used/i.test(detail)) {
        setError('This verification link can no longer be used. Request a new verification email, then open the newest link.')
      } else {
        setError(detail)
      }
    }
    finally { setBusy(false) }
  }
  return <div className="auth-page"><PublicHeader /><main className="auth-main layout-width"><aside className="auth-intro"><span className="eyebrow">Your account</span><h1 tabIndex={-1}>{mode === 'verify' ? 'One step to get started.' : 'A fresh start, securely.'}</h1><p>{mode === 'verify' ? 'Verify the email address you used for AteThat.' : 'Choose a new password for your account.'}</p></aside><section className="auth-card"><h2>{mode === 'verify' ? 'Verify your email' : 'Reset your password'}</h2><Form className="auth-form" onSubmit={submit}>{mode === 'reset' && <label>New password<input name="new_password" type="password" autoComplete="new-password" required minLength={12} maxLength={128} /><small>Use 12 to 128 characters.</small></label>}{error && <p className="form-message form-error" role="alert">{error}</p>}{message && <p className="form-message form-success" role="status">{message}</p>}<button className="button button-primary auth-submit" type="submit" disabled={busy || !token}>{busy ? 'Please wait…' : mode === 'verify' ? 'Verify email' : 'Change password'}</button></Form><p className="auth-switch"><Link to="/login">Return to login</Link></p></section></main></div>
}

function EmailLoginPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const token = params.get('token') || ''
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token) { setError('This link is missing its one-time token. Request a new secure link.'); return }
    const values = Object.fromEntries(new FormData(event.currentTarget))
    if (String(values.new_password || '').length < 12) {
      setError('Use at least 12 characters for your new password.')
      return
    }
    setBusy(true)
    setError('')
    try {
      const session = await postJson<SessionResponse>('/auth/email-login', { token, new_password: values.new_password })
      await offerBrowserPasswordSave(session.user?.email || '', String(values.new_password || ''))
      setAccessToken(session.access_token)
      navigate('/app', { replace: true })
    } catch (cause) {
      const detail = cause instanceof Error ? cause.message : 'Please try again.'
      setError(/email sign-in link expired or already used/i.test(detail)
        ? 'This secure link can no longer be used. Request a new one and open the newest email.'
        : detail)
    } finally {
      setBusy(false)
    }
  }

  return <div className="auth-page"><PublicHeader /><main className="auth-main layout-width"><aside className="auth-intro"><span className="eyebrow">Your account</span><h1 tabIndex={-1}>Add a password, securely.</h1><p>This one-time link lets you add email-and-password sign-in to your AteThat account.</p></aside><section className="auth-card"><h2>Set your password</h2><p className="muted">Use at least 12 characters. You can still sign in with Google afterwards.</p><Form className="auth-form" onSubmit={submit}><label>New password<input name="new_password" type="password" autoComplete="new-password" required minLength={12} maxLength={128} /><small>This link works once and expires after 15 minutes.</small></label>{error && <p className="form-message form-error" role="alert">{error}</p>}<button className="button button-primary auth-submit" type="submit" disabled={busy || !token}>{busy ? 'Saving password…' : 'Set password and sign in'}</button></Form><p className="auth-switch"><Link to="/login">Return to login</Link></p></section></main></div>
}

type FoodDraftState = { draft: DraftFood[]; category: string; label: string }
const FoodDraftContext = createContext<{ value: FoodDraftState; setValue: React.Dispatch<React.SetStateAction<FoodDraftState>> } | null>(null)
function FoodDraftProvider({ children }: { children: ReactNode }) {
  const [value, setValue] = useState<FoodDraftState>({ draft: [], category: 'OTHER', label: '' })
  const location = useLocation()
  useEffect(() => { if (!location.pathname.startsWith('/app')) setValue({ draft: [], category: 'OTHER', label: '' }) }, [location.pathname])
  useEffect(() => { const clear = () => setValue({ draft: [], category: 'OTHER', label: '' }); window.addEventListener('atethat:session-expired', clear); return () => window.removeEventListener('atethat:session-expired', clear) }, [])
  return <FoodDraftContext.Provider value={{ value, setValue }}>{children}</FoodDraftContext.Provider>
}
function RoutePosition() {
  const location = useLocation()
  useEffect(() => { window.scrollTo({ top: 0, behavior: 'instant' }); document.querySelector<HTMLElement>('main h1')?.focus({ preventScroll: true }) }, [location.pathname])
  return null
}

function AppShell({ children }: { children: ReactNode }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const [authenticated, setAuthenticated] = useState(!!getAccessToken())
  const [profileChecked, setProfileChecked] = useState(false)
  const [missingProfile, setMissingProfile] = useState<MissingProfileDetail[]>([])
  const [profileError, setProfileError] = useState('')
  const [profileAttempt, setProfileAttempt] = useState(0)
  useEffect(() => {
    const changed = () => setProfileAttempt(n => n + 1)
    window.addEventListener('atethat:profile-updated', changed)
    return () => window.removeEventListener('atethat:profile-updated', changed)
  }, [])
  const navigate = useNavigate()
  const location = useLocation()
  useEffect(() => {
    const redirectToLogin = () => {
      clearAccessToken()
      setAuthenticated(false)
      setMenuOpen(false)
      navigate('/login', { replace: true })
    }
    window.addEventListener('atethat:session-expired', redirectToLogin)
    return () => window.removeEventListener('atethat:session-expired', redirectToLogin)
  }, [navigate])
  useEffect(() => {
    if (getAccessToken()) { setAuthenticated(true); return }
    let active = true
    refreshAccessToken().then(session => {
      if (active) { setAccessToken(session.access_token); setAuthenticated(true) }
    }).catch(() => { if (active) navigate('/login', { replace: true }) })
    return () => { active = false }
  }, [navigate])
  useEffect(() => {
    if (!authenticated) return
    let active = true
    getAuthenticatedJson<{onboarding_complete:boolean; date_of_birth?:string; sex?:string; height?:object; weight?:object; activity_level?:string}>('/profile').then(profile => {
      if (!active) return
      if (!profile.onboarding_complete && location.pathname !== '/app/welcome') navigate('/app/welcome', {replace:true})
      setMissingProfile(missingProfileDetails(profile))
      setProfileChecked(true)
    }).catch(() => { if(active) setProfileError('Your profile could not be loaded.') })
    return () => { active = false }
  }, [authenticated, navigate, profileAttempt])
  if (authenticated && !profileChecked) return <div className="not-found"><Brand />{profileError ? <><p role="alert">{profileError}</p><button className="button button-primary" onClick={() => {setProfileError('');setProfileAttempt(n=>n+1)}}>Try again</button></> : <Loading>Preparing your account…</Loading>}</div>
  if (!authenticated) return <div className="not-found"><Brand /><p>Checking your session…</p></div>
  return <div className="app-shell"><aside className={`sidebar ${menuOpen ? 'sidebar-open' : ''}`}><div className="sidebar-head"><Brand /><button className="icon-button mobile-only" type="button" onClick={() => setMenuOpen(false)} aria-label="Close menu"><X size={20} /></button></div><nav className="sidebar-nav" aria-label="Main navigation">{navigation.map(({ label, to, icon: IconComponent }) => <NavLink onClick={() => setMenuOpen(false)} key={to} to={to === '/app/settings' && missingProfile.length ? profileAttentionHref(missingProfile) : to} end={to === '/app'} className={({ isActive }) => `nav-item ${isActive ? 'nav-item-active' : ''}`}><IconComponent size={19} /><span>{label}</span>{to === '/app/settings' && missingProfile.length > 0 && <small className="settings-profile-badge" title={`${missingProfile.length} optional profile details not provided`} aria-label={`${missingProfile.length} optional profile inputs missing`}>{missingProfile.length}</small>}</NavLink>)}</nav><div className="sidebar-foot"><span className="sidebar-foot-icon"><ShieldCheck size={18} /></span><p>Know what nourishes you.<small>Your daily view, at your pace.</small></p></div></aside>
    {menuOpen && <button className="mobile-scrim" type="button" aria-label="Close menu" onClick={() => setMenuOpen(false)} />}
    <div className="app-main"><header className="app-header"><div className="app-header-left"><button className="icon-button mobile-only" type="button" aria-label="Open menu" onClick={() => setMenuOpen(true)}><Menu size={21} /></button><span className="app-header-title">AteThat</span></div><div className="app-header-right"><span className="header-timezone"><Clock3 size={15} /> Time shown in your account timezone</span><Link className="icon-button" to="/app/welcome?replay=1" aria-label="Open feature guide"><CircleHelp size={20} /></Link><ThemeButton /><Link className="button button-primary button-small header-log" to="/app/log-food"><Plus size={17} /> Log food</Link></div></header><main className="content"><div className="app-page-enter" key={location.pathname}>{children}</div></main></div>
    <nav className="bottom-nav" aria-label="Mobile navigation">{navigation.slice(0, 4).map(({ label, to, icon: IconComponent }) => <NavLink key={to} to={to === '/app/settings' && missingProfile.length ? profileAttentionHref(missingProfile) : to} end={to === '/app'} className={({ isActive }) => isActive ? 'bottom-item bottom-item-active' : 'bottom-item'}><IconComponent size={20} /><span>{label}</span></NavLink>)}<button type="button" className="bottom-item" onClick={() => setMenuOpen(true)}><Menu size={20} /><span>More</span></button></nav>
  </div>
}

function PageHead({ eyebrow, title, description, action }: { eyebrow?: string; title: string; description: string; action?: ReactNode }) {
  return <div className="page-head"><div>{eyebrow && <span className="eyebrow">{eyebrow}</span>}<h1 tabIndex={-1}>{title}</h1><p>{description}</p></div>{action}</div>
}

function EmptyState({ icon: IconComponent, title, text, action }: { icon: Icon; title: string; text: string; action?: ReactNode }) {
  return <div className="empty-state"><span className="empty-icon"><IconComponent size={26} /></span><h2>{title}</h2><p>{text}</p>{action}</div>
}

type DashboardData = { tracking_day: string; timezone: string; daily_reset_time: string; nutrition: { totals: FoodResult['nutrients']; targets: FoodResult['nutrients']; progress: Record<string, { percent: string }> }; food_log_groups: { items: LoggedGroup[]; has_more: boolean } }
function Dashboard() {
  const [data, setData] = useState<DashboardData | null>(null)
  const [upcoming, setUpcoming] = useState<HealthHistoryEntry[]>([])
  const [upcomingLoading, setUpcomingLoading] = useState(true)
  const [upcomingError, setUpcomingError] = useState('')
  const [error, setError] = useState('')
  useEffect(() => { getAuthenticatedJson<DashboardData>('/dashboard').then(setData)
    .catch(cause => setError(cause instanceof Error ? cause.message : 'Could not load your dashboard.'))
    const from = new Date()
    const to = new Date(from.getTime() + 24 * 60 * 60 * 1000)
    getAuthenticatedJson<{ items: HealthHistoryEntry[] }>(`/health-items/occurrences?${new URLSearchParams({ from: from.toISOString(), to: to.toISOString(), limit: '5' })}`)
      .then(page => setUpcoming(page.items)).catch(() => setUpcomingError('Could not load your health routine.')).finally(() => setUpcomingLoading(false)) }, [])
  if (!data) return <><PageHead title="Today at a glance" description="Your food and routines for today." />{error ? <p role="alert">{error}</p> : <Loading>Loading your day…</Loading>}</>
  const energy = data.nutrition.totals.ENERGY
  return <><PageHead eyebrow="Your daily view" title="Today at a glance" description="Your food and routines for today." action={<Link className="button button-primary" to="/app/log-food"><Plus size={17} /> Log food</Link>} />
    {error && <p className="form-message form-error" role="alert">{error}</p>}{data && <p className="muted">Tracking Day {data.tracking_day} · {data.timezone} · daily reset {data.daily_reset_time}</p>}
    <div className="dashboard-grid"><section className="panel energy-panel"><div className="panel-top"><span className="panel-kicker">Energy</span><span className="info-pill"><Info size={14} /> {data?.timezone || 'Account timezone'}</span></div><h2>{energy?.status === 'KNOWN' ? `${energy.amount} kcal` : 'No confirmed energy total yet'}</h2><p>{data?.nutrition.targets.ENERGY?.status === 'KNOWN' ? `Against your selected daily calorie target of ${data.nutrition.targets.ENERGY.amount} kcal. This is not a personalized calorie prescription.` : 'Choose nutrition targets in Settings to see progress.'}</p>{data.nutrition.targets.ENERGY?.status === 'KNOWN' && <div className="empty-meter"><span style={{ width: `${Math.min(100, Math.max(0, Number(data?.nutrition.progress.ENERGY?.percent || 0)))}%` }} /></div>}</section>
      <section className="panel nutrient-panel"><div className="panel-top"><h2>Routine nutrients</h2><CircleHelp size={18} aria-hidden="true" /></div>{[['PROTEIN', 'Protein', 'protein'], ['CARBOHYDRATE', 'Carbohydrates', 'carbs'], ['FAT', 'Fat', 'fat'], ['FIBER', 'Fibre', 'fibre']].map(([key, label, color]) => <div className="nutrient-row" key={key}><div><span className={`nutrient-dot ${color}`} />{label}</div><span>{data?.nutrition.totals[key]?.status === 'KNOWN' ? `${data.nutrition.totals[key].amount} ${data.nutrition.totals[key].unit}` : 'unknown'}</span>{data.nutrition.targets[key]?.status === 'KNOWN' && <div className="nutrient-track"><span style={{ width: `${Math.min(100, Math.max(0, Number(data?.nutrition.progress[key]?.percent || 0)))}%` }} /></div>}</div>)}<Link className="quiet-link" to="/app/targets">View nutrition targets <ArrowRight size={16} /></Link></section></div>
    <div className="dashboard-grid dashboard-secondary"><section className="panel"><div className="panel-top"><h2>Today’s food</h2><Link className="quiet-link" to="/app/history">View history <ArrowRight size={15} /></Link></div>{data?.food_log_groups.items.length ? data.food_log_groups.items.map(group => <div className="dashboard-group" key={group.id}><strong>{group.label || foodLabel(group.meal_category)}</strong><span>{group.item_count} {group.item_count === 1 ? 'food' : 'foods'} · {group.aggregate_final_nutrition.ENERGY?.status === 'KNOWN' ? `${group.aggregate_final_nutrition.ENERGY.amount} kcal` : 'energy unknown'}</span></div>) : <EmptyState icon={UtensilsCrossed} title="No food logged yet" text="Use Log food above to add your first meal." />}</section><section className="panel"><div className="panel-top"><h2>Health routine</h2><Link className="quiet-link" to="/app/health">View items <ArrowRight size={15} /></Link></div>{upcomingLoading ? <Loading>Loading your routine…</Loading> : upcomingError ? <p role="status">{upcomingError}</p> : upcoming.length ? upcoming.map(item => <div className="dashboard-group" key={item.occurrence_id}><strong>{item.name}</strong><span>{item.scheduled_local.date} {item.scheduled_local.time} ({item.scheduled_local.timezone}) · {healthStatusLabel(item.status, item.type)}</span></div>) : <EmptyState icon={HeartPulse} title="No upcoming items" text="Scheduled medication, supplements and tasks for the next 24 hours appear here." />}</section></div>
    <WellnessOverview />
    <Link className="quiet-link" to="/app/target-calculations">How targets are calculated</Link>
    <TextSummary from={data.tracking_day} to={data.tracking_day} mode="DAY" />
  </>
}

type FoodResult = { data_type?: string; food_reference: { type: 'USDA'; provider_food_id: string }; name: string; food_state: string; base_quantity: { amount: string; unit: string }; nutrients: Record<string, { status: string; amount?: string; unit: string }>; source_url: string }
type FoodPage = { items: FoodResult[]; page: { next_cursor: string | null; has_more: boolean } }
type CustomFood = { id: string; version: number; name: string; description: string; food_state: string; base_quantity: { amount: string; unit: string }; nutrients: FoodResult['nutrients'] }
type SavedMeal = { id: string; version: number; name: string; recipe_servings?: string; preparation?: string; components: { position: number; food_reference: { type: 'USDA' | 'CUSTOM'; provider_food_id: string | null; custom_food_id: string | null }; quantity: { amount: string; unit: string }; food_state: string }[] }
type SavedFoodLink = { food_key: string; food_reference: { type: 'USDA' | 'CUSTOM'; provider_food_id: string | null; custom_food_id: string | null }; name: string; confirmed_log_count?: number }
type SavedServing = { id: string; version: number; label: string; food_reference: { type: 'USDA' | 'CUSTOM'; provider_food_id: string | null; custom_food_id: string | null }; quantity: { amount: string; unit: string }; food_state: string }
type DraftFood = { key: string; type: 'USDA' | 'CUSTOM'; id: string; name: string; foodState: string; quantity: string; unit: string; portion?: { label: string; amount: string }; portionCount?: string; overrides: Record<string, { status: string; amount: string; unit: string }> }
type LoggedGroup = { id: string | null; tracking_day: string; item_count: number; meal_category: string; label: string; aggregate_final_nutrition: FoodResult['nutrients']; items: { id: string | null; position: number; food_source: { name: string }; quantity: { amount: string; unit: string }; final_nutrition: FoodResult['nutrients'] }[] }
type GroupPreview = { preview_token: string; expires_at: string; group: LoggedGroup }


function LogFood() {
  const dialogs = useDialogs()
  const [searchMode, setSearchMode] = useState<'LOCAL'|'USDA'>('LOCAL')
  const [usage, setUsage] = useState<SearchUsage | null>(null)
  const [usageError, setUsageError] = useState('')
  async function refreshUsage() { try { const result = await getAuthenticatedJson<{usage:SearchUsage}>('/foods/search?source=STATUS'); setUsage(previous => ({...result.usage, matched_count:previous?.matched_count || 0})); setUsageError('') } catch { setUsageError('Search allowance could not be loaded. Try again.') } }
  useEffect(() => { void refreshUsage() }, [])
  const [query, setQuery] = useState('')
  const [state, setState] = useState('')
  const [source, setSource] = useState<'USDA' | 'CUSTOM' | 'MEAL'>('USDA')
  const [savedSource, setSavedSource] = useState<'FAVOURITES' | 'RECENT' | 'FREQUENT' | ''>('')
  const [items, setItems] = useState<FoodResult[]>([])
  const [customFoods, setCustomFoods] = useState<CustomFood[]>([])
  const [meals, setMeals] = useState<SavedMeal[]>([])
  const [savedLinks, setSavedLinks] = useState<SavedFoodLink[]>([])
  const [savedCursor, setSavedCursor] = useState<string | null>(null)
  const [showServings, setShowServings] = useState(false)
  const [servings, setServings] = useState<SavedServing[]>([])
  const [servingCursor, setServingCursor] = useState<string | null>(null)
  const draftContext = useContext(FoodDraftContext)!
  const { draft, category, label } = draftContext.value
  const setDraft = (update: React.SetStateAction<DraftFood[]>) => draftContext.setValue(previous => ({ ...previous, draft: typeof update === 'function' ? update(previous.draft) : update }))
  const setCategory = (category: string) => draftContext.setValue(previous => ({ ...previous, category }))
  const setLabel = (label: string) => draftContext.setValue(previous => ({ ...previous, label }))
  const mounted = useRef(true)
  useEffect(() => { mounted.current = true; return () => { mounted.current = false } }, [])
  const [basketOpen, setBasketOpen] = useState(false)
  const [notice, setNotice] = useState('')
  const [sourceLoading, setSourceLoading] = useState(false)
  const [savedLoading, setSavedLoading] = useState(false)
  const [searching, setSearching] = useState(false)
  const noticeRef = useRef<HTMLDivElement>(null)
  const [reviewDay, setReviewDay] = useState<{ tracking_day: string; timezone: string; daily_reset_time: string } | null>(null)
  useEffect(() => { getAuthenticatedJson<typeof reviewDay>('/tracking-day/current').then(setReviewDay).catch(() => {}) }, [])
  const [preview, setPreview] = useState<GroupPreview | null>(null)
  const [confirmationKey, setConfirmationKey] = useState('')
  const [confirmed, setConfirmed] = useState<LoggedGroup | null>(null)
  useEffect(() => { if (confirmed) noticeRef.current?.scrollIntoView({ block: 'center' }) }, [confirmed])
  const [cursor, setCursor] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [searched, setSearched] = useState(false)
  useEffect(() => {
    if (source !== 'CUSTOM') return
    let active = true; setSourceLoading(true); setError('')
    getAuthenticatedJson<{ items: CustomFood[] }>('/foods/custom?limit=50').then(page => { if (active) setCustomFoods(page.items) })
      .catch(cause => { if (active) setError(cause instanceof Error ? cause.message : 'Could not load your foods.') }).finally(() => { if (active) setSourceLoading(false) })
    return () => { active = false }
  }, [source])
  useEffect(() => {
    if (source !== 'MEAL') return
    let active = true; setSourceLoading(true); setError('')
    getAuthenticatedJson<{ items: SavedMeal[] }>('/meals?limit=50').then(page => { if (active) setMeals(page.items) })
      .catch(cause => { if (active) setError(cause instanceof Error ? cause.message : 'Could not load saved meals.') }).finally(() => { if (active) setSourceLoading(false) })
    return () => { active = false }
  }, [source])
  useEffect(() => {
    if (!savedSource) return
    let active = true; setSavedLoading(true); setSavedLinks([]); setSavedCursor(null)
    const path = savedSource === 'FAVOURITES' ? '/foods/favourites' : savedSource === 'RECENT' ? '/foods/recent' : '/foods/frequent'
    getAuthenticatedJson<{ items: SavedFoodLink[]; page: { next_cursor: string | null } }>(`${path}?limit=20`)
      .then(page => { if (active) { setSavedLinks(page.items); setSavedCursor(page.page.next_cursor) } })
      .catch(cause => { if (active) setError(cause instanceof Error ? cause.message : 'Could not load foods.') }).finally(() => { if (active) setSavedLoading(false) })
    return () => { active = false }
  }, [savedSource])
  useEffect(() => {
    let active = true
    getAuthenticatedJson<{ items: SavedServing[]; page: { next_cursor: string | null } }>('/servings?limit=20')
      .then(page => { if (active) { setServings(page.items); setServingCursor(page.page.next_cursor) } })
      .catch(cause => { if (active) setError(cause instanceof Error ? cause.message : 'Could not load servings.') })
    return () => { active = false }
  }, [])
  async function addSavedLink(link: SavedFoodLink) {
    setBusy(true); setError('')
    try {
      const food = link.food_reference.type === 'USDA'
        ? await getAuthenticatedJson<FoodResult>(`/foods/provider/${encodeURIComponent(link.food_reference.provider_food_id || '')}`)
        : await getAuthenticatedJson<CustomFood>(`/foods/custom/${encodeURIComponent(link.food_reference.custom_food_id || '')}`)
      addFood(food)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not load this food.') }
    finally { setBusy(false) }
  }
  async function moreSavedLinks() {
    if (!savedCursor) return
    setBusy(true); setError('')
    try {
      const path = savedSource === 'FAVOURITES' ? '/foods/favourites' : savedSource === 'RECENT' ? '/foods/recent' : '/foods/frequent'
      const page = await getAuthenticatedJson<{ items: SavedFoodLink[]; page: { next_cursor: string | null } }>(`${path}?${new URLSearchParams({ limit: '20', cursor: savedCursor })}`)
      setSavedLinks(previous => [...previous, ...page.items]); setSavedCursor(page.page.next_cursor)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not load more foods.') }
    finally { setBusy(false) }
  }
  async function favourite(food: DraftFood) {
    setBusy(true); setError('')
    try {
      await postAuthenticatedJson('/foods/favourites', { food_reference: food.type === 'CUSTOM' ? { type: 'CUSTOM', custom_food_id: food.id } : { type: 'USDA', provider_food_id: food.id } })
      setNotice(`${food.name} saved to favourites.`)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not save favourite.') }
    finally { setBusy(false) }
  }
  async function removeFavourite(link: SavedFoodLink) {
    setBusy(true); setError('')
    try {
      await deleteAuthenticatedJson(`/foods/favourites/${encodeURIComponent(link.food_key)}`)
      setSavedLinks(previous => previous.filter(item => item.food_key !== link.food_key))
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not remove favourite.') }
    finally { setBusy(false) }
  }
  function addFood(food: FoodResult | CustomFood, servingAmount?: string, servingState?: string) {
    if (!mounted.current) return
    if (draft.length >= 50) { setError('You can log up to 50 foods in one meal.'); return }
    const custom = 'id' in food
    setDraft(previous => [...previous, { key: crypto.randomUUID(), type: custom ? 'CUSTOM' : 'USDA',
      id: custom ? food.id : food.food_reference.provider_food_id, name: food.name,
      foodState: servingState || food.food_state, quantity: servingAmount || food.base_quantity.amount, unit: food.base_quantity.unit, overrides: {} }])
    setPreview(null); setConfirmed(null); setNotice(`${food.name} added. Review your selected foods when ready.`)
  }
  async function addServing(serving: SavedServing) {
    setBusy(true); setError('')
    try {
      const food = serving.food_reference.type === 'USDA'
        ? await getAuthenticatedJson<FoodResult>(`/foods/provider/${encodeURIComponent(serving.food_reference.provider_food_id || '')}`)
        : await getAuthenticatedJson<CustomFood>(`/foods/custom/${encodeURIComponent(serving.food_reference.custom_food_id || '')}`)
      addFood(food, serving.quantity.amount, serving.food_state)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not use serving.') }
    finally { setBusy(false) }
  }
  async function saveServing(food: DraftFood) {
    const label = (await dialogs.prompt(`Name this serving of ${food.name}`, `1 serving of ${food.name}`))?.trim()
    if (!label) return
    setBusy(true); setError('')
    try {
      const item = await postAuthenticatedJson<SavedServing>('/servings', { label,
        food_reference: food.type === 'USDA' ? { type: 'USDA', provider_food_id: food.id } : { type: 'CUSTOM', custom_food_id: food.id },
        quantity: { amount: food.quantity, unit: food.unit }, food_state: food.foodState })
      setShowServings(true); setServings(previous => [item, ...previous]); setNotice(`Household portion “${item.label}” saved.`)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not save serving.') }
    finally { setBusy(false) }
  }
  async function moreServings() {
    if (!servingCursor) return
    setBusy(true); setError('')
    try {
      const page = await getAuthenticatedJson<{ items: SavedServing[]; page: { next_cursor: string | null } }>(`/servings?${new URLSearchParams({ limit: '20', cursor: servingCursor })}`)
      setServings(previous => [...previous, ...page.items]); setServingCursor(page.page.next_cursor)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not load more servings.') }
    finally { setBusy(false) }
  }
  async function renameServing(serving: SavedServing) {
    const label = (await dialogs.prompt('Serving label', serving.label))?.trim()
    if (!label || label === serving.label) return
    setBusy(true); setError('')
    try {
      const updated = await patchAuthenticatedJson<SavedServing>(`/servings/${serving.id}`, { version: serving.version,
        label, food_reference: serving.food_reference, quantity: serving.quantity, food_state: serving.food_state })
      setServings(previous => previous.map(item => item.id === serving.id ? updated : item))
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not rename serving.') }
    finally { setBusy(false) }
  }
  async function deleteServing(serving: SavedServing) {
    if (!(await dialogs.confirm(`Delete serving ${serving.label}? Confirmed logs remain unchanged.`))) return
    setBusy(true); setError('')
    try { await deleteAuthenticatedJson(`/servings/${serving.id}`); setServings(previous => previous.filter(item => item.id !== serving.id)) }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not delete serving.') }
    finally { setBusy(false) }
  }
  function editDraft(key: string, quantity: string) {
    setDraft(previous => previous.map(item => item.key === key ? { ...item, quantity, overrides: {}, portion: undefined, portionCount: undefined } : item))
    setPreview(null)
  }
  function setOverride(key: string, nutrient: string, amount: string, unit: string) {
    setDraft(previous => previous.map(item => {
      if (item.key !== key) return item
      const overrides = { ...item.overrides }
      if (amount.trim()) overrides[nutrient] = { status: 'KNOWN', amount: amount.trim(), unit }
      else delete overrides[nutrient]
      return { ...item, overrides }
    }))
    setPreview(null)
  }
  async function search(event?: FormEvent<HTMLFormElement>, nextCursor?: string) {
    event?.preventDefault()
    if (query.trim().length < 2) return
    setSearching(true); setError('')
    try {
      const params = new URLSearchParams({ q: query.trim(), limit: '20', source: searchMode })
      if (state) params.set('food_state', state)
      if (nextCursor) params.set('cursor', nextCursor)
      const response = await getAuthenticatedJson<FoodPage & {usage:SearchUsage}>(`/foods/search?${params}`)
      setUsage(response.usage)
      setItems(previous => nextCursor ? [...previous, ...response.items] : response.items)
      setCursor(response.page.next_cursor)
      setSearched(true)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Food search is unavailable right now.') }
    finally { setSearching(false); void refreshUsage() }
  }
  async function review() {
    if (draft.length === 0 || draft.some(item => !Number.isFinite(Number(item.quantity)) || Number(item.quantity) <= 0)) { setError('Enter a positive amount for every food.'); setBasketOpen(true); return }
    setBusy(true); setError(''); setConfirmed(null)
    try {
      const response = await postAuthenticatedJson<GroupPreview>('/food-log-groups/preview', {
        logged_at: new Date().toISOString(), meal_category: category, label,
        items: draft.map((item, position) => ({ position,
          food_reference: item.type === 'USDA' ? { type: 'USDA', provider_food_id: item.id } : { type: 'CUSTOM', custom_food_id: item.id },
          quantity: { amount: item.quantity, unit: item.unit }, food_state: item.foodState, user_override_nutrition: item.overrides || {} })),
      })
      setPreview(response); setBasketOpen(false); setConfirmationKey(crypto.randomUUID())
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not preview the group.') }
    finally { setBusy(false) }
  }
  async function saveMeal() {
    const name = (await dialogs.prompt('Name this reusable meal'))?.trim()
    if (!name) return
    setBusy(true); setError('')
    try {
      await postAuthenticatedJson('/meals', { name, components: draft.map((item, position) => ({ position,
        food_reference: item.type === 'USDA' ? { type: 'USDA', provider_food_id: item.id } : { type: 'CUSTOM', custom_food_id: item.id },
        quantity: { amount: item.quantity, unit: item.unit }, food_state: item.foodState })) })
      setNotice(`Meal “${name}” saved to Foods & meals.`)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not save meal.') }
    finally { setBusy(false) }
  }
  async function reviewMeal(meal: SavedMeal) {
    if (draft.length && !(await dialogs.confirm('Replace your selected foods with this saved meal?'))) return
    const servings = meal.recipe_servings ? (await dialogs.prompt(`This recipe makes ${meal.recipe_servings} servings. How many servings did you eat?`, '1')) : undefined
    if (meal.recipe_servings && !servings) return
    setBusy(true); setError(''); setConfirmed(null)
    try {
      const response = await postAuthenticatedJson<GroupPreview>(`/meals/${meal.id}/preview`, {
        logged_at: new Date().toISOString(), meal_category: category, label: label || meal.name, ...(servings ? { servings } : {}) })
      if (!mounted.current) return
      setPreview(response); setBasketOpen(false); setConfirmationKey(crypto.randomUUID())
      setLabel(label || meal.name)
      setDraft(meal.components.map((component, index) => ({
        key: crypto.randomUUID(), type: component.food_reference.type,
        id: component.food_reference.type === 'USDA' ? component.food_reference.provider_food_id || '' : component.food_reference.custom_food_id || '',
        name: response.group.items[index].food_source.name, foodState: component.food_state,
        quantity: response.group.items[index].quantity.amount, unit: response.group.items[index].quantity.unit, overrides: {},
      })))
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not preview saved meal.') }
    finally { setBusy(false) }
  }
  async function confirm() {
    if (!preview) return
    setBusy(true); setError('')
    try {
      const group = await postAuthenticatedIdempotentJson<LoggedGroup>('/food-log-groups/confirm', { preview_token: preview.preview_token }, confirmationKey)
      if (!mounted.current) return
      setConfirmed(group); setPreview(null); setDraft([]); setLabel(''); setBasketOpen(false); setNotice('')
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not confirm the group.') }
    finally { setBusy(false) }
  }
  const mappedServings = (food: DraftFood) => servings.filter(serving => serving.food_reference.type === food.type && (food.type === 'CUSTOM' ? serving.food_reference.custom_food_id : serving.food_reference.provider_food_id) === food.id && serving.quantity.unit === food.unit && serving.food_state === food.foodState)
  const basket = <fieldset className="recipe-fieldset basket-editor" disabled={busy}>
    <div className="panel-top">{!basketOpen && <h2>Selected foods</h2>}<span className="info-pill">{draft.length} {draft.length === 1 ? 'food' : 'foods'}</span></div>
    {!draft.length && <p className="muted">Search and add a food to get started.</p>}
    {draft.map(item => <div className="draft-food" key={item.key}>
      <strong>{item.name}</strong><div><label>Amount ({item.portion ? 'servings' : item.unit})<input aria-label={`Amount of ${item.name}`} type="number" min="0.001" step="any" value={item.portion ? item.portionCount || '1' : item.quantity} onChange={event => {
        if (!item.portion) editDraft(item.key, event.target.value)
        else { const count = event.target.value; setDraft(previous => previous.map(food => food.key === item.key ? { ...food, portionCount: count, quantity: count ? String(Number(count) * Number(item.portion!.amount)) : '', overrides: {} } : food)); setPreview(null) }
      }} /></label><button className="icon-button" type="button" aria-label={`Remove ${item.name}`} disabled={busy} onClick={() => { setDraft(previous => previous.filter(food => food.key !== item.key)); setPreview(null) }}><X size={17} /></button></div>
      {mappedServings(item).length > 0 && <label className="draft-meta">Portion<select aria-label={`Portion for ${item.name}`} value={item.portion?.label || ''} onChange={event => { const serving = mappedServings(item).find(value => value.label === event.target.value); setDraft(previous => previous.map(food => food.key === item.key ? { ...food, quantity: serving?.quantity.amount || food.quantity, portion: serving ? { label: serving.label, amount: serving.quantity.amount } : undefined, portionCount: '1', overrides: {} } : food)); setPreview(null) }}><option value="">{item.unit === 'g' ? 'Grams' : 'Millilitres'}</option>{mappedServings(item).map(serving => <option key={serving.id} value={serving.label}>{serving.label} = {serving.quantity.amount} {serving.quantity.unit}</option>)}</select></label>}
      {item.portion && <small className="muted">{item.quantity} {item.unit} total</small>}
      <details className="secondary-details"><summary>More options for {item.name}</summary><div className="secondary-actions"><button className="quiet-link" disabled={busy} onClick={() => void favourite(item)}>Save favourite</button><button className="quiet-link" disabled={busy} onClick={() => void saveServing(item)}>Save household portion</button></div>
        {!mappedServings(item).length && <p className="muted">Weigh the food once to save your own household portion, such as 1 bowl = 150 g. We do not guess weights.</p>}
        <details><summary>Edit nutrition</summary><p className="muted">Values apply to the amount above. Leave blank to use the source. Changing the amount clears these edits.</p><NutrientOverrideInputs values={item.overrides} onChange={(code, amount, unit) => setOverride(item.key, code, amount, unit)} /></details>
      </details>
    </div>)}
    {draft.length > 0 && <><label className="draft-meta">Meal<select value={category} onChange={event => { setCategory(event.target.value); setPreview(null) }}>{['BREAKFAST','LUNCH','DINNER','SNACK','OTHER'].map(value => <option key={value} value={value}>{foodLabel(value)}</option>)}</select></label>
      <details className="secondary-details"><summary>Name or save this meal</summary><label className="draft-meta">Meal name (optional)<input maxLength={160} value={label} onChange={event => { setLabel(event.target.value); setPreview(null) }} /></label><button className="quiet-link" disabled={busy} onClick={saveMeal}>Save as reusable meal</button></details>
      <button className="button button-primary full-width" disabled={busy || draft.length > 50} onClick={review}>{busy ? 'Preparing review…' : 'Review'}</button>
    </>}
  </fieldset>
  return <><PageHead eyebrow="Food logging" title="Log food" description="Search, add your foods, then review and log your meal." />
    <div className="page-notice" ref={noticeRef}>{confirmed && <p className="form-message form-success" role="status">Logged {confirmed.item_count} {confirmed.item_count === 1 ? 'food' : 'foods'} for {confirmed.tracking_day}. <Link to={`/app/history?tracking_day=${confirmed.tracking_day}`}>View history</Link></p>}</div>
    {notice && <p className="compact-notice" role="status">{notice}</p>}{error && <p className="form-message form-error" role="alert">{error}</p>}
    <div className="food-log-layout"><section className="panel food-discovery">
      <div className="filter-row" role="group" aria-label="Food source">{(['USDA','CUSTOM','MEAL'] as const).map(value => <button className={`filter-chip ${source === value ? 'filter-chip-active' : ''}`} aria-pressed={source === value} key={value} onClick={() => { if (value !== source) { setSourceLoading(value !== 'USDA'); setSource(value) } setError('') }}>{value === 'USDA' ? 'Search foods' : value === 'CUSTOM' ? 'My foods' : 'Saved meals'}</button>)}</div>
      {source === 'USDA' ? <>
        <div className="search-source-switch" role="group" aria-label="Search database">{(['LOCAL','USDA'] as const).map(mode => <button type="button" key={mode} className={`filter-chip ${searchMode === mode ? 'filter-chip-active' : ''}`} aria-pressed={searchMode === mode} disabled={searching} onClick={() => { setSearchMode(mode); setItems([]); setCursor(null); setSearched(false); setError('') }}>{mode === 'LOCAL' ? 'Local database' : 'Search USDA'}</button>)}</div>
        <div className="search-account-status" aria-live="polite">{usage ? <><span>{usage.catalogue_count.toLocaleString()} foods in local database</span><span>{usage.local_searches_today} local searches today · unlimited</span><span>{usage.usda_remaining} of {usage.usda_limit} USDA searches left today</span><small>USDA allowance resets at {new Date(usage.resets_at).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit',timeZone:'UTC'})} UTC. Your custom foods are in My foods.</small></> : <span>{usageError || 'Loading search allowance…'}</span>}{usageError && <button type="button" className="quiet-link" onClick={() => void refreshUsage()}>Retry</button>}</div>
        <Form className="food-search-form" onSubmit={event => search(event)}><label className="search-field"><Search size={19} /><input type="search" placeholder="Search foods, e.g. banana" aria-label="Search foods" data-validation-message="Enter a food name with 2–100 characters." maxLength={100} value={query} onChange={event => setQuery(event.target.value)} minLength={2} required /></label><button className="button button-primary" disabled={searching || (searchMode === 'USDA' && usage?.usda_remaining === 0)}>{searching ? 'Searching…' : searchMode === 'USDA' ? 'Search USDA' : 'Search local'}</button><details className="search-filters"><summary>Filters</summary><label>Preparation<select value={state} onChange={event => setState(event.target.value)}><option value="">Any preparation</option><option value="RAW">Raw</option><option value="COOKED">Cooked</option></select></label></details></Form>
        {!searched && !searching && <p className="muted">{searchMode === 'LOCAL' ? 'Search foods already in the database without using your USDA allowance. Can’t find a food? Choose Search USDA.' : 'Find new foods in USDA and add them to the shared local database. Each search or next page uses one search from your daily allowance.'}</p>}
        <div className="food-results">{searched && usage && <p className="muted" role="status">{usage.matched_count.toLocaleString()} matches in {searchMode === 'LOCAL' ? 'the local database' : 'USDA'}</p>}{rankFoods(items, query).map(item => <article className="food-result" key={item.food_reference.provider_food_id}><div><h3>{item.name}</h3><p>{item.data_type === 'Branded' ? 'Packaged product' : 'USDA food'} · per {item.base_quantity.amount} {item.base_quantity.unit}{item.food_state !== 'UNKNOWN' ? ` · ${foodLabel(item.food_state)}` : ''}</p><p>{display(item.nutrients.ENERGY)}</p><details className="secondary-details"><summary>Source & nutrition</summary><p>{['PROTEIN','CARBOHYDRATE','FAT','FIBER'].map(code => `${foodLabel(code)}: ${display(item.nutrients[code])}`).join(' · ')}</p><p>{item.data_type || 'USDA'} · FDC {item.food_reference.provider_food_id}{item.food_state === 'UNKNOWN' ? ' · Preparation not supplied by source' : ''}</p><a className="quiet-link" href={item.source_url} target="_blank" rel="noreferrer">View USDA record</a></details></div><button className="button button-outline button-small" aria-label={`Add ${item.name} (FDC ${item.food_reference.provider_food_id})`} disabled={busy || draft.length >= 50} onClick={() => addFood(item)}>Add</button></article>)}</div>
        {searched && !searching && !items.length && !error && <p className="muted">No matching foods. Try another name or preparation filter.</p>}{cursor && <button className="quiet-link" disabled={searching} onClick={() => void search(undefined, cursor)}>Load more foods</button>}
      </> : sourceLoading ? <Loading>Loading {source === 'CUSTOM' ? 'your foods' : 'saved meals'}…</Loading> : source === 'CUSTOM' ? <div className="food-results">{!customFoods.length && !error && <p>No foods saved yet. <Link to="/app/meals">Create a food</Link>; your selected foods will stay here.</p>}{customFoods.map(food => <article className="food-result" key={food.id}><div><h3>{food.name}</h3><p>{food.base_quantity.amount} {food.base_quantity.unit} · {foodLabel(food.food_state)}</p></div><button className="button button-outline button-small" disabled={busy || draft.length >= 50} onClick={() => addFood(food)}>Add</button></article>)}</div> : <div className="food-results">{!meals.length && !error && <p>No saved meals yet. Add foods and choose “Save as reusable meal” under Selected foods.</p>}{meals.filter(meal => !meal.recipe_servings).map(meal => <article className="food-result" key={meal.id}><div><h3>{meal.name}</h3><p>{meal.components.length} {meal.components.length === 1 ? 'food' : 'foods'}</p></div><button className="button button-outline button-small" disabled={busy} onClick={() => void reviewMeal(meal)}>Review meal</button></article>)}<Link className="quiet-link" to="/app/recipes">Log a recipe portion →</Link></div>}
      <details className="secondary-details saved-foods"><summary>Recent, favourites & saved portions</summary><div className="filter-row">{(['RECENT','FAVOURITES','FREQUENT'] as const).map(value => <button key={value} className={`filter-chip ${savedSource === value ? 'filter-chip-active' : ''}`} aria-pressed={savedSource === value} onClick={() => setSavedSource(value)}>{foodLabel(value)}</button>)}</div>
        {savedLoading ? <Loading /> : savedSource && !savedLinks.length ? <p className="muted">No {savedSource.toLowerCase()} foods yet.</p> : savedLinks.map(link => <article className="food-result" key={link.food_key}><h3>{link.name}</h3><div className="history-actions"><button className="button button-outline button-small" disabled={busy} onClick={() => void addSavedLink(link)}>Add</button>{savedSource === 'FAVOURITES' && <button className="quiet-link" disabled={busy} onClick={() => void removeFavourite(link)}>Unsave</button>}</div></article>)}{savedCursor && <button className="quiet-link" disabled={busy} onClick={() => void moreSavedLinks()}>More saved foods</button>}
        <button className="quiet-link" aria-expanded={showServings} onClick={() => setShowServings(!showServings)}>{showServings ? 'Hide' : 'Show'} saved portions</button>{showServings && <>{!servings.length && <p className="muted">Save a household portion from a selected food’s More options menu.</p>}{servings.map(serving => <article className="food-result" key={serving.id}><div><h3>{serving.label}</h3><p>{serving.quantity.amount} {serving.quantity.unit}</p></div><div className="history-actions"><button className="button button-outline button-small" disabled={busy} onClick={() => void addServing(serving)}>Add</button><details><summary>Manage</summary><button className="quiet-link" disabled={busy} onClick={() => void renameServing(serving)}>Rename</button><button className="quiet-link" disabled={busy} onClick={() => void deleteServing(serving)}>Delete</button></details></div></article>)}{servingCursor && <button className="quiet-link" disabled={busy} onClick={() => void moreServings()}>More portions</button>}</>}
      </details>
    </section><aside className="panel food-basket">{basket}</aside></div>
    {draft.length > 0 && !preview && <div className="food-mobile-action"><button className="quiet-link" type="button" aria-label="Edit selected foods" disabled={busy} onClick={() => setBasketOpen(true)}>{draft.length} {draft.length === 1 ? 'food' : 'foods'} · Edit</button><button className="button button-primary" disabled={busy} onClick={review}>{busy ? 'Preparing…' : 'Review'}</button></div>}
    {basketOpen && !preview && <Modal title="Your selected foods" onClose={() => setBasketOpen(false)} busy={busy}>{error && <p role="alert" className="form-message form-error">{error}</p>}{basket}</Modal>}
    {preview && <Modal title="Review your meal" onClose={() => setPreview(null)} busy={busy}>
      <p>{label || foodLabel(category)} · {preview.group.tracking_day}{reviewDay ? ` · ${reviewDay.timezone}` : ''}</p>
      {preview.group.items.map(item => <div className="review-food" key={item.position}><strong>{item.food_source.name}</strong><span>{item.quantity.amount} {item.quantity.unit}</span></div>)}
      <NutritionSummary values={preview.group.aggregate_final_nutrition} />
      {error && <p className="form-message form-error" role="alert">{error}</p>}
      <div className="dialog-actions"><button className="button button-outline" disabled={busy} onClick={() => { setPreview(null); setBasketOpen(true) }}>Edit foods</button><button className="button button-primary" disabled={busy} onClick={confirm}>{busy ? 'Logging…' : 'Log food'}</button></div>

    </Modal>}
  </>
}

type HistoryPageResponse = { items: LoggedGroup[]; page: { next_cursor: string | null; has_more: boolean } }
type NutritionDay = { tracking_day: string; timezone: string; daily_reset_time: string; totals: FoodResult['nutrients']; confirmed_group_count: number; confirmed_item_count: number; detail_compacted: boolean }

function HistoryPage() {
  const dialogs = useDialogs()
  const [queryParams] = useSearchParams()
  const [day, setDay] = useState('')
  const [groups, setGroups] = useState<LoggedGroup[]>([])
  const [summary, setSummary] = useState<NutritionDay | null>(null)
  const [cursor, setCursor] = useState<string | null>(null)
  const [expanded, setExpanded] = useState<Record<string, LoggedGroup>>({})
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [loading, setLoading] = useState(true)
  const [correction, setCorrection] = useState<{ preview: GroupPreview; key: string } | null>(null)
  async function reloadSummary() {
    const daily = await getAuthenticatedJson<{ items: NutritionDay[] }>(`/history/nutrition?from_tracking_day=${day}&to_tracking_day=${day}&limit=1`)
    setSummary(daily.items[0] || null)
  }
  useEffect(() => {
    const requested = queryParams.get('tracking_day')
    if (requested && /^\d{4}-\d{2}-\d{2}$/.test(requested)) { setDay(requested); return }
    getAuthenticatedJson<{ tracking_day: string }>('/tracking-day/current').then(value => setDay(value.tracking_day))
      .catch(cause => setError(cause instanceof Error ? cause.message : 'Could not load Tracking Day.'))
  }, [queryParams])
  useEffect(() => {
    if (!day) return
    let active = true; setLoading(true); setError(''); setGroups([]); setSummary(null)
    const params = new URLSearchParams({ tracking_day: day, limit: '20' })
    Promise.all([getAuthenticatedJson<HistoryPageResponse>(`/food-log-groups?${params}`),
      getAuthenticatedJson<{ items: NutritionDay[] }>(`/history/nutrition?from_tracking_day=${day}&to_tracking_day=${day}&limit=1`)]).then(([response, daily]) => {
      if (active) { setGroups(response.items); setCursor(response.page.next_cursor); setSummary(daily.items[0] || null); setExpanded({}) }
    }).catch(cause => { if (active) setError(cause instanceof Error ? cause.message : 'Could not load food history.') }).finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [day])
  async function more() {
    if (!cursor) return
    setBusy(true); setError('')
    try {
      const params = new URLSearchParams({ tracking_day: day, limit: '20', cursor })
      const response = await getAuthenticatedJson<HistoryPageResponse>(`/food-log-groups?${params}`)
      setGroups(previous => [...previous, ...response.items]); setCursor(response.page.next_cursor)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not load more history.') }
    finally { setBusy(false) }
  }
  async function expand(id: string) {
    if (expanded[id]) { setExpanded(previous => { const next = { ...previous }; delete next[id]; return next }); return }
    try {
      const detail = await getAuthenticatedJson<LoggedGroup>(`/food-log-groups/${id}`)
      setExpanded(previous => ({ ...previous, [id]: detail }))
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not load group details.') }
  }
  async function remove(id: string) {
    if (!(await dialogs.confirm('Remove this entire confirmed food group from your visible history and totals?'))) return
    setBusy(true); setError('')
    try {
      await deleteAuthenticatedJson<void>(`/food-log-groups/${id}`)
      setGroups(previous => previous.filter(group => group.id !== id))
      await reloadSummary()
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not remove the group.') }
    finally { setBusy(false) }
  }
  async function correct(item: LoggedGroup['items'][number]) {
    if (!item.id) return
    const values = await dialogs.fields('Correct food amount', [
      {name:'amount', label:`Amount (${item.quantity.unit})`, type:'number', value:item.quantity.amount, min:'0.000001', step:'any'},
      {name:'reason', label:'Reason for correction (required)', minLength:3, validationMessage:'Enter a reason with at least 3 characters, excluding leading and trailing spaces (for example, “Incorrect amount”).'},
    ], item.food_source.name + '. Review the updated nutrition before saving.', 'Review correction')
    if (!values) return
    const {amount, reason} = values
    setBusy(true); setError('')
    try {
      const preview = await patchAuthenticatedJson<GroupPreview>(`/food-logs/${item.id}`, { quantity: { amount, unit: item.quantity.unit }, reason })
      setCorrection({ preview, key: crypto.randomUUID() })
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not prepare correction.') }
    finally { setBusy(false) }
  }
  async function confirmCorrection() {
    if (!correction) return
    setBusy(true); setError('')
    try {
      await postAuthenticatedIdempotentJson('/food-log-groups/confirm', { preview_token: correction.preview.preview_token }, correction.key)
      const params = new URLSearchParams({ tracking_day: day, limit: '20' })
      const response = await getAuthenticatedJson<HistoryPageResponse>(`/food-log-groups?${params}`)
      setGroups(response.items); setCursor(response.page.next_cursor); setExpanded({}); setCorrection(null)
      await reloadSummary()
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not confirm correction.') }
    finally { setBusy(false) }
  }
  return <><PageHead eyebrow="Your records" title="Food history" description="Your logged meals, organised by day." />
    <section className="panel section-panel"><AppDatePicker className="history-date" label="Tracking Day" value={day} onChange={setDay} />{error && <p className="form-message form-error" role="alert">{error}</p>}{summary && <p className="muted">Daily total: {summary.totals.ENERGY?.status === 'KNOWN' ? `${summary.totals.ENERGY.amount} kcal` : 'energy unknown'} · {summary.confirmed_group_count} {summary.confirmed_group_count === 1 ? 'meal' : 'meals'} · {summary.confirmed_item_count} {summary.confirmed_item_count === 1 ? 'food' : 'foods'} · {summary.timezone}{summary.detail_compacted ? ' · Older item details compacted after 90 days' : ''}</p>}{loading && !error && <Loading>Loading your food history…</Loading>}{!loading && !error && groups.length === 0 && !summary && <EmptyState icon={CalendarDays} title="No food logged for this day" text="Confirmed groups will appear here." />}{!loading && groups.map(group => <article className="history-group" key={group.id}><div><h2>{group.label || foodLabel(group.meal_category)}</h2><p>{group.item_count} {group.item_count === 1 ? 'food' : 'foods'} · {group.tracking_day}</p><p>Energy: {group.aggregate_final_nutrition.ENERGY?.status === 'KNOWN' ? `${group.aggregate_final_nutrition.ENERGY.amount} kcal` : 'unknown'}</p></div><div className="history-actions"><button className="button button-outline button-small" type="button" onClick={() => group.id && expand(group.id)}>{group.id && expanded[group.id] ? 'Hide details' : 'Show details'}</button><button className="quiet-link" type="button" disabled={busy} onClick={() => group.id && remove(group.id)}>Remove meal</button></div>{group.id && expanded[group.id] && <div className="history-details">{expanded[group.id].items.map(item => <div className="history-item" key={item.position}><p>{item.food_source.name} · {item.quantity.amount} {item.quantity.unit} · energy {item.final_nutrition.ENERGY?.status === 'KNOWN' ? `${item.final_nutrition.ENERGY.amount} kcal` : 'unknown'}</p><button className="quiet-link" type="button" disabled={busy} onClick={() => correct(item)}>Correct amount</button></div>)}</div>}</article>)}{cursor && <button className="button button-outline button-small" type="button" disabled={busy} onClick={more}>Load more</button>}{correction && <Modal title="Review correction" busy={busy} onClose={() => setCorrection(null)}><p className="muted">Confirm this updated meal to replace the previous amounts. The earlier revision is kept in your audit history.</p>{error && <p className="form-message form-error" role="alert">{error}</p>}{correction.preview.group.items.map(item => <div className="review-food" key={item.position}><strong>{item.food_source.name}</strong><span>{item.quantity.amount} {item.quantity.unit}</span></div>)}<NutritionSummary values={correction.preview.group.aggregate_final_nutrition} /><div className="dialog-actions"><button className="button button-outline" type="button" disabled={busy} onClick={() => setCorrection(null)}>Cancel</button><button className="button button-primary" type="button" disabled={busy} onClick={confirmCorrection}>{busy ? 'Saving…' : 'Confirm correction'}</button></div></Modal>}</section>
  </>
}

type HealthHistoryEntry = { occurrence_id: string; name: string; type: string; status: string; scheduled_at: string; scheduled_local: { date: string; time: string; timezone: string } }

function CustomFoodsPage() {
  const dialogs = useDialogs()
  const [creating, setCreating] = useState(false)
  const [loading, setLoading] = useState(true)
  const [foods, setFoods] = useState<CustomFood[]>([])
  const [foodCursor, setFoodCursor] = useState<string | null>(null)
  const [editing, setEditing] = useState<CustomFood | null>(null)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  async function reload() {
    const page = await getAuthenticatedJson<{ items: CustomFood[]; page: { next_cursor: string | null } }>('/foods/custom?limit=50')
    setFoods(page.items)
    setFoodCursor(page.page.next_cursor)
  }
  async function moreFoods() {
    if (!foodCursor) return
    setBusy(true); setError('')
    try {
      const page = await getAuthenticatedJson<{ items: CustomFood[]; page: { next_cursor: string | null } }>(`/foods/custom?${new URLSearchParams({ limit: '50', cursor: foodCursor })}`)
      setFoods(previous => [...previous, ...page.items]); setFoodCursor(page.page.next_cursor)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not load more foods.') }
    finally { setBusy(false) }
  }
  useEffect(() => { reload().catch(cause => setError(cause instanceof Error ? cause.message : 'Could not load custom foods.')).finally(() => setLoading(false)) }, [])
  async function saveFood(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError(''); setMessage('')
    const form = event.currentTarget
    const values = Object.fromEntries(new FormData(form))
    const nutrients: Record<string, { status: string; amount?: string; unit: string }> = { ...editing?.nutrients }
    for (const [key, unit] of nutrientFields) {
      const amount = String(values[key] || '').trim()
      nutrients[key] = amount ? { status: 'KNOWN', amount, unit } : { status: 'UNKNOWN', unit }
    }
    try {
      const body = { name: values.name, description: values.description,
        food_state: values.food_state, base_quantity: { amount: values.base_amount, unit: values.base_unit }, nutrients }
      if (editing) {
        await patchAuthenticatedJson(`/foods/custom/${editing.id}`, { ...body, version: editing.version })
      } else {
        await postAuthenticatedJson('/foods/custom', body)
      }
      form.reset(); setEditing(null); setCreating(false); await reload(); setMessage(editing ? 'Your custom food was updated.' : 'Your custom food was saved.')
    } catch (cause) { setError(cause instanceof Error ? cause.message : `Could not ${editing ? 'update' : 'save'} custom food.`) }
    finally { setBusy(false) }
  }
  async function remove(id: string) {
    if (!(await dialogs.confirm('Delete this custom food? Confirmed food logs keep their historical snapshot.'))) return
    setBusy(true); setError('')
    try { await deleteAuthenticatedJson<void>(`/foods/custom/${id}`); await reload() }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not delete food.') }
    finally { setBusy(false) }
  }
  return <><PageHead eyebrow="Your kitchen" title="Foods & meals" description="Your saved foods and reusable meals. Recipes with a serving yield have their own collection." action={<button className="button button-primary" onClick={() => setCreating(true)}>New food</button>} />
    {error && <p className="form-message form-error" role="alert">{error}</p>}{message && <p className="form-message form-success" role="status">{message}</p>}<div className="collection-layout"><section className="panel"><h2>My foods</h2>{loading ? <Loading>Loading your foods…</Loading> : !error && foods.length === 0 && <EmptyState icon={UtensilsCrossed} title="No custom foods yet" text="Save a product or food using nutrition values you know." />}{foods.map(food => <article className="food-result" key={food.id}><div><h3>{food.name}</h3><p>{food.base_quantity.amount} {food.base_quantity.unit} · {food.food_state.toLowerCase()}</p></div><div className="history-actions"><button className="quiet-link" type="button" disabled={busy} onClick={() => { setEditing(food); setError(''); setMessage('') }}>Edit</button><button className="quiet-link" type="button" disabled={busy} onClick={() => remove(food.id)}>Delete</button></div></article>)}{foodCursor && <button className="button button-outline button-small" type="button" disabled={busy} onClick={() => void moreFoods()}>Load more</button>}</section></div><SavedMealsManager />{(creating || editing) && <Modal title={editing ? `Edit ${editing.name}` : 'New food'} busy={busy} onClose={() => { setCreating(false); setEditing(null) }}><Form key={editing?.id || 'new'} className="custom-food-form" onSubmit={saveFood}><label>Name<input name="name" required minLength={2} maxLength={120} defaultValue={editing?.name || ''} /></label><label>Description<input name="description" maxLength={500} defaultValue={editing?.description || ''} /></label><label>Food state<select name="food_state" defaultValue={editing?.food_state || 'PREPARED'}><option value="PREPARED">Prepared</option><option value="RAW">Raw</option><option value="COOKED">Cooked</option><option value="UNKNOWN">Unknown</option></select></label><div className="custom-food-base"><label>Base amount<input name="base_amount" type="number" min="0.001" step="any" defaultValue={editing?.base_quantity.amount || '100'} required /></label><label>Unit<select name="base_unit" defaultValue={editing?.base_quantity.unit || 'g'}><option>g</option><option>ml</option></select></label></div><p className="muted">Enter nutrients for the base amount. Leave unknown values empty.</p><NutrientInputs values={editing?.nutrients} />{error && <p className="form-message form-error" role="alert">{error}</p>}{message && <p className="form-message form-success" role="status">{message}</p>}<div className="history-actions"><button className="button button-primary" type="submit" disabled={busy}>{busy ? 'Saving…' : editing ? 'Save changes' : 'Save custom food'}</button><button className="button button-outline" type="button" disabled={busy} onClick={() => { setEditing(null); setCreating(false); setError(''); setMessage('') }}>Cancel</button></div></Form></Modal>}
  </>
}

function SavedMealsManager() {
  const dialogs = useDialogs()
  const [loading, setLoading] = useState(true)
  const [meals, setMeals] = useState<SavedMeal[]>([])
  const [cursor, setCursor] = useState<string | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function load(nextCursor?: string) {
    const query = new URLSearchParams({ limit: '50' })
    if (nextCursor) query.set('cursor', nextCursor)
    const page = await getAuthenticatedJson<{ items: SavedMeal[]; page: { next_cursor: string | null } }>(`/meals?${query}`)
    setMeals(previous => nextCursor ? [...previous, ...page.items] : page.items)
    setCursor(page.page.next_cursor)
  }
  useEffect(() => { load().catch(cause => setError(cause instanceof Error ? cause.message : 'Could not load meals.')).finally(() => setLoading(false)) }, [])
  async function rename(meal: SavedMeal) {
    const name = (await dialogs.prompt('Saved meal name', meal.name))?.trim()
    if (!name || name === meal.name) return
    setBusy(true); setError('')
    try {
      await patchAuthenticatedJson(`/meals/${meal.id}`, { version: meal.version, name, components: meal.components, recipe_servings: meal.recipe_servings || '', preparation: meal.preparation || '' })
      await load()
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not rename meal.') }
    finally { setBusy(false) }
  }
  async function remove(meal: SavedMeal) {
    if (!(await dialogs.confirm(`Delete saved meal ${meal.name}? Historical food logs remain unchanged.`))) return
    setBusy(true); setError('')
    try { await deleteAuthenticatedJson(`/meals/${meal.id}`); await load() }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not delete meal.') }
    finally { setBusy(false) }
  }
  return <section className="panel section-panel"><div className="panel-top"><h2>Saved meals</h2><Link className="quiet-link" to="/app/log-food">Build a meal <ArrowRight size={15} /></Link></div>{error && <p className="form-message form-error" role="alert">{error}</p>}{loading && <Loading>Loading saved meals…</Loading>}{!loading && !error && meals.filter(meal => !meal.recipe_servings).length === 0 && <p className="muted">No saved meals yet. Add foods in Log food, then save the selection as a meal.</p>}{meals.filter(meal => !meal.recipe_servings).map(meal => <article className="food-result" key={meal.id}><div><h3>{meal.name}</h3><p>{meal.components.length} {meal.components.length === 1 ? 'food' : 'foods'}</p></div><div className="history-actions"><button className="quiet-link" type="button" disabled={busy} onClick={() => void rename(meal)}>Rename</button><button className="quiet-link" type="button" disabled={busy} onClick={() => void remove(meal)}>Delete</button></div></article>)}{cursor && <button className="button button-outline button-small" type="button" disabled={busy} onClick={() => void load(cursor)}>Load more</button>}</section>
}

type HealthItem = { id: string; version: number; type: string; name: string; dosage: { amount_text: string; unit_text: string }; instructions: string; notes: string; start_date: string; active: boolean; schedule: { timezone: string; recurrence: { frequency: string; interval: number; weekdays: string[]; local_times: string[] }; effective_from_date: string } }
function HealthPage() {
  const dialogs = useDialogs()
  const [scheduleEditor, setScheduleEditor] = useState<SchedulableItem | null>(null)
  const [items, setItems] = useState<HealthItem[]>([])
  const [occurrenceRevision, setOccurrenceRevision] = useState(0)
  const [timezone, setTimezone] = useState('')
  const [creating, setCreating] = useState(false)
  const [itemType, setItemType] = useState('SUPPLEMENT')
  const [loading, setLoading] = useState(true)
  const [times, setTimes] = useState(['08:00'])
  const [monthDays, setMonthDays] = useState([1])
  const day = calendarDay(timezone || 'UTC')
  const [startDate,setStartDate] = useState(''),[endDate,setEndDate] = useState('')
  const [frequency, setFrequency] = useState('DAILY')
  const [weekdays, setWeekdays] = useState<string[]>([])
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  async function reloadItems() {
    const result:HealthItem[]=[];let cursor:string|null=null
    do { const query=new URLSearchParams({limit:'50',...(cursor?{cursor}:{})});const page:Page<HealthItem>=await getAuthenticatedJson(`/health-items?${query}`);result.push(...page.items);cursor=page.page.next_cursor } while(cursor)
    setItems(result)
  }
  function reloadOccurrences() { setOccurrenceRevision(value=>value+1) }
  useEffect(() => {
    getAuthenticatedJson<{ timezone: string }>('/profile').then(profile => setTimezone(profile.timezone)).catch(() => setError('Could not load your schedule timezone. Please reload before adding an item.'))
    reloadItems().catch(cause => setError(cause instanceof Error ? cause.message : 'Could not load health items.')).finally(() => setLoading(false))
  }, [])
  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError(''); setMessage('')
    const form = event.currentTarget
    const values = Object.fromEntries(new FormData(form))
    const localTimes = new FormData(form).getAll('schedule_time').map(String)
    if (new Set(localTimes).size !== localTimes.length) { setError('Each scheduled time must be different.'); setBusy(false); return }
    if (!startDate || (endDate && endDate < startDate)) { setError('Choose a start date and an end date on or after it.'); setBusy(false); return }
    if (!timezone || !localTimes.length || localTimes.some(time => !/^\d{2}:\d{2}$/.test(time)) || (frequency === 'MONTHLY' && !monthDays.length) || (frequency === 'WEEKLY' && !weekdays.length)) { setError('Choose a valid time and the days for your schedule.'); setBusy(false); return }
    try {
      await postAuthenticatedJson('/health-items', { type: values.type, name: values.name,
        ...(values.type !== 'TASK' ? { dosage: { amount_text: values.dosage_amount, unit_text: values.dosage_unit } } : {}),
        instructions: values.instructions, notes: values.notes, start_date: values.start_date,
        schedule: { timezone, recurrence: { frequency, interval: Number(values.interval || '1'),
          weekdays: frequency === 'WEEKLY' ? weekdays : [], local_times: localTimes, month_days:frequency==='MONTHLY'?monthDays:[] },
          effective_from_date: values.start_date, effective_to_date:values.end_date||'' } })
      form.reset(); setCreating(false); setWeekdays([]); setTimes(['08:00']); setFrequency('DAILY'); setMessage('Health item saved. Times shown below use its schedule timezone.')
      await Promise.all([reloadItems(), reloadOccurrences()])
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not save health item.') }
    finally { setBusy(false) }
  }
  async function action(occurrence: HealthOccurrence, status: 'TAKEN' | 'SKIPPED') {
    setBusy(true); setError('')
    try {
      await putAuthenticatedJson(`/health-items/occurrences/${occurrence.occurrence_id}`, { status, expected_event_version:occurrence.event_version||0,
        ...(status === 'TAKEN' ? { completed_at: new Date().toISOString() } : {}), notes: '' })
      await reloadOccurrences()
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not record action.') }
    finally { setBusy(false) }
  }
  async function deactivate(item: HealthItem) {
    if (!(await dialogs.confirm(`Deactivate ${item.name}? Past recorded actions will remain visible.`))) return
    setBusy(true); setError('')
    try {
      await postAuthenticatedJson(`/health-items/${item.id}/deactivate`, { version: item.version, end_date: day })
      await Promise.all([reloadItems(), reloadOccurrences()])
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not deactivate item.') }
    finally { setBusy(false) }
  }
  async function editItem(item: HealthItem, changeSchedule: boolean) {
    if (changeSchedule) {setScheduleEditor(item); return}
    const name = (await dialogs.prompt('Health item name', item.name))?.trim()
    if (!name) return
    setBusy(true); setError('')
    try {
      await patchAuthenticatedJson(`/health-items/${item.id}`, { version: item.version, type: item.type, name,
        dosage: item.dosage, instructions: item.instructions, notes: item.notes, start_date: item.start_date })
      await Promise.all([reloadItems(), reloadOccurrences()])
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not update health item.') }
    finally { setBusy(false) }
  }
  async function reschedule(occurrence: HealthOccurrence) {
    const next = new Date(Date.parse(occurrence.scheduled_at) + 86400000)
    const local = new Date(next.getTime() - next.getTimezoneOffset()*60000).toISOString().slice(0,16)
    const values = await dialogs.fields('Reschedule occurrence', [{name:'time',label:'New date and time',type:'datetime-local',value:local}], `Time is shown in your device timezone (${Intl.DateTimeFormat().resolvedOptions().timeZone}).`, 'Reschedule')
    if (!values) return
    const target = new Date(values.time).toISOString()
    if (Date.parse(target)<=Date.now()) {setError('Choose a future date and time when rescheduling an unfinished occurrence.');return}
    setBusy(true); setError('')
    try {
      await putAuthenticatedJson(`/health-items/occurrences/${occurrence.occurrence_id}`, { status: 'RESCHEDULED', rescheduled_to: target, expected_event_version:occurrence.event_version||0, notes: '' })
      setMessage('Occurrence rescheduled and active at the new time. Select that date to view it. Its previous values remain in history.')
      await reloadOccurrences()
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not reschedule occurrence.') }
    finally { setBusy(false) }
  }
  return <><PageHead eyebrow="Your routine" title="Health items" description="Your medication, supplement and everyday task routine." action={<button className="button button-primary" disabled={!timezone} onClick={() => { setItemType('SUPPLEMENT'); setStartDate(day); setEndDate(''); setError(''); setCreating(true) }}>Add health item</button>} />
    {error && <p className="form-message form-error" role="alert">{error}</p>}{message && <p className="form-message form-success" role="status">{message}</p>}
    {timezone ? <HealthOccurrences key={timezone} timezone={timezone} revision={occurrenceRevision} busy={busy} onAction={action} onReschedule={reschedule}/> : <Loading>Loading your timezone…</Loading>}
    <details className="health-routines panel"><summary>Manage health items · {items.length}</summary><div className="health-routines-content"><h2>Your routines</h2>{loading ? <Loading>Loading your items…</Loading> : !error && items.length === 0 && <EmptyState icon={HeartPulse} title="No health items yet" text="Create an item with a schedule to see occurrences." />}{items.map(item => <article className="food-result" key={item.id}><div><h3>{item.name}</h3><p>{item.type.toLowerCase()}{item.type !== 'TASK' && (item.dosage.amount_text || item.dosage.unit_text) && <> · {item.dosage.amount_text} {item.dosage.unit_text}</>} · {item.schedule.recurrence.frequency === 'PAUSED' ? 'Schedule paused' : `${foodLabel(item.schedule.recurrence.frequency)} schedule`} · {item.schedule.timezone}</p><p>{item.active ? 'Item enabled' : 'Item disabled'}</p></div><details className="secondary-details"><summary>Manage {item.name}</summary><div className="secondary-actions"><button className="quiet-link" disabled={busy || !item.active} onClick={() => void editItem(item, false)}>Rename</button><button className="quiet-link" disabled={busy} onClick={() => void editItem(item, true)}>Schedule & history</button>{item.active && <button className="quiet-link" disabled={busy} onClick={() => void deactivate(item)}>Disable item</button>}</div></details></article>)}</div></details>{creating && <Modal title="Add a health item" busy={busy} onClose={() => setCreating(false)}>{error && <p className="form-message form-error" role="alert">{error}</p>}<Form className="custom-food-form health-form" onSubmit={create}><label>Type<select name="type" value={itemType} onChange={event => setItemType(event.target.value)}><option value="SUPPLEMENT">Supplement</option><option value="MEDICATION">Medication</option><option value="TASK">Task</option></select></label><label>Name<input name="name" minLength={2} maxLength={120} placeholder={itemType === 'TASK' ? 'For example, brush teeth or apply solution' : undefined} required /></label>{itemType !== 'TASK' && <div className="custom-food-base"><label>Amount<input name="dosage_amount" maxLength={80} placeholder="1" /></label><label>Unit<input name="dosage_unit" maxLength={80} placeholder="tablet" /></label></div>}<label>Instructions (your own words)<input name="instructions" maxLength={500} /></label><label>Notes<input name="notes" maxLength={1000} /></label><AppDatePicker label="Start date" name="start_date" value={startDate} onChange={setStartDate}/><label>Frequency<select value={frequency} onChange={event => setFrequency(event.target.value)}><option value="DAILY">Daily</option><option value="WEEKLY">Weekly</option><option value="MONTHLY">Monthly</option></select></label>{frequency === 'WEEKLY' && <div className="weekday-list">{['MON','TUE','WED','THU','FRI','SAT','SUN'].map(value => <label key={value}><input type="checkbox" checked={weekdays.includes(value)} onChange={() => setWeekdays(previous => previous.includes(value) ? previous.filter(day => day !== value) : [...previous, value])} />{value}</label>)}</div>}{frequency==='MONTHLY' && <MonthDays value={monthDays} onChange={setMonthDays} />}<ScheduleTimes value={times} onChange={setTimes} /><details><summary>Advanced schedule options</summary><div className="health-schedule-advanced"><label>Repeat every {frequency === 'WEEKLY' ? 'weeks' : frequency === 'MONTHLY' ? 'months' : 'days'}<input name="interval" type="number" min="1" max="365" defaultValue="1" /></label><AppDatePicker label="Last schedule date (optional)" name="end_date" value={endDate} min={startDate} onChange={setEndDate} clearable/></div></details><p className="muted">Schedule timezone: {timezone}. Change your account timezone in Settings before adding an item if needed.</p><button className="button button-primary" disabled={busy} type="submit">Save item</button></Form></Modal>}

    {scheduleEditor && <ScheduleEditor key={scheduleEditor.id} item={scheduleEditor} onClose={() => setScheduleEditor(null)} onSaved={async () => { await Promise.all([reloadItems(), reloadOccurrences()]); setMessage('Schedule version saved. Past recorded events are preserved.') }} />}
    <details className="health-audit"><summary>Completion statistics & audit history</summary><HealthAdherence revision={`${occurrenceRevision}`} /></details>
  </>
}

type TargetValue = { reference_value: string; active_value: string; unit: string; is_manual_override: boolean }
type TargetVersion = { id: string; revision: number; status: string; target_basis: string; effective_from_tracking_day: string; source_attribution: { name: string; publisher: string; version: string; information_url: string }; targets: Record<string, TargetValue> }
function TargetsPage() {
  const [items, setItems] = useState<TargetVersion[]>([])
  const [cursor, setCursor] = useState<string | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function load(next?: string) {
    setBusy(true)
    try {
      const page = await getAuthenticatedJson<{items: TargetVersion[]; page: {next_cursor: string | null}}>('/nutrition/targets/history?limit=20' + (next ? '&cursor=' + encodeURIComponent(next) : ''))
      setItems(previous => next ? [...previous, ...page.items] : page.items); setCursor(page.page.next_cursor)
    } catch (e) { setError(e instanceof Error ? e.message : 'Target history unavailable.') } finally { setBusy(false) }
  }
  useEffect(() => { void load() }, [])
  return <><PageHead title="Nutrition target history" description="Accepted revisions remain attached to their effective Tracking Day. Edit targets in Settings." action={<Link className="button button-primary" to="/app/settings#nutrition-targets">Edit targets in Settings</Link>} />{error && <p role="alert">{error}</p>}{items.map(version => <section className="panel" key={version.id}><h2>Revision {version.revision} · {version.status}</h2><p>Effective {version.effective_from_tracking_day}</p><TargetRows version={version} /><TargetSource version={version} /></section>)}{cursor && <button className="button button-outline" disabled={busy} onClick={() => void load(cursor)}>Load more revisions</button>}{!items.length && <p role="status">{busy ? 'Loading history…' : 'No target revisions yet.'}</p>}</>
}
function nutrientLabel(key: string) { return ({ ENERGY: 'Energy', PROTEIN: 'Protein', CARBOHYDRATE: 'Carbohydrates', FAT: 'Fat', FIBER: 'Fibre' } as Record<string, string>)[key] || key }
function TargetRows({ version }: { version: TargetVersion }) { return <div className="target-rows">{Object.entries(version.targets).map(([key, value]) => <div key={key}><span>{nutrientLabel(key)}</span><strong>{value.active_value} {value.unit}</strong></div>)}</div> }
function TargetSource({ version }: { version: TargetVersion }) { return <details className="target-source"><summary>Reference: {version.source_attribution.name}</summary><p>{version.source_attribution.publisher} · {version.source_attribution.version}. <a href={version.source_attribution.information_url} target="_blank" rel="noreferrer">View reference</a>. {version.target_basis === 'PERSONALIZED_ESTIMATE' ? 'Calculated defaults are approximate, use the profile inputs selected at that time, and are not a medical prescription. Selected custom values may differ.' : 'General benchmark metadata is not a personalized recommendation. Selected custom values may differ.'}</p></details> }

function Settings() {
  const [missingDetails, setMissingDetails] = useState<MissingProfileDetail[]>([])
  const settingsLocation = useLocation()
  const missingField = (key: string) => missingDetails.find(field => field.key === key)
  const profileFieldClass = (key: string) => missingField(key) ? 'profile-field-attention' : undefined
  useEffect(() => {
    const id = settingsLocation.hash.slice(1)
    if (id.startsWith('profile-field-')) focusProfileField(id)
    else if (id === 'profile-attention') { const notice = document.getElementById(id); notice?.scrollIntoView({behavior:'smooth',block:'start'}); notice?.focus({preventScroll:true}) }
  }, [settingsLocation.hash, missingDetails])
  const dialogs = useDialogs()
  const { theme, setTheme } = useTheme()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [profile, setProfile] = useState<{ version: number; display_name: string; timezone: string; date_of_birth?: string; height?: {amount:string;unit:string}; weight?: {amount:string;unit:string}; goal?:string; sex?:string; activity_level?:string; has_google_identity: boolean } | null>(null)
  const [preferences, setPreferences] = useState<{ version: number; daily_reset_time: string; unit_system: string; dashboard_nutrients: string[] } | null>(null)
  const [trackingDay, setTrackingDay] = useState<{ tracking_day: string; timezone: string; daily_reset_time: string; as_of_utc: string } | null>(null)
  const [saveMessage, setSaveMessage] = useState(params.get('deletion_cancelled') ? 'Your account deletion request was cancelled and your account was restored.' : '')
  const [saveError, setSaveError] = useState(params.get('error') === 'google_deletion' ? 'Google could not verify the deletion request.' : '')
  const [googleLinkMessage] = useState(params.get('linked') === 'google' ? 'Google sign-in is linked to your account.' : '')
  const [googleLinkError, setGoogleLinkError] = useState(params.get('error') === 'google_link_email_mismatch' ? 'Use the same Google email address that is already on this AteThat account.' : params.get('error') === 'google_link' ? 'Google account linking could not be completed.' : '')
  const [saving, setSaving] = useState(false)
  useEffect(() => {
    let active = true
    Promise.all([
      getAuthenticatedJson<{ version: number; display_name: string; timezone: string; date_of_birth?: string; height?: {amount:string;unit:string}; weight?: {amount:string;unit:string}; goal?:string; sex?:string; activity_level?:string; has_google_identity: boolean }>('/profile'),
      getAuthenticatedJson<{ version: number; daily_reset_time: string; unit_system: string; dashboard_nutrients: string[] }>('/preferences'),
      getAuthenticatedJson<{ tracking_day: string; timezone: string; daily_reset_time: string; as_of_utc: string }>('/tracking-day/current'),
    ]).then(([nextProfile, nextPreferences, nextTrackingDay]) => {
      if (active) { setProfile(nextProfile); setMissingDetails(missingProfileDetails(nextProfile)); setPreferences(nextPreferences); setTrackingDay(nextTrackingDay) }
    })
      .catch(cause => { if (active) setSaveError(cause instanceof Error ? cause.message : 'Could not load your profile.') })
    return () => { active = false }
  }, [])
  async function refreshTrackingDay() {
    setTrackingDay(await getAuthenticatedJson<{ tracking_day: string; timezone: string; daily_reset_time: string; as_of_utc: string }>('/tracking-day/current'))
  }
  async function saveProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!profile) return
    setSaving(true); setSaveError(''); setSaveMessage('')
    try {
      const values = Object.fromEntries(new FormData(event.currentTarget))
      const updated = await patchAuthenticatedJson<typeof profile>('/profile', { version: profile.version, display_name: values.display_name, timezone: values.timezone, date_of_birth: values.date_of_birth, goal:values.goal, sex:values.sex, ...(values.activity_level ? {activity_level:values.activity_level} : {}), ...(values.height ? {height:{amount:values.height,unit:values.height_unit}} : {}), ...(values.weight ? {weight:{amount:values.weight,unit:values.weight_unit}} : {}) })
      // Keep identity state stable even if an older API instance responds
      // without this derived field during a rolling deployment.
      setProfile({ ...updated, has_google_identity: updated.has_google_identity ?? profile.has_google_identity }); setMissingDetails(missingProfileDetails(updated))
      await refreshTrackingDay()
      window.dispatchEvent(new Event('atethat:profile-updated')); setSaveMessage('Profile saved. Existing logs keep their original timezone context.'); if (await dialogs.confirm('Your profile was saved. Would you like to review nutrition targets for these profile inputs? Accepted targets have not been changed.')) document.getElementById('nutrition-targets')?.scrollIntoView({behavior:'smooth'}); if (params.get('setup') === 'targets') navigate('/app/targets')
    } catch (cause) { setSaveError(cause instanceof Error ? cause.message : 'Could not save your profile.') }
    finally { setSaving(false) }
  }
  async function savePreferences(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!preferences) return
    setSaving(true); setSaveError(''); setSaveMessage('')
    try {
      const values = Object.fromEntries(new FormData(event.currentTarget))
      const updated = await patchAuthenticatedJson<typeof preferences>('/preferences', {
        version: preferences.version, daily_reset_time: values.daily_reset_time, unit_system: values.unit_system,
      })
      setPreferences(updated)
      await refreshTrackingDay()
      setSaveMessage('Preferences saved. Existing logs keep their original Tracking Day context.')
    } catch (cause) { setSaveError(cause instanceof Error ? cause.message : 'Could not save preferences.') }
    finally { setSaving(false) }
  }
  async function logout() {
    try { await postJson<void>('/auth/logout', {}) } finally { clearAccessToken(); navigate('/login', { replace: true }) }
  }
  async function linkGoogle() {
    setGoogleLinkError('')
    try {
      const result = await getAuthenticatedJson<{ authorization_url: string }>('/account/identities/google/link/start')
      window.location.assign(result.authorization_url)
    } catch (cause) { setGoogleLinkError(cause instanceof Error ? cause.message : 'Could not start Google account linking.') }
  }
  async function requestDeletion(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const password = String(new FormData(event.currentTarget).get('password') || '')
    if (!(await dialogs.confirm('Request account deletion? You will be signed out immediately. You can cancel within 30 days by verifying your sign-in method.'))) return
    setSaving(true); setSaveError(''); setSaveMessage('')
    try {
      const result = await postAuthenticatedJson<{ purge_after: string }>('/account/deletion-request', { password })
      clearAccessToken()
      navigate(`/cancel-deletion?requested=1&purge_after=${encodeURIComponent(result.purge_after)}`, { replace: true })
    } catch (cause) { setSaveError(cause instanceof Error ? cause.message : 'Could not request account deletion.') }
    finally { setSaving(false) }
  }
  async function requestGoogleDeletion() {
    if (!(await dialogs.confirm('Request account deletion? You will be signed out immediately after Google verifies your identity.'))) return
    setSaving(true); setSaveError('')
    try {
      const response = await getAuthenticatedJson<{ authorization_url: string }>('/account/deletion-request/google/start')
      window.location.assign(response.authorization_url)
    } catch (cause) { setSaveError(cause instanceof Error ? cause.message : 'Could not start Google verification.'); setSaving(false) }
  }
  if (!profile || !preferences || !trackingDay) return <><Link className="quiet-link guide-replay" to="/app/welcome?replay=1"><CircleHelp size={17} /> Open setup & feature guide</Link><PageHead title="Profile & settings" description="Your profile, timezone and preferences." />{saveError ? <p role="alert">{saveError}</p> : <Loading>Loading settings…</Loading>}</>
  return <>
    <Link className="quiet-link guide-replay" to="/app/welcome?replay=1"><CircleHelp size={17} /> Open setup & feature guide</Link>
    <PageHead eyebrow="Your account" title="Profile & settings" description="Choose how AteThat looks and set the timezone used for your account." action={<button className="button button-outline" type="button" onClick={logout}>Log out</button>} />
    <ProfileAttention fields={missingDetails} />
    <TargetSettings />
    <div className="settings-grid">

      <section className="panel settings-profile" id="profile-details">
        <div className="panel-top"><h2>Profile & timezone</h2><Clock3 size={19} /></div>
        <p className="muted">Timestamps are stored in UTC and shown using your chosen timezone. Highlighted fields below match the Settings badge. All these extra profile details are optional.</p>
        {profile ? <Form key={profile.version} className="auth-form" onSubmit={saveProfile}>
          <label>Display name<input name="display_name" required maxLength={80} defaultValue={profile.display_name} /></label>
          <div id="profile-field-date_of_birth" className={profileFieldClass('date_of_birth')}>
            <AppDatePicker className="form-date-picker" label="Date of birth" name="date_of_birth" value={profile.date_of_birth || ''} describedBy={missingField('date_of_birth') ? 'profile-field-date_of_birth-hint' : undefined} onChange={date_of_birth => setProfile(current => current ? { ...current, date_of_birth } : current)} />
            <ProfileFieldHint field={missingField('date_of_birth')} />
            <small>Used to check eligibility for adult nutrition benchmarks.</small>
          </div>
          <details className="secondary-details" open={missingDetails.some(field => field.key !== 'date_of_birth') || undefined}>
            <summary>Optional profile details{missingDetails.some(field => field.key !== 'date_of_birth') ? ' — highlighted details below' : ''}</summary>
            <div className="auth-form">
              <div className="recipe-meta">
                <label id="profile-field-height" className={profileFieldClass('height')}>Height
                  <input name="height" aria-label="Height" type="number" min="0.001" step="any" defaultValue={profile.height?.amount || ''} aria-describedby={missingField('height') ? 'profile-field-height-hint' : undefined} />
                  <ProfileFieldHint field={missingField('height')} />
                </label>
                <label>Height unit<select name="height_unit" defaultValue={profile.height?.unit || 'cm'}>{Array.from(new Set([profile.height?.unit || 'cm','cm','m','in'])).map(unit => <option key={unit}>{unit}</option>)}</select></label>
              </div>
              <div className="recipe-meta">
                <label id="profile-field-weight" className={profileFieldClass('weight')}>Profile weight
                  <input name="weight" aria-label="Profile weight" type="number" min="0.001" step="any" defaultValue={profile.weight?.amount || ''} aria-describedby={missingField('weight') ? 'profile-field-weight-hint' : undefined} />
                  <ProfileFieldHint field={missingField('weight')} />
                </label>
                <label>Weight unit<select name="weight_unit" defaultValue={profile.weight?.unit || 'kg'}>{Array.from(new Set([profile.weight?.unit || 'kg','kg','lb'])).map(unit => <option key={unit}>{unit}</option>)}</select></label>
              </div>
              <label id="profile-field-activity_level" className={profileFieldClass('activity_level')}>Activity level
                <select name="activity_level" aria-label="Activity level" defaultValue={profile.activity_level || ''} aria-describedby={missingField('activity_level') ? 'profile-field-activity_level-hint' : undefined}><option value="">Moderate (assumed default)</option><option value="SEDENTARY">Sedentary</option><option value="LIGHT">Light</option><option value="MODERATE">Moderate</option><option value="ACTIVE">Active</option><option value="VERY_ACTIVE">Very active</option></select>
                <ProfileFieldHint field={missingField('activity_level')} />
              </label>
              <label id="profile-field-sex" className={profileFieldClass('sex')}>Sex (optional)
                <select name="sex" aria-label="Sex (optional)" defaultValue={profile.sex || ''} aria-describedby={missingField('sex') ? 'profile-field-sex-hint' : undefined}><option value="">Not supplied</option><option value="FEMALE">Female</option><option value="MALE">Male</option><option value="DECLINED">Prefer not to provide</option></select>
                <ProfileFieldHint field={missingField('sex')} />
              </label>
              <p className="muted">If absent, planning uses 175 cm, 70.4375 kg and Moderate activity as assumptions, not measurements. Weight logs never change profile weight.</p>
              <label>Tracking goal<input name="goal" defaultValue={profile.goal || ''} maxLength={120} /></label>
              <small>Leave a measurement blank to keep its saved value. Calculated defaults use your saved profile details only after you review and confirm them in Nutrition targets.</small>
            </div>
          </details>
          <label>Default timezone<select name="timezone" defaultValue={profile.timezone}>{Array.from(new Set([profile.timezone, 'Asia/Kolkata', 'UTC', ...Intl.supportedValuesOf('timeZone')])).map(zone => <option key={zone} value={zone}>{zone}</option>)}</select></label>
          <button className="button button-primary" type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save profile'}</button>
        </Form> : <p className="muted">Loading profile…</p>}
        {saveError && <p className="form-message form-error" role="alert">{saveError}</p>}{saveMessage && <p className="form-message form-success" role="status">{saveMessage}</p>}
      </section>
      <section className="panel"><div className="panel-top"><h2>Appearance</h2><Sun size={19} /></div><p className="muted">Choose a theme or follow your device setting.</p><div className="theme-options">{(['light', 'dark', 'system'] as const).map(option => <button key={option} type="button" className={`theme-option ${theme === option ? 'theme-option-selected' : ''}`} onClick={() => setTheme(option)}><span>{option === 'light' ? <Sun size={20} /> : option === 'dark' ? <Moon size={20} /> : <Settings2 size={20} />}{option[0].toUpperCase() + option.slice(1)}</span>{theme === option && <Check size={18} />}</button>)}</div></section>
      <section className="panel"><div className="panel-top"><h2>Daily preferences</h2><CalendarDays size={19} /></div><p className="muted">Your nutrition day starts at the local reset time below. Changing it affects future records only.</p>{preferences ? <Form key={preferences.version} className="auth-form" onSubmit={savePreferences}><AppTimeInput label="Daily reset time" name="daily_reset_time" value={preferences.daily_reset_time} onChange={value=>setPreferences({...preferences,daily_reset_time:value})}/><label>Unit system<select name="unit_system" defaultValue={preferences.unit_system}><option value="METRIC">Metric</option><option value="IMPERIAL">Imperial</option></select></label><button className="button button-primary" type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save preferences'}</button></Form> : <p className="muted">Loading preferences…</p>}{trackingDay && <p className="tracking-day-note">Current Tracking Day: <strong>{trackingDay.tracking_day}</strong> · {trackingDay.timezone} · resets at {trackingDay.daily_reset_time}</p>}</section>
      {import.meta.env.VITE_GOOGLE_SIGNIN_ENABLED === 'true' && <section className="panel"><div className="panel-top"><h2>Sign-in methods</h2><LockKeyhole size={19} /></div><div className="sign-in-methods-content">{profile?.has_google_identity ? <><p className="muted">Your Google account is linked. You can use Google to sign in to this AteThat account.</p><p className="form-message form-success">Google account linked</p></> : <><p className="muted">Use the same Google email address as this AteThat account. A different Google email will be rejected.</p>{googleLinkError && <p className="form-message form-error" role="alert">{googleLinkError}</p>}{googleLinkMessage && <p className="form-message form-success" role="status">{googleLinkMessage}</p>}<button className="button button-outline" type="button" onClick={linkGoogle}><GoogleMark /> Link Google account</button></>}</div></section>}
      <section className="panel"><div className="panel-top"><h2>Delete account</h2><LockKeyhole size={19} /></div><p className="muted">Requesting deletion signs you out everywhere. Your records are permanently removed after 30 days unless you cancel with fresh verification.</p><Form className="auth-form" onSubmit={requestDeletion}><label>Current password<input name="password" type="password" autoComplete="current-password" required /></label><button className="button button-outline" type="submit" disabled={saving}>Request deletion</button></Form>{import.meta.env.VITE_GOOGLE_SIGNIN_ENABLED === 'true' && <button className="quiet-link" type="button" disabled={saving} onClick={() => void requestGoogleDeletion()}>Request deletion with Google verification</button>}</section>
    </div>
  </>
}

function CancelDeletionPage() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(params.get('error') === 'google' ? 'Google could not verify the cancellation. Please try again.' : '')
  const requested = params.get('requested') === '1'
  const purgeAfter = params.get('purge_after')
  async function cancel(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError('')
    const values = Object.fromEntries(new FormData(event.currentTarget))
    try {
      const session = await postJson<SessionResponse>('/auth/cancel-deletion', { email: values.email, password: values.password })
      setAccessToken(session.access_token)
      navigate('/app/settings?deletion_cancelled=password', { replace: true })
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not cancel deletion.') }
    finally { setBusy(false) }
  }
  async function cancelWithGoogle() {
    setBusy(true); setError('')
    try {
      const response = await getJson<{ authorization_url: string }>('/auth/google/cancel-deletion/start')
      window.location.assign(response.authorization_url)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not start Google verification.') ; setBusy(false) }
  }
  return <div className="auth-page"><PublicHeader /><main className="auth-main layout-width"><aside className="auth-intro"><span className="eyebrow"><span className="eyebrow-dot" /> Account deletion</span><h1 tabIndex={-1}>Your data is scheduled for removal.</h1><p>Deletion requests have a 30-day cancellation period. Fresh verification is required before access can be restored.</p></aside><section className="auth-card"><div className="auth-card-top"><span className="auth-icon"><LockKeyhole size={22} /></span><Link className="quiet-link" to="/login"><ArrowLeft size={15} /> Back to login</Link></div><h2>{requested ? 'Deletion request received' : 'Cancel account deletion'}</h2>{purgeAfter && <p className="muted">Scheduled permanent removal: {new Date(purgeAfter).toLocaleString()}.</p>}<p className="muted">Use the sign-in method that belongs to the pending account. Cancellation restores the account and signs you in.</p><Form className="auth-form" onSubmit={cancel}><label>Email address<input name="email" type="email" autoComplete="email" required /></label><label>Password<input name="password" type="password" autoComplete="current-password" required /></label>{error && <p className="form-message form-error" role="alert">{error}</p>}<button className="button button-primary auth-submit" type="submit" disabled={busy}>{busy ? 'Verifying…' : 'Cancel deletion and restore account'}</button></Form>{import.meta.env.VITE_GOOGLE_SIGNIN_ENABLED === 'true' && <><div className="divider"><span>or</span></div><button className="button button-google" type="button" disabled={busy} onClick={() => void cancelWithGoogle()}><GoogleMark /> Verify with Google</button></>}</section></main></div>
}

function NotFound() {
  return <div className="not-found"><Brand /><h1 tabIndex={-1}>Page not found</h1><p>That page isn’t part of AteThat.</p><Link className="button button-primary" to="/">Go home <ArrowRight size={16} /></Link></div>
}

export default function App() {
  return <ThemeProvider><DatePickerProvider><DialogProvider><FoodDraftProvider><RoutePosition /><Routes>
    <Route path="/" element={<RedirectIfAuthenticated><Landing /></RedirectIfAuthenticated>} />
    <Route path="/terms" element={<TermsPage />} />
    <Route path="/login" element={<RedirectIfAuthenticated><AuthPage mode="login" /></RedirectIfAuthenticated>} />
    <Route path="/signup" element={<RedirectIfAuthenticated><AuthPage mode="signup" /></RedirectIfAuthenticated>} />
    <Route path="/forgot-password" element={<AuthPage mode="forgot" />} />
    <Route path="/resend-verification" element={<AuthPage mode="resend" />} />
    <Route path="/verify-email" element={<TokenPage mode="verify" />} />
    <Route path="/reset-password" element={<TokenPage mode="reset" />} />
    <Route path="/email-login" element={<EmailLoginPage />} />
    <Route path="/cancel-deletion" element={<CancelDeletionPage />} />
    <Route path="/app" element={<AppShell><Dashboard /></AppShell>} />
    <Route path="/app/log-food" element={<AppShell><LogFood /></AppShell>} />
    <Route path="/app/history" element={<AppShell><HistoryPage /></AppShell>} />
    <Route path="/app/analysis" element={<AppShell><NutritionAnalysisPage /></AppShell>} />
    <Route path="/app/recipes" element={<AppShell><RecipesPage /></AppShell>} />
    <Route path="/app/wellness" element={<AppShell><WellnessPage /></AppShell>} />
    <Route path="/app/meals" element={<AppShell><CustomFoodsPage /></AppShell>} />
    <Route path="/app/health" element={<AppShell><HealthPage /></AppShell>} />
    <Route path="/app/target-calculations" element={<AppShell><TargetCalculations /></AppShell>} />
    <Route path="/app/targets" element={<AppShell><TargetsPage /></AppShell>} />
    <Route path="/app/welcome" element={<AppShell><Welcome /></AppShell>} />
    <Route path="/app/settings" element={<AppShell><Settings /></AppShell>} />
    <Route path="*" element={<NotFound />} />
  </Routes></FoodDraftProvider></DialogProvider></DatePickerProvider></ThemeProvider>
}
