import { createContext, useContext, useEffect, useId, useMemo, useState, type FormEvent, type ReactNode } from 'react'
import { Link, NavLink, Route, Routes, useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import {
  Activity, ArrowLeft, ArrowRight, CalendarDays, Check, ChevronDown, ChevronLeft, ChevronRight,
  CircleHelp, Clock3, HeartPulse, Home, Info, LockKeyhole, Menu,
  Moon, Plus, Search, Settings2, ShieldCheck, Sun, UtensilsCrossed, X,
} from 'lucide-react'
import { apiBaseUrl, deleteAuthenticatedJson, getAuthenticatedJson, getJson, patchAuthenticatedJson, postAuthenticatedIdempotentJson, postAuthenticatedJson, postJson, putAuthenticatedJson, refreshAccessToken } from './api'
import { clearAccessToken, getAccessToken, setAccessToken } from './session'
import { ThemeProvider, useTheme } from './theme'

type Icon = typeof Home

const navigation: { label: string; to: string; icon: Icon }[] = [
  { label: 'Dashboard', to: '/app', icon: Home },
  { label: 'Log food', to: '/app/log-food', icon: Plus },
  { label: 'History', to: '/app/history', icon: CalendarDays },
  { label: 'Analysis', to: '/app/analysis', icon: Activity },
  { label: 'Meals', to: '/app/meals', icon: UtensilsCrossed },
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
        <h1>Know what <em>nourishes</em> you.</h1>
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
    <div className="terms-heading"><span className="eyebrow">AteThat · Legal</span><h1>Terms and conditions</h1><p className="muted">Draft for local testing · Last updated 8 October 2026</p></div>
    <div className="terms-content">
      <section><h2>About AteThat</h2><p>AteThat helps you record food, nutrition, and health routines. Features shown in the current local version are still under development. Availability, accuracy, and these terms may change before a public launch.</p></section>
      <section><h2>Your account and information</h2><p>Keep your sign-in credentials secure and provide information you have the right to use. You control the records you add to your account. Nutrition information, including USDA-derived values, can be incomplete or vary by product, preparation, and serving size.</p></section>
      <section><h2>Health information</h2><p>AteThat is a tracking tool, not a medical professional or a substitute for medical advice, diagnosis, or treatment. Check important nutrition or medication decisions with a qualified professional.</p></section>
      <section><h2>Account deletion and permanent removal</h2><p>A verified account-deletion request starts a <strong>30-day deletion-pending period</strong> and ends account access immediately. You may cancel the request during that period only by freshly verifying your sign-in method. At the end of the 30 days, user-owned account data is scheduled for permanent removal, except information we must retain to meet applicable legal obligations. Removed data may not be recoverable.</p><p>The 30-day period does not start merely by stopping use of AteThat or deleting a browser session.</p></section>
      <section><h2>Service changes</h2><p>During development, the service may be interrupted, changed, or reset. Please do not rely on this local version as the only copy of important records. We will review and complete these terms, including operator and contact details, before public launch.</p></section>
    </div>
    <Link className="quiet-link" to="/"><ArrowLeft size={16} /> Back to home</Link>
  </main><PublicFooter /></div>
}

function Feature({ icon: IconComponent, title, text }: { icon: Icon; title: string; text: string }) {
  return <article className="feature-card"><span className="feature-icon"><IconComponent size={21} /></span><h3>{title}</h3><p>{text}</p></article>
}

type SessionResponse = { access_token: string }

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
    setBusy(true)
    const values = Object.fromEntries(new FormData(event.currentTarget))
    try {
      if (isLogin) {
        const session = await postJson<SessionResponse>('/auth/login', { email: values.email, password: values.password })
        setAccessToken(session.access_token)
        navigate('/app')
      } else if (isSignup) {
        await postJson<{ message: string }>('/auth/register', { display_name: values.display_name, email: values.email, password: values.password, timezone: values.timezone })
        setMessage('Check your email for a verification link before logging in.')
      } else if (mode === 'forgot') {
        await postJson<{ message: string }>('/auth/forgot-password', { email: values.email })
        setMessage('If an eligible account exists, password reset instructions have been sent.')
      } else {
        await postJson<{ message: string }>('/auth/resend-verification', { email: values.email })
        setMessage('If verification is still needed, a new email has been sent.')
      }
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Something went wrong. Please try again.')
    } finally { setBusy(false) }
  }

  return <div className="auth-page"><PublicHeader /><main className="auth-main layout-width">
    <aside className="auth-intro"><span className="eyebrow"><span className="eyebrow-dot" /> Your space, your pace</span><h1>Make room for a clearer daily picture.</h1><p>AteThat brings the essentials together in a calm, readable experience. Your data remains yours.</p><div className="auth-benefits"><div><UtensilsCrossed size={19} /> Food and nutrition in one place</div><div><Clock3 size={19} /> Timezone-aware daily view</div><div><LockKeyhole size={19} /> Private account access</div></div></aside>
    <section className="auth-card" aria-labelledby="auth-heading"><div className="auth-card-top"><span className="auth-icon"><LockKeyhole size={22} /></span><Link className="quiet-link" to="/"><ArrowLeft size={15} /> Back to home</Link></div>
      <h2 id="auth-heading">{isLogin ? 'Welcome back' : isSignup ? 'Create your account' : mode === 'forgot' ? 'Reset your password' : 'Resend verification'}</h2>
      <p className="muted">{isLogin ? 'Sign in to continue to your daily overview.' : isSignup ? 'Start with the basics. You can complete your profile later.' : mode === 'forgot' ? 'Enter your email and we’ll send reset instructions if your account is eligible.' : 'Enter the email you used to sign up.'}</p>
      {(isLogin || isSignup) && <button className="button button-google" type="button" disabled={import.meta.env.VITE_GOOGLE_SIGNIN_ENABLED !== 'true'} title={import.meta.env.VITE_GOOGLE_SIGNIN_ENABLED === 'true' ? 'Continue with Google' : 'Configure Google OAuth to enable'} onClick={() => { window.location.assign(`${apiBaseUrl}/auth/google/start?return_to=%2Fapp`) }}><GoogleMark /> Continue with Google</button>}
      {(isLogin || isSignup) && <div className="divider"><span>or continue with email</span></div>}
      <form className="auth-form" onSubmit={submit}>
        {isSignup && <label>Display name<input name="display_name" autoComplete="name" required maxLength={80} placeholder="Your name" /></label>}
        <label>Email address<input name="email" type="email" autoComplete="email" required placeholder="name@example.com" /></label>
        {(isLogin || isSignup) && <label>Password<span className="password-field"><input name="password" type={showPassword ? 'text' : 'password'} autoComplete={isSignup ? 'new-password' : 'current-password'} required minLength={isSignup ? 12 : 1} placeholder="Enter your password" /><button type="button" onClick={() => setShowPassword(!showPassword)} aria-label={showPassword ? 'Hide password' : 'Show password'}>{showPassword ? 'Hide' : 'Show'}</button></span></label>}
        {isSignup && <label>Default timezone<select name="timezone" defaultValue="Asia/Kolkata">{['Asia/Kolkata', 'UTC', ...Intl.supportedValuesOf('timeZone').filter(zone => zone !== 'Asia/Kolkata')].map(zone => <option key={zone} value={zone}>{zone}</option>)}</select><small>Change this manually in settings when you travel.</small></label>}
        {isLogin && <Link className="form-side-link" to="/forgot-password">Forgot password?</Link>}
        {error && <p className="form-message form-error" role="alert">{error}</p>}
        {message && <p className="form-message form-success" role="status">{message}</p>}
        <button className="button button-primary auth-submit" type="submit" disabled={busy}>{busy ? 'Please wait…' : isLogin ? 'Log in' : isSignup ? 'Create account' : mode === 'forgot' ? 'Send reset instructions' : 'Resend email'} <ArrowRight size={17} /></button>
      </form>
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
    setBusy(true); setError(''); setMessage('')
    try {
      if (mode === 'verify') {
        await postJson<{ email_verified: boolean }>('/auth/verify-email', { token })
        setMessage('Your email is verified. You can now log in.')
      } else {
        const values = Object.fromEntries(new FormData(event.currentTarget))
        await postJson<void>('/auth/reset-password', { token, new_password: values.new_password })
        clearAccessToken()
        setMessage('Your password has been changed. Please log in again.')
      }
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Please try again.') }
    finally { setBusy(false) }
  }
  return <div className="auth-page"><PublicHeader /><main className="auth-main layout-width"><aside className="auth-intro"><span className="eyebrow">Your account</span><h1>{mode === 'verify' ? 'One step to get started.' : 'A fresh start, securely.'}</h1><p>{mode === 'verify' ? 'Verify the email address you used for AteThat.' : 'Choose a new password for your account.'}</p></aside><section className="auth-card"><h2>{mode === 'verify' ? 'Verify your email' : 'Reset your password'}</h2><form className="auth-form" onSubmit={submit}>{mode === 'reset' && <label>New password<input name="new_password" type="password" autoComplete="new-password" required minLength={12} maxLength={128} /></label>}{error && <p className="form-message form-error" role="alert">{error}</p>}{message && <p className="form-message form-success" role="status">{message}</p>}<button className="button button-primary auth-submit" type="submit" disabled={busy || !token}>{busy ? 'Please wait…' : mode === 'verify' ? 'Verify email' : 'Change password'}</button></form><p className="auth-switch"><Link to="/login">Return to login</Link></p></section></main></div>
}

function AppShell({ children }: { children: ReactNode }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const [authenticated, setAuthenticated] = useState(!!getAccessToken())
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
    if (getAccessToken()) return
    let active = true
    refreshAccessToken().then(session => {
      if (active) { setAccessToken(session.access_token); setAuthenticated(true) }
    }).catch(() => { if (active) navigate('/login', { replace: true }) })
    return () => { active = false }
  }, [navigate])
  if (!authenticated) return <div className="not-found"><Brand /><p>Checking your session…</p></div>
  return <div className="app-shell"><aside className={`sidebar ${menuOpen ? 'sidebar-open' : ''}`}><div className="sidebar-head"><Brand /><button className="icon-button mobile-only" type="button" onClick={() => setMenuOpen(false)} aria-label="Close menu"><X size={20} /></button></div><nav className="sidebar-nav" aria-label="Main navigation">{navigation.map(({ label, to, icon: IconComponent }) => <NavLink onClick={() => setMenuOpen(false)} key={to} to={to} end={to === '/app'} className={({ isActive }) => `nav-item ${isActive ? 'nav-item-active' : ''}`}><IconComponent size={19} /><span>{label}</span></NavLink>)}</nav><div className="sidebar-foot"><span className="sidebar-foot-icon"><ShieldCheck size={18} /></span><p>Know what nourishes you.<small>Your daily view, at your pace.</small></p></div></aside>
    {menuOpen && <button className="mobile-scrim" type="button" aria-label="Close menu" onClick={() => setMenuOpen(false)} />}
    <div className="app-main"><header className="app-header"><div className="app-header-left"><button className="icon-button mobile-only" type="button" aria-label="Open menu" onClick={() => setMenuOpen(true)}><Menu size={21} /></button><span className="app-header-title">AteThat</span></div><div className="app-header-right"><span className="header-timezone"><Clock3 size={15} /> Time shown in your account timezone</span><ThemeButton /><Link className="button button-primary button-small header-log" to="/app/log-food"><Plus size={17} /> Log food</Link></div></header><main className="content"><div className="app-page-enter" key={location.pathname}>{children}</div></main></div>
    <nav className="bottom-nav" aria-label="Mobile navigation">{navigation.slice(0, 4).map(({ label, to, icon: IconComponent }) => <NavLink key={to} to={to} end={to === '/app'} className={({ isActive }) => isActive ? 'bottom-item bottom-item-active' : 'bottom-item'}><IconComponent size={20} /><span>{label}</span></NavLink>)}<button type="button" className="bottom-item" onClick={() => setMenuOpen(true)}><Menu size={20} /><span>More</span></button></nav>
  </div>
}

function PageHead({ eyebrow, title, description, action }: { eyebrow?: string; title: string; description: string; action?: ReactNode }) {
  return <div className="page-head"><div>{eyebrow && <span className="eyebrow">{eyebrow}</span>}<h1>{title}</h1><p>{description}</p></div>{action}</div>
}

function EmptyState({ icon: IconComponent, title, text, action }: { icon: Icon; title: string; text: string; action?: ReactNode }) {
  return <div className="empty-state"><span className="empty-icon"><IconComponent size={26} /></span><h2>{title}</h2><p>{text}</p>{action}</div>
}

type DashboardData = { tracking_day: string; timezone: string; daily_reset_time: string; nutrition: { totals: FoodResult['nutrients']; targets: FoodResult['nutrients']; progress: Record<string, { percent: string }> }; food_log_groups: { items: LoggedGroup[]; has_more: boolean } }
function Dashboard() {
  const [data, setData] = useState<DashboardData | null>(null)
  const [upcoming, setUpcoming] = useState<HealthHistoryEntry[]>([])
  const [error, setError] = useState('')
  useEffect(() => { getAuthenticatedJson<DashboardData>('/dashboard').then(setData)
    .catch(cause => setError(cause instanceof Error ? cause.message : 'Could not load your dashboard.'))
    const from = new Date()
    const to = new Date(from.getTime() + 24 * 60 * 60 * 1000)
    getAuthenticatedJson<{ items: HealthHistoryEntry[] }>(`/health-items/occurrences?${new URLSearchParams({ from: from.toISOString(), to: to.toISOString(), limit: '5' })}`)
      .then(page => setUpcoming(page.items)).catch(() => {}) }, [])
  const energy = data?.nutrition.totals.ENERGY
  return <><PageHead eyebrow="Your daily view" title="Today at a glance" description="Energy first, followed by your routine nutrients and confirmed food groups." action={<Link className="button button-primary" to="/app/log-food"><Plus size={17} /> Log food</Link>} />
    {error && <p className="form-message form-error" role="alert">{error}</p>}{data && <p className="muted">Tracking Day {data.tracking_day} · {data.timezone} · daily reset {data.daily_reset_time}</p>}
    <div className="dashboard-grid"><section className="panel energy-panel"><div className="panel-top"><span className="panel-kicker">Energy</span><span className="info-pill"><Info size={14} /> {data?.timezone || 'Account timezone'}</span></div><h2>{energy?.status === 'KNOWN' ? `${energy.amount} kcal` : 'No confirmed energy total yet'}</h2><p>{data?.nutrition.targets.ENERGY?.status === 'KNOWN' ? `Against your accepted general benchmark of ${data.nutrition.targets.ENERGY.amount} kcal. This is not a personalized calorie prescription.` : 'Accept a nutrition benchmark to see progress.'}</p><div className="empty-meter"><span style={{ width: `${Math.min(100, Math.max(0, Number(data?.nutrition.progress.ENERGY?.percent || 0)))}%` }} /></div><Link className="quiet-link" to="/app/log-food">Log food <ArrowRight size={16} /></Link></section>
      <section className="panel nutrient-panel"><div className="panel-top"><h2>Routine nutrients</h2><CircleHelp size={18} aria-hidden="true" /></div>{[['PROTEIN', 'Protein', 'protein'], ['CARBOHYDRATE', 'Carbohydrates', 'carbs'], ['FAT', 'Fat', 'fat'], ['FIBER', 'Fibre', 'fibre']].map(([key, label, color]) => <div className="nutrient-row" key={key}><div><span className={`nutrient-dot ${color}`} />{label}</div><span>{data?.nutrition.totals[key]?.status === 'KNOWN' ? `${data.nutrition.totals[key].amount} ${data.nutrition.totals[key].unit}` : 'unknown'}</span><div className="nutrient-track"><span style={{ width: `${Math.min(100, Math.max(0, Number(data?.nutrition.progress[key]?.percent || 0)))}%` }} /></div></div>)}<Link className="quiet-link" to="/app/targets">View nutrition targets <ArrowRight size={16} /></Link></section></div>
    <div className="dashboard-grid dashboard-secondary"><section className="panel"><div className="panel-top"><h2>Today’s food</h2><Link className="quiet-link" to="/app/history">View history <ArrowRight size={15} /></Link></div>{data?.food_log_groups.items.length ? data.food_log_groups.items.map(group => <div className="dashboard-group" key={group.id}><strong>{group.label || group.meal_category.toLowerCase()}</strong><span>{group.item_count} foods · {group.aggregate_final_nutrition.ENERGY?.status === 'KNOWN' ? `${group.aggregate_final_nutrition.ENERGY.amount} kcal` : 'energy unknown'}</span></div>) : <EmptyState icon={UtensilsCrossed} title="No food logged yet" text="You can add several foods together and review their totals before confirming." action={<Link className="button button-outline button-small" to="/app/log-food">Log your first meal</Link>} />}</section><section className="panel"><div className="panel-top"><h2>Health routine</h2><Link className="quiet-link" to="/app/health">View items <ArrowRight size={15} /></Link></div>{upcoming.length ? upcoming.map(item => <div className="dashboard-group" key={item.occurrence_id}><strong>{item.name}</strong><span>{item.scheduled_local.date} {item.scheduled_local.time} ({item.scheduled_local.timezone}) · {item.status.toLowerCase()}</span></div>) : <EmptyState icon={HeartPulse} title="No upcoming items" text="Scheduled medication and supplements for the next 24 hours appear here." />}</section></div>
  </>
}

type FoodResult = { food_reference: { type: 'USDA'; provider_food_id: string }; name: string; food_state: string; base_quantity: { amount: string; unit: string }; nutrients: Record<string, { status: string; amount?: string; unit: string }>; source_url: string }
type FoodPage = { items: FoodResult[]; page: { next_cursor: string | null; has_more: boolean } }
type CustomFood = { id: string; version: number; name: string; description: string; food_state: string; base_quantity: { amount: string; unit: string }; nutrients: FoodResult['nutrients'] }
type SavedMeal = { id: string; version: number; name: string; components: { position: number; food_reference: { type: 'USDA' | 'CUSTOM'; provider_food_id: string | null; custom_food_id: string | null }; quantity: { amount: string; unit: string }; food_state: string }[] }
type SavedFoodLink = { food_key: string; food_reference: { type: 'USDA' | 'CUSTOM'; provider_food_id: string | null; custom_food_id: string | null }; name: string; confirmed_log_count?: number }
type SavedServing = { id: string; version: number; label: string; food_reference: { type: 'USDA' | 'CUSTOM'; provider_food_id: string | null; custom_food_id: string | null }; quantity: { amount: string; unit: string }; food_state: string }
type DraftFood = { key: string; type: 'USDA' | 'CUSTOM'; id: string; name: string; foodState: string; quantity: string; unit: string; overrides: Record<string, { status: string; amount: string; unit: string }> }
type LoggedGroup = { id: string | null; tracking_day: string; item_count: number; meal_category: string; label: string; aggregate_final_nutrition: FoodResult['nutrients']; items: { id: string | null; position: number; food_source: { name: string }; quantity: { amount: string; unit: string }; final_nutrition: FoodResult['nutrients'] }[] }
type GroupPreview = { preview_token: string; expires_at: string; group: LoggedGroup }

type DatePickerContextValue = { openPickerId: string | null; setOpenPickerId: (id: string | null) => void }
const DatePickerContext = createContext<DatePickerContextValue | null>(null)
function DatePickerProvider({ children }: { children: ReactNode }) {
  const [openPickerId, setOpenPickerId] = useState<string | null>(null)
  return <DatePickerContext.Provider value={{ openPickerId, setOpenPickerId }}>{children}</DatePickerContext.Provider>
}

function parseLocalDate(value: string) {
  const [year, month, day] = value.split('-').map(Number)
  return new Date(year, month - 1, day)
}

function isoLocalDate(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function AppDatePicker({ label, value, name, onChange, className = '' }: { label: string; value: string; name?: string; onChange: (value: string) => void; className?: string }) {
  const context = useContext(DatePickerContext)
  if (!context) throw new Error('AppDatePicker must be rendered inside DatePickerProvider.')
  const generatedId = useId()
  const pickerId = `date-${generatedId}`
  const open = context.openPickerId === pickerId
  const [visibleMonth, setVisibleMonth] = useState(() => parseLocalDate(value || isoLocalDate(new Date())))
  const selected = value ? parseLocalDate(value) : null
  const days = useMemo(() => {
    const first = new Date(visibleMonth.getFullYear(), visibleMonth.getMonth(), 1)
    const dayCount = new Date(visibleMonth.getFullYear(), visibleMonth.getMonth() + 1, 0).getDate()
    return Array.from({ length: first.getDay() + dayCount }, (_, index) => index < first.getDay() ? null : new Date(visibleMonth.getFullYear(), visibleMonth.getMonth(), index - first.getDay() + 1))
  }, [visibleMonth])
  useEffect(() => {
    if (!open) return
    const closeOnEscape = (event: KeyboardEvent) => { if (event.key === 'Escape') context.setOpenPickerId(null) }
    window.addEventListener('keydown', closeOnEscape)
    return () => window.removeEventListener('keydown', closeOnEscape)
  }, [context, open])
  const displayValue = selected?.toLocaleDateString(undefined, { day: '2-digit', month: 'short', year: 'numeric' }) || 'Select date'
  return <div className={`app-date-picker ${className}`}><span className="app-date-label">{label}</span>{name && <input type="hidden" name={name} value={value} />}<button className="app-date-trigger" type="button" aria-haspopup="dialog" aria-expanded={open} onClick={() => { setVisibleMonth(selected || new Date()); context.setOpenPickerId(open ? null : pickerId) }}><CalendarDays size={17} /><span>{displayValue}</span><ChevronDown size={16} /></button>{open && <div className="app-calendar" role="dialog" aria-label={`${label} date picker`}><div className="app-calendar-head"><button type="button" className="calendar-nav" aria-label="Previous month" onClick={() => setVisibleMonth(previous => new Date(previous.getFullYear(), previous.getMonth() - 1, 1))}><ChevronLeft size={17} /></button><strong>{visibleMonth.toLocaleDateString(undefined, { month: 'long', year: 'numeric' })}</strong><button type="button" className="calendar-nav" aria-label="Next month" onClick={() => setVisibleMonth(previous => new Date(previous.getFullYear(), previous.getMonth() + 1, 1))}><ChevronRight size={17} /></button></div><div className="calendar-weekdays">{['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'].map(day => <span key={day}>{day}</span>)}</div><div className="calendar-days">{days.map((date, index) => date ? <button type="button" key={isoLocalDate(date)} className={`calendar-day ${isoLocalDate(date) === value ? 'calendar-day-selected' : ''} ${isoLocalDate(date) === isoLocalDate(new Date()) ? 'calendar-day-today' : ''}`} onClick={() => { onChange(isoLocalDate(date)); context.setOpenPickerId(null) }}>{date.getDate()}</button> : <span key={`blank-${index}`} aria-hidden="true" />)}</div></div>}</div>
}

function LogFood() {
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
  const [draft, setDraft] = useState<DraftFood[]>([])
  const [category, setCategory] = useState('BREAKFAST')
  const [label, setLabel] = useState('')
  const [preview, setPreview] = useState<GroupPreview | null>(null)
  const [confirmationKey, setConfirmationKey] = useState('')
  const [confirmed, setConfirmed] = useState<LoggedGroup | null>(null)
  const [cursor, setCursor] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [searched, setSearched] = useState(false)
  useEffect(() => {
    if (source !== 'CUSTOM') return
    let active = true
    getAuthenticatedJson<{ items: CustomFood[] }>('/foods/custom?limit=50').then(page => { if (active) setCustomFoods(page.items) })
      .catch(cause => { if (active) setError(cause instanceof Error ? cause.message : 'Could not load your foods.') })
    return () => { active = false }
  }, [source])
  useEffect(() => {
    if (source !== 'MEAL') return
    let active = true
    getAuthenticatedJson<{ items: SavedMeal[] }>('/meals?limit=50').then(page => { if (active) setMeals(page.items) })
      .catch(cause => { if (active) setError(cause instanceof Error ? cause.message : 'Could not load saved meals.') })
    return () => { active = false }
  }, [source])
  useEffect(() => {
    if (!savedSource) return
    let active = true
    const path = savedSource === 'FAVOURITES' ? '/foods/favourites' : savedSource === 'RECENT' ? '/foods/recent' : '/foods/frequent'
    getAuthenticatedJson<{ items: SavedFoodLink[]; page: { next_cursor: string | null } }>(`${path}?limit=20`)
      .then(page => { if (active) { setSavedLinks(page.items); setSavedCursor(page.page.next_cursor) } })
      .catch(cause => { if (active) setError(cause instanceof Error ? cause.message : 'Could not load foods.') })
    return () => { active = false }
  }, [savedSource])
  useEffect(() => {
    if (!showServings) return
    let active = true
    getAuthenticatedJson<{ items: SavedServing[]; page: { next_cursor: string | null } }>('/servings?limit=20')
      .then(page => { if (active) { setServings(page.items); setServingCursor(page.page.next_cursor) } })
      .catch(cause => { if (active) setError(cause instanceof Error ? cause.message : 'Could not load servings.') })
    return () => { active = false }
  }, [showServings])
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
    const custom = 'id' in food
    setDraft(previous => [...previous, { key: crypto.randomUUID(), type: custom ? 'CUSTOM' : 'USDA',
      id: custom ? food.id : food.food_reference.provider_food_id, name: food.name,
      foodState: servingState || food.food_state, quantity: servingAmount || food.base_quantity.amount, unit: food.base_quantity.unit, overrides: {} }])
    setPreview(null); setConfirmed(null)
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
    const label = window.prompt(`Name this serving of ${food.name}`, `1 serving of ${food.name}`)?.trim()
    if (!label) return
    setBusy(true); setError('')
    try {
      const item = await postAuthenticatedJson<SavedServing>('/servings', { label,
        food_reference: food.type === 'USDA' ? { type: 'USDA', provider_food_id: food.id } : { type: 'CUSTOM', custom_food_id: food.id },
        quantity: { amount: food.quantity, unit: food.unit }, food_state: food.foodState })
      setShowServings(true); setServings(previous => [item, ...previous])
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
    const label = window.prompt('Serving label', serving.label)?.trim()
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
    if (!window.confirm(`Delete serving ${serving.label}? Confirmed logs remain unchanged.`)) return
    setBusy(true); setError('')
    try { await deleteAuthenticatedJson(`/servings/${serving.id}`); setServings(previous => previous.filter(item => item.id !== serving.id)) }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not delete serving.') }
    finally { setBusy(false) }
  }
  function editDraft(key: string, quantity: string) {
    setDraft(previous => previous.map(item => item.key === key ? { ...item, quantity } : item))
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
    setBusy(true); setError('')
    try {
      const params = new URLSearchParams({ q: query.trim(), limit: '20' })
      if (state) params.set('food_state', state)
      if (nextCursor) params.set('cursor', nextCursor)
      const response = await getAuthenticatedJson<FoodPage>(`/foods/search?${params}`)
      setItems(previous => nextCursor ? [...previous, ...response.items] : response.items)
      setCursor(response.page.next_cursor)
      setSearched(true)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Food search is unavailable right now.') }
    finally { setBusy(false) }
  }
  async function review() {
    setBusy(true); setError(''); setConfirmed(null)
    try {
      const response = await postAuthenticatedJson<GroupPreview>('/food-log-groups/preview', {
        logged_at: new Date().toISOString(), meal_category: category, label,
        items: draft.map((item, position) => ({ position,
          food_reference: item.type === 'USDA' ? { type: 'USDA', provider_food_id: item.id } : { type: 'CUSTOM', custom_food_id: item.id },
          quantity: { amount: item.quantity, unit: item.unit }, food_state: item.foodState, user_override_nutrition: item.overrides || {} })),
      })
      setPreview(response); setConfirmationKey(crypto.randomUUID())
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not preview the group.') }
    finally { setBusy(false) }
  }
  async function saveMeal() {
    const name = window.prompt('Name this reusable meal')?.trim()
    if (!name) return
    setBusy(true); setError('')
    try {
      await postAuthenticatedJson('/meals', { name, components: draft.map((item, position) => ({ position,
        food_reference: item.type === 'USDA' ? { type: 'USDA', provider_food_id: item.id } : { type: 'CUSTOM', custom_food_id: item.id },
        quantity: { amount: item.quantity, unit: item.unit }, food_state: item.foodState })) })
      setSource('MEAL')
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not save meal.') }
    finally { setBusy(false) }
  }
  async function reviewMeal(meal: SavedMeal) {
    setBusy(true); setError(''); setConfirmed(null)
    try {
      const response = await postAuthenticatedJson<GroupPreview>(`/meals/${meal.id}/preview`, {
        logged_at: new Date().toISOString(), meal_category: category, label: label || meal.name })
      setPreview(response); setConfirmationKey(crypto.randomUUID()); setDraft([])
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not preview saved meal.') }
    finally { setBusy(false) }
  }
  async function confirm() {
    if (!preview) return
    setBusy(true); setError('')
    try {
      const group = await postAuthenticatedIdempotentJson<LoggedGroup>('/food-log-groups/confirm', { preview_token: preview.preview_token }, confirmationKey)
      setConfirmed(group); setPreview(null); setDraft([])
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not confirm the group.') }
    finally { setBusy(false) }
  }
  return <><PageHead eyebrow="Food logging" title="Build your food log" description="Add one or several foods, review the totals together, then explicitly confirm the group." />
    <section className="panel section-panel"><div className="panel-top"><h2>Quick picks</h2><span className="muted">Your own favourites and confirmed history</span></div><div className="filter-row">{(['FAVOURITES', 'RECENT', 'FREQUENT'] as const).map(value => <button key={value} type="button" className={`filter-chip ${savedSource === value ? 'filter-chip-active' : ''}`} onClick={() => setSavedSource(value)}>{value === 'FAVOURITES' ? 'Favourites' : value === 'RECENT' ? 'Recent' : 'Frequent'}</button>)}</div>{savedSource && savedLinks.length === 0 && <p className="muted">No {savedSource.toLowerCase()} foods yet.</p>}{savedSource && savedLinks.map(link => <article className="food-result" key={link.food_key}><div><h3>{link.name}</h3><p>{link.food_reference.type}{link.confirmed_log_count ? ` · logged ${link.confirmed_log_count} times` : ''}</p></div><div className="history-actions"><button className="button button-outline button-small" type="button" disabled={busy} onClick={() => void addSavedLink(link)}>Add</button>{savedSource === 'FAVOURITES' && <button className="quiet-link" type="button" disabled={busy} onClick={() => void removeFavourite(link)}>Unsave</button>}</div></article>)}{savedCursor && <button className="button button-outline button-small" type="button" disabled={busy} onClick={() => void moreSavedLinks()}>Load more</button>}{draft.length > 0 && <div className="filter-row">{draft.map(food => <button className="quiet-link" type="button" key={food.key} disabled={busy} onClick={() => void favourite(food)}>Save {food.name} as favourite</button>)}</div>}</section>
    <section className="panel section-panel"><div className="panel-top"><h2>Named servings</h2><button className="quiet-link" type="button" onClick={() => setShowServings(value => !value)}>{showServings ? 'Hide' : 'Show'} servings</button></div><p className="muted">Save a reusable quantity such as “1 roti = 45 g”. It fills the amount in your draft, then you still review and confirm the food log.</p>{showServings && servings.length === 0 && <p className="muted">No servings saved yet.</p>}{showServings && servings.map(serving => <article className="food-result" key={serving.id}><div><h3>{serving.label}</h3><p>{serving.quantity.amount} {serving.quantity.unit} · {serving.food_reference.type}</p></div><div className="history-actions"><button className="button button-outline button-small" type="button" disabled={busy} onClick={() => void addServing(serving)}>Add</button><button className="quiet-link" type="button" disabled={busy} onClick={() => void renameServing(serving)}>Rename</button><button className="quiet-link" type="button" disabled={busy} onClick={() => void deleteServing(serving)}>Delete</button></div></article>)}{showServings && servingCursor && <button className="button button-outline button-small" type="button" disabled={busy} onClick={() => void moreServings()}>Load more servings</button>}{draft.map(food => <button className="quiet-link" type="button" key={food.key} disabled={busy} onClick={() => void saveServing(food)}>Save {food.quantity} {food.unit} of {food.name} as serving</button>)}</section>
    {draft.length > 0 && <section className="panel section-panel"><h2>Adjust nutrition values (optional)</h2><p className="muted">Enter the value for the exact quantity in your draft. Your value replaces the source value for that nutrient in this log only. Leave a field blank to use the source value.</p>{draft.map(food => <div className="group-review" key={food.key}><h3>{food.name} · {food.quantity} {food.unit}</h3><div className="filter-row">{[['ENERGY', 'kcal'], ['PROTEIN', 'g'], ['CARBOHYDRATE', 'g'], ['FAT', 'g'], ['FIBER', 'g']].map(([key, unit]) => <label key={key}>{nutrientLabel(key)} ({unit})<input type="number" min="0" step="any" value={food.overrides?.[key]?.amount || ''} onChange={event => setOverride(food.key, key, event.target.value, unit)} placeholder="Use source" /></label>)}</div></div>)}</section>}
    <div className="two-column"><section className="panel"><div className="filter-row"><button className={`filter-chip ${source === 'USDA' ? 'filter-chip-active' : ''}`} type="button" onClick={() => setSource('USDA')}>USDA foods</button><button className={`filter-chip ${source === 'CUSTOM' ? 'filter-chip-active' : ''}`} type="button" onClick={() => setSource('CUSTOM')}>My foods</button><button className={`filter-chip ${source === 'MEAL' ? 'filter-chip-active' : ''}`} type="button" onClick={() => setSource('MEAL')}>Saved meals</button></div>{source === 'USDA' ? <><form className="food-search-form" onSubmit={event => search(event)}><label className="search-field"><Search size={19} /><input type="search" placeholder="Search USDA foods" aria-label="Search USDA foods" value={query} onChange={event => setQuery(event.target.value)} minLength={2} required /></label><select aria-label="Food state" value={state} onChange={event => setState(event.target.value)}><option value="">Any state</option><option value="RAW">Raw</option><option value="COOKED">Cooked</option></select><button className="button button-primary button-small" type="submit" disabled={busy}>{busy ? 'Searching…' : 'Search'}</button></form><p className="muted">USDA FoodData Central values are per 100 g. Missing values are unknown, not zero.</p>{searched && items.length === 0 && <EmptyState icon={Search} title="No foods found" text="Try a broader search or a different food state." />}{items.length > 0 && <div className="food-results">{items.map(item => <article className="food-result" key={item.food_reference.provider_food_id}><div><h3>{item.name}</h3><p>USDA · {item.base_quantity.amount} {item.base_quantity.unit} · {item.food_state.toLowerCase()}</p><p>{['ENERGY', 'PROTEIN', 'CARBOHYDRATE', 'FAT', 'FIBER'].map(key => `${key.toLowerCase()}: ${item.nutrients[key]?.status === 'KNOWN' ? `${item.nutrients[key].amount} ${item.nutrients[key].unit}` : 'unknown'}`).join(' · ')}</p><a className="quiet-link" href={item.source_url} target="_blank" rel="noreferrer">USDA source <ArrowRight size={15} /></a></div><button className="button button-outline button-small" type="button" onClick={() => addFood(item)}>Add</button></article>)}{cursor && <button className="button button-outline button-small" type="button" disabled={busy} onClick={() => search(undefined, cursor)}>Load more foods</button>}</div>}</> : source === 'CUSTOM' ? <div className="food-results">{customFoods.length === 0 && <EmptyState icon={UtensilsCrossed} title="No custom foods" text="Create one in Meals & foods, then come back to add it." />}{customFoods.map(food => <article className="food-result" key={food.id}><div><h3>{food.name}</h3><p>{food.base_quantity.amount} {food.base_quantity.unit} · {food.food_state.toLowerCase()}</p></div><button className="button button-outline button-small" type="button" onClick={() => addFood(food)}>Add</button></article>)}</div> : <div className="food-results">{meals.length === 0 && <EmptyState icon={UtensilsCrossed} title="No saved meals" text="Add foods to a draft, then save it as a reusable meal." />}{meals.map(meal => <article className="food-result" key={meal.id}><div><h3>{meal.name}</h3><p>{meal.components.length} foods</p></div><button className="button button-outline button-small" type="button" disabled={busy} onClick={() => reviewMeal(meal)}>Review & log</button></article>)}</div>}</section><aside className="panel tray-panel"><div className="panel-top"><h2>Your food group</h2><span className="info-pill">{draft.length} items</span></div>{draft.length === 0 && !confirmed && !preview && <EmptyState icon={UtensilsCrossed} title="Your group is empty" text="Add foods or choose a saved meal." />}{draft.map(item => <div className="draft-food" key={item.key}><strong>{item.name}</strong><div><label>Quantity <input type="number" min="0.001" step="any" value={item.quantity} onChange={event => editDraft(item.key, event.target.value)} /> {item.unit}</label><button className="icon-button" type="button" aria-label={`Remove ${item.name}`} onClick={() => { setDraft(previous => previous.filter(food => food.key !== item.key)); setPreview(null) }}><X size={17} /></button></div></div>)}{draft.length > 0 && <><label className="draft-meta">Meal category<select value={category} onChange={event => { setCategory(event.target.value); setPreview(null) }}><option>BREAKFAST</option><option>LUNCH</option><option>DINNER</option><option>SNACK</option><option>OTHER</option></select></label><label className="draft-meta">Label (optional)<input maxLength={160} value={label} onChange={event => { setLabel(event.target.value); setPreview(null) }} /></label><button className="button button-primary full-width" type="button" disabled={busy || draft.length > 50} onClick={review}>Review food group</button><button className="button button-outline full-width" type="button" disabled={busy} onClick={saveMeal}>Save as reusable meal</button></>}{preview && <div className="group-review"><h3>Review before saving</h3><p>{preview.group.item_count} foods · Tracking Day {preview.group.tracking_day}</p>{preview.group.items.map(item => <p key={item.position}>{item.food_source.name}: {item.quantity.amount} {item.quantity.unit}</p>)}<h4>Combined nutrition</h4>{Object.entries(preview.group.aggregate_final_nutrition).map(([key, value]) => <p key={key}>{key}: {value.status === 'KNOWN' ? `${value.amount} ${value.unit}` : 'unknown'}</p>)}<p className="muted">Preview expires at {new Date(preview.expires_at).toLocaleTimeString()}.</p><button className="button button-primary full-width" type="button" disabled={busy} onClick={confirm}>Confirm and save</button></div>}{confirmed && <p className="form-message form-success" role="status">Saved {confirmed.item_count} foods for {confirmed.tracking_day}. <Link to="/app/history">View history</Link></p>}{error && <p className="form-message form-error" role="alert">{error}</p>}</aside></div>
  </>
}

type HistoryPageResponse = { items: LoggedGroup[]; page: { next_cursor: string | null; has_more: boolean } }
type NutritionDay = { tracking_day: string; timezone: string; daily_reset_time: string; totals: FoodResult['nutrients']; confirmed_group_count: number; confirmed_item_count: number; detail_compacted: boolean }

function HistoryPage() {
  const [queryParams] = useSearchParams()
  const [day, setDay] = useState('')
  const [groups, setGroups] = useState<LoggedGroup[]>([])
  const [summary, setSummary] = useState<NutritionDay | null>(null)
  const [cursor, setCursor] = useState<string | null>(null)
  const [expanded, setExpanded] = useState<Record<string, LoggedGroup>>({})
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
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
    let active = true
    const params = new URLSearchParams({ tracking_day: day, limit: '20' })
    Promise.all([getAuthenticatedJson<HistoryPageResponse>(`/food-log-groups?${params}`),
      getAuthenticatedJson<{ items: NutritionDay[] }>(`/history/nutrition?from_tracking_day=${day}&to_tracking_day=${day}&limit=1`)]).then(([response, daily]) => {
      if (active) { setGroups(response.items); setCursor(response.page.next_cursor); setSummary(daily.items[0] || null); setExpanded({}) }
    }).catch(cause => { if (active) setError(cause instanceof Error ? cause.message : 'Could not load food history.') })
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
    if (!window.confirm('Remove this entire confirmed food group from your visible history and totals?')) return
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
    const amount = window.prompt(`Correct quantity for ${item.food_source.name} (${item.quantity.unit})`, item.quantity.amount)?.trim()
    if (!amount) return
    const reason = window.prompt('Reason for correction')?.trim()
    if (!reason) return
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
  return <><PageHead eyebrow="Your records" title="Food history" description="Confirmed food groups are shown by Tracking Day. Removing a group takes it out of normal totals." />
    <section className="panel section-panel"><AppDatePicker className="history-date" label="Tracking Day" value={day} onChange={setDay} />{error && <p className="form-message form-error" role="alert">{error}</p>}{summary && <p className="muted">Daily total: {summary.totals.ENERGY?.status === 'KNOWN' ? `${summary.totals.ENERGY.amount} kcal` : 'energy unknown'} · {summary.confirmed_group_count} groups · {summary.confirmed_item_count} foods · {summary.timezone}{summary.detail_compacted ? ' · Older item details compacted after 90 days' : ''}</p>}{groups.length === 0 && !summary && <EmptyState icon={CalendarDays} title="No food logged for this day" text="Confirmed groups will appear here." />}{groups.map(group => <article className="history-group" key={group.id}><div><h2>{group.label || group.meal_category.toLowerCase()}</h2><p>{group.item_count} foods · {group.tracking_day}</p><p>Energy: {group.aggregate_final_nutrition.ENERGY?.status === 'KNOWN' ? `${group.aggregate_final_nutrition.ENERGY.amount} kcal` : 'unknown'}</p></div><div className="history-actions"><button className="button button-outline button-small" type="button" onClick={() => group.id && expand(group.id)}>{group.id && expanded[group.id] ? 'Hide details' : 'Show details'}</button><button className="quiet-link" type="button" disabled={busy} onClick={() => group.id && remove(group.id)}>Remove group</button></div>{group.id && expanded[group.id] && <div className="history-details">{expanded[group.id].items.map(item => <div className="history-item" key={item.position}><p>{item.food_source.name} · {item.quantity.amount} {item.quantity.unit} · energy {item.final_nutrition.ENERGY?.status === 'KNOWN' ? `${item.final_nutrition.ENERGY.amount} kcal` : 'unknown'}</p><button className="quiet-link" type="button" disabled={busy} onClick={() => correct(item)}>Correct amount</button></div>)}</div>}</article>)}{cursor && <button className="button button-outline button-small" type="button" disabled={busy} onClick={more}>Load more</button>}{correction && <div className="group-review"><h2>Review correction</h2><p>This replaces the entire food group after confirmation. The old revision remains as an audit record.</p><p>New energy: {correction.preview.group.aggregate_final_nutrition.ENERGY?.status === 'KNOWN' ? `${correction.preview.group.aggregate_final_nutrition.ENERGY.amount} kcal` : 'unknown'}</p><button className="button button-primary" type="button" disabled={busy} onClick={confirmCorrection}>Confirm correction</button><button className="quiet-link" type="button" onClick={() => setCorrection(null)}>Cancel</button></div>}</section>
  </>
}

type FullAnalysis = { from: string; to: string; totals: FoodResult['nutrients']; confirmed_group_count: number; confirmed_item_count: number; compacted_day_count: number }
type HealthHistoryEntry = { occurrence_id: string; name: string; type: string; status: string; scheduled_at: string; scheduled_local: { date: string; time: string; timezone: string } }

function AnalysisPage() {
  const today = new Date().toISOString().slice(0, 10)
  const weekAgo = new Date(Date.now() - 7 * 86400000).toISOString().slice(0, 10)
  const [fromDay, setFromDay] = useState(weekAgo)
  const [toDay, setToDay] = useState(today)
  const [days, setDays] = useState<NutritionDay[]>([])
  const [dayCursor, setDayCursor] = useState<string | null>(null)
  const [health, setHealth] = useState<HealthHistoryEntry[]>([])
  const [healthCursor, setHealthCursor] = useState<string | null>(null)
  const [analysis, setAnalysis] = useState<FullAnalysis | null>(null)
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function load() {
    if (fromDay > toDay || (Date.parse(`${toDay}T00:00:00Z`) - Date.parse(`${fromDay}T00:00:00Z`)) > 365 * 86400000) {
      setError('Choose a date range of at most 365 days.'); return
    }
    setBusy(true); setError('')
    setLoaded(false); setAnalysis(null); setDays([]); setHealth([]); setDayCursor(null); setHealthCursor(null)
    try {
      const from = `${fromDay}T00:00:00Z`
      const to = new Date(Date.parse(`${toDay}T00:00:00Z`) + 86400000).toISOString()
      const dayParams = new URLSearchParams({ from_tracking_day: fromDay, to_tracking_day: toDay, limit: '30' })
      const healthParams = new URLSearchParams({ from, to, limit: '30' })
      const [daily, actionPage] = await Promise.all([
        getAuthenticatedJson<{ items: NutritionDay[]; page: { next_cursor: string | null } }>(`/history/nutrition?${dayParams}`),
        getAuthenticatedJson<{ items: HealthHistoryEntry[]; page: { next_cursor: string | null } }>(`/history/health-items?${healthParams}`),
      ])
      setDays(daily.items)
      setDayCursor(daily.page.next_cursor)
      setHealth(actionPage.items)
      setHealthCursor(actionPage.page.next_cursor)
      setLoaded(true)
      try {
        setAnalysis(await getAuthenticatedJson<FullAnalysis>(`/nutrition/full-analysis?${new URLSearchParams({ from, to })}`))
      } catch (cause) {
        setAnalysis(null)
        setError(cause instanceof Error ? `${cause.message} Daily summaries remain available below.` : 'UTC range total is unavailable; daily summaries remain available below.')
      }
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not load analysis.') }
    finally { setBusy(false) }
  }
  async function moreDays() {
    if (!dayCursor) return
    setBusy(true); setError('')
    try {
      const query = new URLSearchParams({ from_tracking_day: fromDay, to_tracking_day: toDay, limit: '30', cursor: dayCursor })
      const page = await getAuthenticatedJson<{ items: NutritionDay[]; page: { next_cursor: string | null } }>(`/history/nutrition?${query}`)
      setDays(previous => [...previous, ...page.items]); setDayCursor(page.page.next_cursor)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not load more days.') }
    finally { setBusy(false) }
  }
  async function moreHealth() {
    if (!healthCursor) return
    setBusy(true); setError('')
    try {
      const from = `${fromDay}T00:00:00Z`
      const to = new Date(Date.parse(`${toDay}T00:00:00Z`) + 86400000).toISOString()
      const query = new URLSearchParams({ from, to, limit: '30', cursor: healthCursor })
      const page = await getAuthenticatedJson<{ items: HealthHistoryEntry[]; page: { next_cursor: string | null } }>(`/history/health-items?${query}`)
      setHealth(previous => [...previous, ...page.items]); setHealthCursor(page.page.next_cursor)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not load more actions.') }
    finally { setBusy(false) }
  }
  return <><PageHead eyebrow="Your patterns" title="Analysis" description="Compare daily food totals and review recorded health actions. Unknown nutrient values stay unknown." />
    <section className="panel analysis-panel"><form className="analysis-filter" onSubmit={event => { event.preventDefault(); void load() }}><AppDatePicker label="From" value={fromDay} onChange={setFromDay} /><AppDatePicker label="Through" value={toDay} onChange={setToDay} /><button className="button button-primary" type="submit" disabled={busy}>Show analysis</button></form>{error && <p className="form-message form-error" role="alert">{error}</p>}{analysis && <><h2>Nutrition within UTC calendar dates</h2><p className="muted">{analysis.confirmed_group_count} confirmed groups · {analysis.confirmed_item_count} foods{analysis.compacted_day_count > 0 ? ` · ${analysis.compacted_day_count} compacted days` : ''}. This total uses UTC midnight boundaries. The daily rows below use your stored Tracking Days, so boundary foods may differ.</p><div className="target-rows">{Object.entries(analysis.totals).map(([key, value]) => <div key={key}><span>{nutrientLabel(key)}</span><strong>{value.status === 'KNOWN' ? `${value.amount} ${value.unit}` : 'Unknown'}</strong></div>)}</div></>}
    </section><div className="two-column"><section className="panel"><h2>Daily nutrition by Tracking Day</h2>{loaded && days.length === 0 && <p className="muted">No confirmed food for this range.</p>}{days.map(day => <article className="food-result" key={day.tracking_day}><div><h3>{day.tracking_day}</h3><p>{day.confirmed_item_count} foods · {day.totals.ENERGY?.status === 'KNOWN' ? `${day.totals.ENERGY.amount} kcal` : 'energy unknown'} · {day.timezone}</p>{day.detail_compacted && <p>Detailed food records were compacted after 90 days.</p>}</div><Link className="quiet-link" to={`/app/history?tracking_day=${day.tracking_day}`}>View day</Link></article>)}{dayCursor && <button className="button button-outline button-small" type="button" disabled={busy} onClick={() => void moreDays()}>Load more</button>}</section><section className="panel"><h2>Health action history</h2>{loaded && health.length === 0 && <p className="muted">No recorded actions for this range.</p>}{health.map(item => <article className="food-result" key={item.occurrence_id}><div><h3>{item.name}</h3><p>{item.status.toLowerCase()} · {item.scheduled_local.date} {item.scheduled_local.time} ({item.scheduled_local.timezone})</p></div></article>)}{healthCursor && <button className="button button-outline button-small" type="button" disabled={busy} onClick={() => void moreHealth()}>Load more</button>}</section></div>
  </>
}

function CustomFoodsPage() {
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
  useEffect(() => { reload().catch(cause => setError(cause instanceof Error ? cause.message : 'Could not load custom foods.')) }, [])
  async function saveFood(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError(''); setMessage('')
    const form = event.currentTarget
    const values = Object.fromEntries(new FormData(form))
    const nutrients: Record<string, { status: string; amount?: string; unit: string }> = {}
    for (const [key, unit] of [['ENERGY', 'kcal'], ['PROTEIN', 'g'], ['CARBOHYDRATE', 'g'], ['FAT', 'g'], ['FIBER', 'g']]) {
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
      form.reset(); setEditing(null); await reload(); setMessage(editing ? 'Your custom food was updated.' : 'Your custom food was saved.')
    } catch (cause) { setError(cause instanceof Error ? cause.message : `Could not ${editing ? 'update' : 'save'} custom food.`) }
    finally { setBusy(false) }
  }
  async function remove(id: string) {
    if (!window.confirm('Delete this custom food? Confirmed food logs keep their historical snapshot.')) return
    setBusy(true); setError('')
    try { await deleteAuthenticatedJson<void>(`/foods/custom/${id}`); await reload() }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not delete food.') }
    finally { setBusy(false) }
  }
  return <><PageHead eyebrow="Your kitchen" title="Custom foods" description="Save private food definitions for reuse. Existing confirmed logs keep their own snapshots." />
    <div className="two-column"><section className="panel"><h2>My foods</h2>{foods.length === 0 && <EmptyState icon={UtensilsCrossed} title="No custom foods yet" text="Add a recipe or product with values you know." />}{foods.map(food => <article className="food-result" key={food.id}><div><h3>{food.name}</h3><p>{food.base_quantity.amount} {food.base_quantity.unit} · {food.food_state.toLowerCase()}</p></div><div className="history-actions"><button className="quiet-link" type="button" disabled={busy} onClick={() => { setEditing(food); setError(''); setMessage('') }}>Edit</button><button className="quiet-link" type="button" disabled={busy} onClick={() => remove(food.id)}>Delete</button></div></article>)}{foodCursor && <button className="button button-outline button-small" type="button" disabled={busy} onClick={() => void moreFoods()}>Load more</button>}</section><section className="panel"><h2>{editing ? `Edit ${editing.name}` : 'Add a custom food'}</h2><form key={editing?.id || 'new'} className="custom-food-form" onSubmit={saveFood}><label>Name<input name="name" required minLength={2} maxLength={120} defaultValue={editing?.name || ''} /></label><label>Description<input name="description" maxLength={500} defaultValue={editing?.description || ''} /></label><label>Food state<select name="food_state" defaultValue={editing?.food_state || 'PREPARED'}><option>PREPARED</option><option>RAW</option><option>COOKED</option><option>UNKNOWN</option></select></label><div className="custom-food-base"><label>Base amount<input name="base_amount" type="number" min="0.001" step="any" defaultValue={editing?.base_quantity.amount || '100'} required /></label><label>Unit<select name="base_unit" defaultValue={editing?.base_quantity.unit || 'g'}><option>g</option><option>ml</option></select></label></div><p className="muted">Enter nutrients for the base amount. Leave unknown values empty.</p>{[['ENERGY', 'kcal'], ['PROTEIN', 'g'], ['CARBOHYDRATE', 'g'], ['FAT', 'g'], ['FIBER', 'g']].map(([key, unit]) => <label key={key}>{key.toLowerCase()} ({unit})<input name={key} type="number" min="0" step="any" defaultValue={editing?.nutrients[key]?.status === 'KNOWN' ? editing.nutrients[key].amount : ''} /></label>)}{error && <p className="form-message form-error" role="alert">{error}</p>}{message && <p className="form-message form-success" role="status">{message}</p>}<div className="history-actions"><button className="button button-primary" type="submit" disabled={busy}>{busy ? 'Saving…' : editing ? 'Save changes' : 'Save custom food'}</button>{editing && <button className="button button-outline" type="button" disabled={busy} onClick={() => { setEditing(null); setError(''); setMessage('') }}>Cancel</button>}</div></form></section></div>
    <SavedMealsManager />
  </>
}

function SavedMealsManager() {
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
  useEffect(() => { load().catch(cause => setError(cause instanceof Error ? cause.message : 'Could not load meals.')) }, [])
  async function rename(meal: SavedMeal) {
    const name = window.prompt('Saved meal name', meal.name)?.trim()
    if (!name || name === meal.name) return
    setBusy(true); setError('')
    try {
      await patchAuthenticatedJson(`/meals/${meal.id}`, { version: meal.version, name, components: meal.components })
      await load()
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not rename meal.') }
    finally { setBusy(false) }
  }
  async function remove(meal: SavedMeal) {
    if (!window.confirm(`Delete saved meal ${meal.name}? Historical food logs remain unchanged.`)) return
    setBusy(true); setError('')
    try { await deleteAuthenticatedJson(`/meals/${meal.id}`); await load() }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not delete meal.') }
    finally { setBusy(false) }
  }
  return <section className="panel section-panel"><div className="panel-top"><h2>Reusable meals</h2><Link className="quiet-link" to="/app/log-food">Build a meal <ArrowRight size={15} /></Link></div>{error && <p className="form-message form-error" role="alert">{error}</p>}{meals.length === 0 && <p className="muted">No saved meals. Build a food group, then save it for reuse.</p>}{meals.map(meal => <article className="food-result" key={meal.id}><div><h3>{meal.name}</h3><p>{meal.components.length} foods</p></div><div className="history-actions"><button className="quiet-link" type="button" disabled={busy} onClick={() => void rename(meal)}>Rename</button><button className="quiet-link" type="button" disabled={busy} onClick={() => void remove(meal)}>Delete</button></div></article>)}{cursor && <button className="button button-outline button-small" type="button" disabled={busy} onClick={() => void load(cursor)}>Load more</button>}</section>
}

type HealthItem = { id: string; version: number; type: string; name: string; dosage: { amount_text: string; unit_text: string }; instructions: string; notes: string; start_date: string; active: boolean; schedule: { timezone: string; recurrence: { frequency: string; interval: number; weekdays: string[]; local_times: string[] }; effective_from_date: string } }
type HealthOccurrence = { occurrence_id: string; name: string; type: string; status: string; scheduled_at: string; scheduled_local: { date: string; time: string; timezone: string } }
function HealthPage() {
  const [items, setItems] = useState<HealthItem[]>([])
  const [occurrences, setOccurrences] = useState<HealthOccurrence[]>([])
  const [timezone, setTimezone] = useState('Asia/Kolkata')
  const [day, setDay] = useState(new Date().toISOString().slice(0, 10))
  const [frequency, setFrequency] = useState('DAILY')
  const [weekdays, setWeekdays] = useState<string[]>([])
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  async function reloadItems() {
    const response = await getAuthenticatedJson<{ items: HealthItem[] }>('/health-items?limit=50')
    setItems(response.items)
  }
  async function reloadOccurrences(selectedDay: string) {
    const from = new Date(`${selectedDay}T00:00:00Z`)
    const to = new Date(from.getTime() + 7 * 24 * 60 * 60 * 1000)
    const params = new URLSearchParams({ from: from.toISOString(), to: to.toISOString(), limit: '100' })
    const response = await getAuthenticatedJson<{ items: HealthOccurrence[] }>(`/health-items/occurrences?${params}`)
    setOccurrences(response.items)
  }
  useEffect(() => {
    getAuthenticatedJson<{ timezone: string }>('/profile').then(profile => setTimezone(profile.timezone)).catch(() => {})
    reloadItems().catch(cause => setError(cause instanceof Error ? cause.message : 'Could not load health items.'))
  }, [])
  useEffect(() => { reloadOccurrences(day).catch(cause => setError(cause instanceof Error ? cause.message : 'Could not load occurrences.')) }, [day])
  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError(''); setMessage('')
    const form = event.currentTarget
    const values = Object.fromEntries(new FormData(form))
    const localTimes = String(values.local_times || '').split(',').map(value => value.trim()).filter(Boolean)
    try {
      await postAuthenticatedJson('/health-items', { type: values.type, name: values.name,
        dosage: { amount_text: values.dosage_amount, unit_text: values.dosage_unit },
        instructions: values.instructions, notes: values.notes, start_date: values.start_date,
        schedule: { timezone, recurrence: { frequency, interval: Number(values.interval),
          weekdays: frequency === 'WEEKLY' ? weekdays : [], local_times: localTimes },
          effective_from_date: values.start_date } })
      form.reset(); setWeekdays([]); setFrequency('DAILY'); setMessage('Health item saved. Times shown below use its schedule timezone.')
      await Promise.all([reloadItems(), reloadOccurrences(day)])
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not save health item.') }
    finally { setBusy(false) }
  }
  async function action(occurrence: HealthOccurrence, status: 'TAKEN' | 'SKIPPED') {
    setBusy(true); setError('')
    try {
      await putAuthenticatedJson(`/health-items/occurrences/${occurrence.occurrence_id}`, { status,
        ...(status === 'TAKEN' ? { completed_at: new Date().toISOString() } : {}), notes: '' })
      await reloadOccurrences(day)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not record action.') }
    finally { setBusy(false) }
  }
  async function deactivate(item: HealthItem) {
    if (!window.confirm(`Deactivate ${item.name}? Past recorded actions will remain visible.`)) return
    setBusy(true); setError('')
    try {
      await postAuthenticatedJson(`/health-items/${item.id}/deactivate`, { version: item.version, end_date: day })
      await Promise.all([reloadItems(), reloadOccurrences(day)])
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not deactivate item.') }
    finally { setBusy(false) }
  }
  async function editItem(item: HealthItem, changeSchedule: boolean) {
    const name = changeSchedule ? item.name : window.prompt('Health item name', item.name)?.trim()
    if (!name) return
    let schedule: HealthItem['schedule'] | undefined
    if (changeSchedule) {
      const localTimes = window.prompt('New local time(s), comma separated (HH:MM)', item.schedule.recurrence.local_times.join(', '))?.split(',').map(value => value.trim()).filter(Boolean)
      if (!localTimes?.length) return
      const effectiveFromDate = window.prompt('New schedule effective from date (YYYY-MM-DD). This must be after the previous schedule start.', new Date(Date.now() + 86400000).toISOString().slice(0, 10))?.trim()
      if (!effectiveFromDate) return
      schedule = { ...item.schedule, recurrence: { ...item.schedule.recurrence, local_times: localTimes }, effective_from_date: effectiveFromDate }
    }
    setBusy(true); setError('')
    try {
      await patchAuthenticatedJson(`/health-items/${item.id}`, { version: item.version, type: item.type, name,
        dosage: item.dosage, instructions: item.instructions, notes: item.notes, start_date: item.start_date,
        ...(schedule ? { schedule } : {}) })
      await Promise.all([reloadItems(), reloadOccurrences(day)])
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not update health item.') }
    finally { setBusy(false) }
  }
  async function reschedule(occurrence: HealthOccurrence) {
    const target = window.prompt('New time as an ISO-8601 UTC instant (for example 2026-10-09T12:30:00Z)', new Date(Date.parse(occurrence.scheduled_at) + 86400000).toISOString())?.trim()
    if (!target) return
    setBusy(true); setError('')
    try {
      await putAuthenticatedJson(`/health-items/occurrences/${occurrence.occurrence_id}`, { status: 'RESCHEDULED', rescheduled_to: target, notes: '' })
      await reloadOccurrences(day)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not reschedule occurrence.') }
    finally { setBusy(false) }
  }
  return <><PageHead eyebrow="Your routine" title="Health items" description="Track medication and supplements in your chosen timezone. This is a record, not medical advice." />
    {error && <p className="form-message form-error" role="alert">{error}</p>}{message && <p className="form-message form-success" role="status">{message}</p>}
    <div className="two-column"><section className="panel"><h2>Your items</h2>{items.length === 0 && <EmptyState icon={HeartPulse} title="No health items yet" text="Create an item with a schedule to see occurrences." />}{items.map(item => <article className="food-result" key={item.id}><div><h3>{item.name}</h3><p>{item.type.toLowerCase()} · {item.dosage.amount_text} {item.dosage.unit_text} · {item.schedule.recurrence.frequency.toLowerCase()} · {item.schedule.timezone}</p><p>{item.active ? 'Active' : 'Inactive'}</p></div>{item.active && <button className="quiet-link" type="button" disabled={busy} onClick={() => deactivate(item)}>Deactivate</button>}</article>)}</section><section className="panel"><h2>Add a health item</h2><form className="custom-food-form health-form" onSubmit={create}><label>Type<select name="type"><option>SUPPLEMENT</option><option>MEDICATION</option></select></label><label>Name<input name="name" minLength={2} maxLength={120} required /></label><div className="custom-food-base"><label>Amount<input name="dosage_amount" maxLength={80} placeholder="1" /></label><label>Unit<input name="dosage_unit" maxLength={80} placeholder="tablet" /></label></div><label>Instructions (your own words)<input name="instructions" maxLength={500} /></label><label>Notes<input name="notes" maxLength={1000} /></label><label>Start date<input name="start_date" type="date" defaultValue={day} required /></label><label>Frequency<select value={frequency} onChange={event => setFrequency(event.target.value)}><option>DAILY</option><option>WEEKLY</option></select></label><label>Interval<input name="interval" type="number" min="1" max="365" defaultValue="1" required /></label>{frequency === 'WEEKLY' && <div className="weekday-list">{['MON','TUE','WED','THU','FRI','SAT','SUN'].map(value => <label key={value}><input type="checkbox" checked={weekdays.includes(value)} onChange={() => setWeekdays(previous => previous.includes(value) ? previous.filter(day => day !== value) : [...previous, value])} />{value}</label>)}</div>}<label>Local time(s), comma separated<input name="local_times" defaultValue="08:00" required placeholder="08:00, 20:00" /></label><p className="muted">Schedule timezone: {timezone}. Change your account timezone in Settings before adding an item if needed.</p><button className="button button-primary" disabled={busy} type="submit">Save item</button></form></section></div>
    <section className="panel health-occurrences"><div className="panel-top"><h2>Scheduled occurrences</h2><label>Week starting <input type="date" value={day} onChange={event => setDay(event.target.value)} /></label></div>{occurrences.length === 0 && <p className="muted">No occurrences in this seven-day window.</p>}{occurrences.map(occurrence => <article className="food-result" key={occurrence.occurrence_id}><div><h3>{occurrence.name}</h3><p>{occurrence.scheduled_local.date} at {occurrence.scheduled_local.time} ({occurrence.scheduled_local.timezone}) · {occurrence.status.toLowerCase()}</p></div><div className="history-actions"><button className="button button-outline button-small" type="button" disabled={busy} onClick={() => action(occurrence, 'TAKEN')}>Taken</button><button className="quiet-link" type="button" disabled={busy} onClick={() => action(occurrence, 'SKIPPED')}>Skipped</button></div></article>)}</section>
    <section className="panel section-panel"><h2>Manage health records</h2><p className="muted">Changing scheduled times creates a new version; earlier recorded actions remain available in Analysis.</p>{items.filter(item => item.active).map(item => <div className="food-result" key={item.id}><strong>{item.name}</strong><div className="history-actions"><button className="quiet-link" type="button" disabled={busy} onClick={() => void editItem(item, false)}>Rename</button><button className="quiet-link" type="button" disabled={busy} onClick={() => void editItem(item, true)}>Change times</button></div></div>)}{occurrences.map(occurrence => <div className="food-result" key={occurrence.occurrence_id}><span>{occurrence.name} · {occurrence.scheduled_local.date} {occurrence.scheduled_local.time}</span><button className="quiet-link" type="button" disabled={busy} onClick={() => void reschedule(occurrence)}>Reschedule this occurrence</button></div>)}</section>
  </>
}

type TargetValue = { reference_value: string; active_value: string; unit: string; is_manual_override: boolean }
type TargetVersion = { id: string; revision: number; status: string; target_basis: string; effective_from_tracking_day: string; source_attribution: { name: string; publisher: string; version: string; information_url: string }; targets: Record<string, TargetValue> }
type TargetSelection = { active: TargetVersion | null; proposed: TargetVersion | null }

function TargetsPage() {
  const [selection, setSelection] = useState<TargetSelection | null>(null)
  const [profile, setProfile] = useState<{ version: number; date_of_birth?: string } | null>(null)
  const [currentDay, setCurrentDay] = useState('')
  const [history, setHistory] = useState<TargetVersion[]>([])
  const [cursor, setCursor] = useState<string | null>(null)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    let active = true
    Promise.all([
      getAuthenticatedJson<TargetSelection>('/nutrition/targets'),
      getAuthenticatedJson<{ version: number; date_of_birth?: string }>('/profile'),
      getAuthenticatedJson<{ tracking_day: string }>('/tracking-day/current'),
    ]).then(([targets, nextProfile, day]) => {
      if (active) { setSelection(targets); setProfile(nextProfile); setCurrentDay(day.tracking_day) }
    }).catch(cause => { if (active) setError(cause instanceof Error ? cause.message : 'Could not load targets.') })
    return () => { active = false }
  }, [])
  async function recalculate() {
    if (!profile || !currentDay) return
    setBusy(true); setError(''); setMessage('')
    try {
      const proposal = await postAuthenticatedJson<TargetVersion>('/nutrition/targets/calculate', { profile_version: profile.version, effective_from_tracking_day: currentDay })
      setSelection(previous => ({ active: previous?.active ?? null, proposed: proposal }))
      setMessage('Review these general benchmarks before choosing any tracking targets.')
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not prepare benchmarks.') }
    finally { setBusy(false) }
  }
  async function accept(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const proposal = selection?.proposed
    if (!proposal) return
    setBusy(true); setError(''); setMessage('')
    try {
      const values = new FormData(event.currentTarget)
      const reason = String(values.get('override_reason') || '').trim()
      const overrides: Record<string, { active_value: string; override_reason: string }> = {}
      for (const [key, target] of Object.entries(proposal.targets)) {
        const amount = String(values.get(`target_${key}`) || '')
        if (amount !== target.reference_value) {
          if (!reason) throw new Error('Add a reason for your adjusted tracking values.')
          overrides[key] = { active_value: amount, override_reason: reason }
        }
      }
      const active = await postAuthenticatedJson<TargetVersion>(`/nutrition/targets/${proposal.id}/accept`, { effective_from_tracking_day: currentDay, overrides })
      setSelection({ active, proposed: null })
      setMessage('Your tracking targets are active. These remain user-selected benchmarks, not personalized medical advice.')
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not accept targets.') }
    finally { setBusy(false) }
  }
  async function reject() {
    const proposal = selection?.proposed
    if (!proposal) return
    setBusy(true); setError(''); setMessage('')
    try {
      await postAuthenticatedJson<void>(`/nutrition/targets/${proposal.id}/reject`, {})
      setSelection(previous => ({ active: previous?.active ?? null, proposed: null }))
      setMessage('Proposal rejected. Your existing active targets, if any, are unchanged.')
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not reject proposal.') }
    finally { setBusy(false) }
  }
  async function loadHistory() {
    setError('')
    try {
      const page = await getAuthenticatedJson<{ items: TargetVersion[]; page: { next_cursor: string | null } }>(`/nutrition/targets/history?limit=10${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`)
      setHistory(previous => [...previous, ...page.items])
      setCursor(page.page.next_cursor)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not load history.') }
  }
  const proposed = selection?.proposed
  const active = selection?.active
  return <><PageHead eyebrow="Your nutrition" title="Nutrition targets" description="Start with transparent FDA food-label benchmarks, then choose the values you want to track." />
    <div className="targets-layout"><section className="panel"><div className="panel-top"><h2>Current tracking targets</h2><Activity size={19} /></div>
      {active ? <><TargetSource version={active} /><TargetRows version={active} /><p className="muted">Active from Tracking Day {active.effective_from_tracking_day}. These values were accepted by you.</p></> : <EmptyState icon={Activity} title="No active targets" text="Nothing is activated automatically. Review and accept a proposal when you are ready." />}
      <button className="button button-outline" type="button" disabled={busy || !profile || !currentDay} onClick={recalculate}>{active ? 'Review a new version' : 'Review FDA benchmarks'}</button>
      {!profile?.date_of_birth && <p className="form-message form-error">Add your date of birth in Settings first. The initial benchmark view is limited to adults.</p>}
    </section>
    <section className="panel"><div className="panel-top"><h2>Review proposal</h2><Info size={19} /></div>
      {proposed ? <><TargetSource version={proposed} /><p className="muted">The FDA values are general food-label comparisons based on a 2,000-kcal reference diet. They do not calculate your personal energy needs. You can adjust values before accepting them as your own tracking targets.</p><form key={proposed.id} className="auth-form" onSubmit={accept}>{Object.entries(proposed.targets).map(([key, value]) => <label key={key}>{nutrientLabel(key)} ({value.unit})<input name={`target_${key}`} type="number" min="0" step="any" required defaultValue={value.active_value} /><small>FDA benchmark: {value.reference_value} {value.unit}</small></label>)}<label>Reason for any adjustments<input name="override_reason" maxLength={200} placeholder="For example, personal tracking preference" /></label><button className="button button-primary" type="submit" disabled={busy}>Accept tracking targets</button><button className="button button-outline" type="button" disabled={busy} onClick={reject}>Reject proposal</button></form></> : <EmptyState icon={Info} title="No proposal to review" text="Preparing benchmarks creates a proposal; it never activates them without your confirmation." />}
    </section></div>
    {error && <p className="form-message form-error" role="alert">{error}</p>}{message && <p className="form-message form-success" role="status">{message}</p>}
    <section className="panel target-history"><div className="panel-top"><h2>Version history</h2><Clock3 size={19} /></div>{history.map(item => <p key={item.id}>Version {item.revision} · {item.status} · {item.effective_from_tracking_day}</p>)}<button className="button button-outline button-small" type="button" onClick={loadHistory}>Load {history.length ? 'more' : 'history'}</button></section>
  </>
}

function nutrientLabel(key: string) { return ({ ENERGY: 'Energy', PROTEIN: 'Protein', CARBOHYDRATE: 'Carbohydrates', FAT: 'Fat', FIBER: 'Fibre' } as Record<string, string>)[key] || key }
function TargetRows({ version }: { version: TargetVersion }) { return <div className="target-rows">{Object.entries(version.targets).map(([key, value]) => <div key={key}><span>{nutrientLabel(key)}</span><strong>{value.active_value} {value.unit}</strong></div>)}</div> }
function TargetSource({ version }: { version: TargetVersion }) { return <details className="target-source"><summary><Info size={16} /> Source: {version.source_attribution.name}</summary><p>{version.source_attribution.publisher} · {version.source_attribution.version}. <a href={version.source_attribution.information_url} target="_blank" rel="noreferrer">View the source</a>. These are {version.target_basis === 'GENERAL_BENCHMARK' ? 'general benchmarks, not personalized recommendations.' : 'source-attributed values.'}</p></details> }

function Settings() {
  const { theme, setTheme } = useTheme()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [profile, setProfile] = useState<{ version: number; display_name: string; timezone: string; date_of_birth?: string } | null>(null)
  const [preferences, setPreferences] = useState<{ version: number; daily_reset_time: string; unit_system: string; dashboard_nutrients: string[] } | null>(null)
  const [trackingDay, setTrackingDay] = useState<{ tracking_day: string; timezone: string; daily_reset_time: string; as_of_utc: string } | null>(null)
  const [saveMessage, setSaveMessage] = useState(params.get('linked') === 'google' ? 'Google sign-in is linked to your account.' : params.get('deletion_cancelled') ? 'Your account deletion request was cancelled and your account was restored.' : '')
  const [saveError, setSaveError] = useState(params.get('error') === 'google_link' ? 'Google account linking could not be completed.' : params.get('error') === 'google_deletion' ? 'Google could not verify the deletion request.' : '')
  const [saving, setSaving] = useState(false)
  useEffect(() => {
    let active = true
    Promise.all([
      getAuthenticatedJson<{ version: number; display_name: string; timezone: string; date_of_birth?: string }>('/profile'),
      getAuthenticatedJson<{ version: number; daily_reset_time: string; unit_system: string; dashboard_nutrients: string[] }>('/preferences'),
      getAuthenticatedJson<{ tracking_day: string; timezone: string; daily_reset_time: string; as_of_utc: string }>('/tracking-day/current'),
    ]).then(([nextProfile, nextPreferences, nextTrackingDay]) => {
      if (active) { setProfile(nextProfile); setPreferences(nextPreferences); setTrackingDay(nextTrackingDay) }
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
      const updated = await patchAuthenticatedJson<typeof profile>('/profile', { version: profile.version, display_name: values.display_name, timezone: values.timezone, date_of_birth: values.date_of_birth })
      setProfile(updated)
      await refreshTrackingDay()
      setSaveMessage('Profile saved. Existing logs keep their original timezone context.')
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
    setSaveError('')
    try {
      const result = await getAuthenticatedJson<{ authorization_url: string }>('/account/identities/google/link/start')
      window.location.assign(result.authorization_url)
    } catch (cause) { setSaveError(cause instanceof Error ? cause.message : 'Could not start Google account linking.') }
  }
  async function requestDeletion(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const password = String(new FormData(event.currentTarget).get('password') || '')
    if (!window.confirm('Request account deletion? You will be signed out immediately. You can cancel within 30 days by verifying your sign-in method.')) return
    setSaving(true); setSaveError(''); setSaveMessage('')
    try {
      const result = await postAuthenticatedJson<{ purge_after: string }>('/account/deletion-request', { password })
      clearAccessToken()
      navigate(`/cancel-deletion?requested=1&purge_after=${encodeURIComponent(result.purge_after)}`, { replace: true })
    } catch (cause) { setSaveError(cause instanceof Error ? cause.message : 'Could not request account deletion.') }
    finally { setSaving(false) }
  }
  async function requestGoogleDeletion() {
    if (!window.confirm('Request account deletion? You will be signed out immediately after Google verifies your identity.')) return
    setSaving(true); setSaveError('')
    try {
      const response = await getAuthenticatedJson<{ authorization_url: string }>('/account/deletion-request/google/start')
      window.location.assign(response.authorization_url)
    } catch (cause) { setSaveError(cause instanceof Error ? cause.message : 'Could not start Google verification.'); setSaving(false) }
  }
  return <>
    <PageHead eyebrow="Your account" title="Profile & settings" description="Choose how AteThat looks and set the timezone used for your account." action={<button className="button button-outline" type="button" onClick={logout}>Log out</button>} />
    <div className="settings-grid">
      <section className="panel"><div className="panel-top"><h2>Appearance</h2><Sun size={19} /></div><p className="muted">Choose a theme or follow your device setting.</p><div className="theme-options">{(['light', 'dark', 'system'] as const).map(option => <button key={option} type="button" className={`theme-option ${theme === option ? 'theme-option-selected' : ''}`} onClick={() => setTheme(option)}><span>{option === 'light' ? <Sun size={20} /> : option === 'dark' ? <Moon size={20} /> : <Settings2 size={20} />}{option[0].toUpperCase() + option.slice(1)}</span>{theme === option && <Check size={18} />}</button>)}</div></section>
      <section className="panel"><div className="panel-top"><h2>Profile & timezone</h2><Clock3 size={19} /></div><p className="muted">Timestamps are stored in UTC and shown using your chosen timezone. Change it manually when you travel.</p>{profile ? <form key={profile.version} className="auth-form" onSubmit={saveProfile}><label>Display name<input name="display_name" required maxLength={80} defaultValue={profile.display_name} /></label><label>Date of birth<input name="date_of_birth" type="date" defaultValue={profile.date_of_birth || ''} /><small>Needed to confirm eligibility for adult nutrition benchmarks.</small></label><label>Default timezone<select name="timezone" defaultValue={profile.timezone}>{Array.from(new Set([profile.timezone, 'Asia/Kolkata', 'UTC', ...Intl.supportedValuesOf('timeZone')])).map(zone => <option key={zone} value={zone}>{zone}</option>)}</select></label><button className="button button-primary" type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save profile'}</button></form> : <p className="muted">Loading profile…</p>}{saveError && <p className="form-message form-error" role="alert">{saveError}</p>}{saveMessage && <p className="form-message form-success" role="status">{saveMessage}</p>}</section>
      <section className="panel"><div className="panel-top"><h2>Daily preferences</h2><CalendarDays size={19} /></div><p className="muted">Your nutrition day starts at the local reset time below. Changing it affects future records only.</p>{preferences ? <form key={preferences.version} className="auth-form" onSubmit={savePreferences}><label>Daily reset time<input name="daily_reset_time" type="time" required defaultValue={preferences.daily_reset_time} /></label><label>Unit system<select name="unit_system" defaultValue={preferences.unit_system}><option value="METRIC">Metric</option><option value="IMPERIAL">Imperial</option></select></label><button className="button button-primary" type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save preferences'}</button></form> : <p className="muted">Loading preferences…</p>}{trackingDay && <p className="tracking-day-note">Current Tracking Day: <strong>{trackingDay.tracking_day}</strong> · {trackingDay.timezone} · resets at {trackingDay.daily_reset_time}</p>}</section>
      {import.meta.env.VITE_GOOGLE_SIGNIN_ENABLED === 'true' && <section className="panel"><div className="panel-top"><h2>Sign-in methods</h2><LockKeyhole size={19} /></div><p className="muted">Link Google explicitly to this account. An email match alone will never merge accounts.</p><button className="button button-outline" type="button" onClick={linkGoogle}><GoogleMark /> Link Google account</button></section>}
      <section className="panel"><div className="panel-top"><h2>Delete account</h2><LockKeyhole size={19} /></div><p className="muted">Requesting deletion signs you out everywhere. Your records are permanently removed after 30 days unless you cancel with fresh verification.</p><form className="auth-form" onSubmit={requestDeletion}><label>Current password<input name="password" type="password" autoComplete="current-password" required /></label><button className="button button-outline" type="submit" disabled={saving}>Request deletion</button></form>{import.meta.env.VITE_GOOGLE_SIGNIN_ENABLED === 'true' && <button className="quiet-link" type="button" disabled={saving} onClick={() => void requestGoogleDeletion()}>Request deletion with Google verification</button>}</section>
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
  return <div className="auth-page"><PublicHeader /><main className="auth-main layout-width"><aside className="auth-intro"><span className="eyebrow"><span className="eyebrow-dot" /> Account deletion</span><h1>Your data is scheduled for removal.</h1><p>Deletion requests have a 30-day cancellation period. Fresh verification is required before access can be restored.</p></aside><section className="auth-card"><div className="auth-card-top"><span className="auth-icon"><LockKeyhole size={22} /></span><Link className="quiet-link" to="/login"><ArrowLeft size={15} /> Back to login</Link></div><h2>{requested ? 'Deletion request received' : 'Cancel account deletion'}</h2>{purgeAfter && <p className="muted">Scheduled permanent removal: {new Date(purgeAfter).toLocaleString()}.</p>}<p className="muted">Use the sign-in method that belongs to the pending account. Cancellation restores the account and signs you in.</p><form className="auth-form" onSubmit={cancel}><label>Email address<input name="email" type="email" autoComplete="email" required /></label><label>Password<input name="password" type="password" autoComplete="current-password" required /></label>{error && <p className="form-message form-error" role="alert">{error}</p>}<button className="button button-primary auth-submit" type="submit" disabled={busy}>{busy ? 'Verifying…' : 'Cancel deletion and restore account'}</button></form>{import.meta.env.VITE_GOOGLE_SIGNIN_ENABLED === 'true' && <><div className="divider"><span>or</span></div><button className="button button-google" type="button" disabled={busy} onClick={() => void cancelWithGoogle()}><GoogleMark /> Verify with Google</button></>}</section></main></div>
}

function NotFound() {
  return <div className="not-found"><Brand /><h1>Page not found</h1><p>That page isn’t part of AteThat.</p><Link className="button button-primary" to="/">Go home <ArrowRight size={16} /></Link></div>
}

export default function App() {
  return <ThemeProvider><DatePickerProvider><Routes>
    <Route path="/" element={<RedirectIfAuthenticated><Landing /></RedirectIfAuthenticated>} />
    <Route path="/terms" element={<TermsPage />} />
    <Route path="/login" element={<RedirectIfAuthenticated><AuthPage mode="login" /></RedirectIfAuthenticated>} />
    <Route path="/signup" element={<RedirectIfAuthenticated><AuthPage mode="signup" /></RedirectIfAuthenticated>} />
    <Route path="/forgot-password" element={<AuthPage mode="forgot" />} />
    <Route path="/resend-verification" element={<AuthPage mode="resend" />} />
    <Route path="/verify-email" element={<TokenPage mode="verify" />} />
    <Route path="/reset-password" element={<TokenPage mode="reset" />} />
    <Route path="/cancel-deletion" element={<CancelDeletionPage />} />
    <Route path="/app" element={<AppShell><Dashboard /></AppShell>} />
    <Route path="/app/log-food" element={<AppShell><LogFood /></AppShell>} />
    <Route path="/app/history" element={<AppShell><HistoryPage /></AppShell>} />
    <Route path="/app/analysis" element={<AppShell><AnalysisPage /></AppShell>} />
    <Route path="/app/meals" element={<AppShell><CustomFoodsPage /></AppShell>} />
    <Route path="/app/health" element={<AppShell><HealthPage /></AppShell>} />
    <Route path="/app/targets" element={<AppShell><TargetsPage /></AppShell>} />
    <Route path="/app/settings" element={<AppShell><Settings /></AppShell>} />
    <Route path="*" element={<NotFound />} />
  </Routes></DatePickerProvider></ThemeProvider>
}
