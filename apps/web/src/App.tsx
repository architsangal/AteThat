import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { Link, NavLink, Route, Routes, useNavigate, useSearchParams } from 'react-router-dom'
import {
  Activity, ArrowLeft, ArrowRight, CalendarDays, Check,
  CircleHelp, Clock3, HeartPulse, Home, Info, LockKeyhole, Menu,
  Moon, Plus, Search, Settings2, ShieldCheck, Sun, UtensilsCrossed, X,
} from 'lucide-react'
import { apiBaseUrl, getAuthenticatedJson, patchAuthenticatedJson, postJson } from './api'
import { clearAccessToken, getAccessToken, setAccessToken } from './session'
import { ThemeProvider, useTheme } from './theme'

type Icon = typeof Home

const navigation: { label: string; to: string; icon: Icon }[] = [
  { label: 'Dashboard', to: '/app', icon: Home },
  { label: 'Log food', to: '/app/log-food', icon: Plus },
  { label: 'History', to: '/app/history', icon: CalendarDays },
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
    </main><footer className="public-footer layout-width"><Brand /><span>© AteThat</span></footer>
  </div>
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
  useEffect(() => {
    if (getAccessToken()) return
    let active = true
    postJson<SessionResponse>('/auth/refresh', {}).then(session => {
      if (active) { setAccessToken(session.access_token); setAuthenticated(true) }
    }).catch(() => { if (active) navigate('/login', { replace: true }) })
    return () => { active = false }
  }, [navigate])
  if (!authenticated) return <div className="not-found"><Brand /><p>Checking your session…</p></div>
  return <div className="app-shell"><aside className={`sidebar ${menuOpen ? 'sidebar-open' : ''}`}><div className="sidebar-head"><Brand /><button className="icon-button mobile-only" type="button" onClick={() => setMenuOpen(false)} aria-label="Close menu"><X size={20} /></button></div><nav className="sidebar-nav" aria-label="Main navigation">{navigation.map(({ label, to, icon: IconComponent }) => <NavLink onClick={() => setMenuOpen(false)} key={to} to={to} end={to === '/app'} className={({ isActive }) => `nav-item ${isActive ? 'nav-item-active' : ''}`}><IconComponent size={19} /><span>{label}</span></NavLink>)}</nav><div className="sidebar-foot"><span className="sidebar-foot-icon"><ShieldCheck size={18} /></span><p>Know what nourishes you.<small>Your daily view, at your pace.</small></p></div></aside>
    {menuOpen && <button className="mobile-scrim" type="button" aria-label="Close menu" onClick={() => setMenuOpen(false)} />}
    <div className="app-main"><header className="app-header"><div className="app-header-left"><button className="icon-button mobile-only" type="button" aria-label="Open menu" onClick={() => setMenuOpen(true)}><Menu size={21} /></button><span className="app-header-title">AteThat</span></div><div className="app-header-right"><span className="header-timezone"><Clock3 size={15} /> Time shown in your account timezone</span><ThemeButton /><Link className="button button-primary button-small header-log" to="/app/log-food"><Plus size={17} /> Log food</Link></div></header><main className="content">{children}</main></div>
    <nav className="bottom-nav" aria-label="Mobile navigation">{navigation.slice(0, 4).map(({ label, to, icon: IconComponent }) => <NavLink key={to} to={to} end={to === '/app'} className={({ isActive }) => isActive ? 'bottom-item bottom-item-active' : 'bottom-item'}><IconComponent size={20} /><span>{label}</span></NavLink>)}<button type="button" className="bottom-item" onClick={() => setMenuOpen(true)}><Menu size={20} /><span>More</span></button></nav>
  </div>
}

function PageHead({ eyebrow, title, description, action }: { eyebrow?: string; title: string; description: string; action?: ReactNode }) {
  return <div className="page-head"><div>{eyebrow && <span className="eyebrow">{eyebrow}</span>}<h1>{title}</h1><p>{description}</p></div>{action}</div>
}

function EmptyState({ icon: IconComponent, title, text, action }: { icon: Icon; title: string; text: string; action?: ReactNode }) {
  return <div className="empty-state"><span className="empty-icon"><IconComponent size={26} /></span><h2>{title}</h2><p>{text}</p>{action}</div>
}

function Dashboard() {
  return <><PageHead eyebrow="Your daily view" title="Today at a glance" description="Energy first, followed by the routine nutrients and records you choose to track." action={<Link className="button button-primary" to="/app/log-food"><Plus size={17} /> Log food</Link>} />
    <div className="dashboard-grid"><section className="panel energy-panel"><div className="panel-top"><span className="panel-kicker">Energy</span><span className="info-pill"><Info size={14} /> Account timezone</span></div><h2>Start with your first food log.</h2><p>Once you log a meal, your energy total and target progress will appear here.</p><div className="empty-meter"><span /></div><Link className="quiet-link" to="/app/log-food">Log food <ArrowRight size={16} /></Link></section>
      <section className="panel nutrient-panel"><div className="panel-top"><h2>Routine nutrients</h2><CircleHelp size={18} aria-hidden="true" /></div><NutrientPlaceholder label="Protein" color="protein" /><NutrientPlaceholder label="Carbohydrates" color="carbs" /><NutrientPlaceholder label="Fat" color="fat" /><NutrientPlaceholder label="Fibre" color="fibre" /><Link className="quiet-link" to="/app/targets">View nutrition targets <ArrowRight size={16} /></Link></section></div>
    <div className="dashboard-grid dashboard-secondary"><section className="panel"><div className="panel-top"><h2>Today’s food</h2><Link className="quiet-link" to="/app/history">View history <ArrowRight size={15} /></Link></div><EmptyState icon={UtensilsCrossed} title="No food logged yet" text="You can add several foods together and review their totals before confirming." action={<Link className="button button-outline button-small" to="/app/log-food">Log your first meal</Link>} /></section><section className="panel"><div className="panel-top"><h2>Health routine</h2><Link className="quiet-link" to="/app/health">View items <ArrowRight size={15} /></Link></div><EmptyState icon={HeartPulse} title="Your routine starts here" text="Add medication or supplements you want to keep track of. Your schedule stays private." action={<Link className="button button-outline button-small" to="/app/health">View health items</Link>} /></section></div>
  </>
}

function NutrientPlaceholder({ label, color }: { label: string; color: string }) {
  return <div className="nutrient-row"><div><span className={`nutrient-dot ${color}`} />{label}</div><span>—</span><div className="nutrient-track"><span /></div></div>
}

function LogFood() {
  return <><PageHead eyebrow="Food logging" title="Build your food log" description="Search USDA foods or use your own saved foods, then review one or more items together before confirming." />
    <div className="two-column"><section className="panel"><label className="search-field"><Search size={19} /><input type="search" placeholder="Search foods" aria-label="Search foods" disabled /><span className="search-hint">Search available with the API</span></label><div className="filter-row"><span className="filter-chip filter-chip-active">USDA foods</span><span className="filter-chip">My foods</span><span className="filter-chip">Saved meals</span></div><EmptyState icon={Search} title="Food search will appear here" text="Results will show source, serving size, and known nutrients before you add an item." /></section><aside className="panel tray-panel"><div className="panel-top"><h2>Your food group</h2><span className="info-pill">0 items</span></div><EmptyState icon={UtensilsCrossed} title="Your group is empty" text="Add foods, adjust quantities, and review the combined nutrition before saving." /><button className="button button-primary full-width" type="button" disabled>Review food group</button></aside></div>
  </>
}

function SectionPage({ kind }: { kind: 'history' | 'meals' | 'health' | 'targets' }) {
  const content = {
    history: { eyebrow: 'Your records', title: 'Food history', description: 'Browse confirmed food groups and daily nutrition summaries.', icon: CalendarDays, empty: 'No food history yet', text: 'Confirmed meals will appear here. Older compacted days will show daily totals.' },
    meals: { eyebrow: 'Your kitchen', title: 'Custom foods & saved meals', description: 'Keep reusable foods and meal combinations private to your account.', icon: UtensilsCrossed, empty: 'No saved foods or meals yet', text: 'Create your own foods and meals when the API is ready.' },
    health: { eyebrow: 'Your routine', title: 'Health items', description: 'Track medication and supplements with schedules in your chosen timezone.', icon: HeartPulse, empty: 'No health items yet', text: 'Items and schedule occurrences will appear here after you add them.' },
    targets: { eyebrow: 'Your nutrition', title: 'Nutrition targets', description: 'Review energy and nutrient targets, with source information for each recommendation.', icon: Activity, empty: 'No targets set yet', text: 'Complete your profile to calculate and review your first target proposal.' },
  }[kind]
  return <><PageHead eyebrow={content.eyebrow} title={content.title} description={content.description} /><section className="panel section-panel"><EmptyState icon={content.icon} title={content.empty} text={content.text} /></section></>
}

function Settings() {
  const { theme, setTheme } = useTheme()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [profile, setProfile] = useState<{ version: number; display_name: string; timezone: string } | null>(null)
  const [saveMessage, setSaveMessage] = useState(params.get('linked') === 'google' ? 'Google sign-in is linked to your account.' : '')
  const [saveError, setSaveError] = useState(params.get('error') === 'google_link' ? 'Google account linking could not be completed.' : '')
  const [saving, setSaving] = useState(false)
  useEffect(() => {
    let active = true
    getAuthenticatedJson<{ version: number; display_name: string; timezone: string }>('/profile')
      .then(value => { if (active) setProfile(value) })
      .catch(cause => { if (active) setSaveError(cause instanceof Error ? cause.message : 'Could not load your profile.') })
    return () => { active = false }
  }, [])
  async function saveProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!profile) return
    setSaving(true); setSaveError(''); setSaveMessage('')
    try {
      const values = Object.fromEntries(new FormData(event.currentTarget))
      const updated = await patchAuthenticatedJson<typeof profile>('/profile', { version: profile.version, display_name: values.display_name, timezone: values.timezone })
      setProfile(updated)
      setSaveMessage('Profile saved. Existing logs keep their original timezone context.')
    } catch (cause) { setSaveError(cause instanceof Error ? cause.message : 'Could not save your profile.') }
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
  return <>
    <PageHead eyebrow="Your account" title="Profile & settings" description="Choose how AteThat looks and set the timezone used for your account." action={<button className="button button-outline" type="button" onClick={logout}>Log out</button>} />
    <div className="settings-grid">
      <section className="panel"><div className="panel-top"><h2>Appearance</h2><Sun size={19} /></div><p className="muted">Choose a theme or follow your device setting.</p><div className="theme-options">{(['light', 'dark', 'system'] as const).map(option => <button key={option} type="button" className={`theme-option ${theme === option ? 'theme-option-selected' : ''}`} onClick={() => setTheme(option)}><span>{option === 'light' ? <Sun size={20} /> : option === 'dark' ? <Moon size={20} /> : <Settings2 size={20} />}{option[0].toUpperCase() + option.slice(1)}</span>{theme === option && <Check size={18} />}</button>)}</div></section>
      <section className="panel"><div className="panel-top"><h2>Profile & timezone</h2><Clock3 size={19} /></div><p className="muted">Timestamps are stored in UTC and shown using your chosen timezone. Change it manually when you travel.</p>{profile ? <form className="auth-form" onSubmit={saveProfile}><label>Display name<input name="display_name" required maxLength={80} defaultValue={profile.display_name} /></label><label>Default timezone<select name="timezone" defaultValue={profile.timezone}>{Array.from(new Set([profile.timezone, 'Asia/Kolkata', 'UTC', ...Intl.supportedValuesOf('timeZone')])).map(zone => <option key={zone} value={zone}>{zone}</option>)}</select></label><button className="button button-primary" type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save profile'}</button></form> : <p className="muted">Loading profile…</p>}{saveError && <p className="form-message form-error" role="alert">{saveError}</p>}{saveMessage && <p className="form-message form-success" role="status">{saveMessage}</p>}</section>
      {import.meta.env.VITE_GOOGLE_SIGNIN_ENABLED === 'true' && <section className="panel"><div className="panel-top"><h2>Sign-in methods</h2><LockKeyhole size={19} /></div><p className="muted">Link Google explicitly to this account. An email match alone will never merge accounts.</p><button className="button button-outline" type="button" onClick={linkGoogle}><GoogleMark /> Link Google account</button></section>}
    </div>
  </>
}

function NotFound() {
  return <div className="not-found"><Brand /><h1>Page not found</h1><p>That page isn’t part of AteThat.</p><Link className="button button-primary" to="/">Go home <ArrowRight size={16} /></Link></div>
}

export default function App() {
  return <ThemeProvider><Routes>
    <Route path="/" element={<Landing />} />
    <Route path="/login" element={<AuthPage mode="login" />} />
    <Route path="/signup" element={<AuthPage mode="signup" />} />
    <Route path="/forgot-password" element={<AuthPage mode="forgot" />} />
    <Route path="/resend-verification" element={<AuthPage mode="resend" />} />
    <Route path="/verify-email" element={<TokenPage mode="verify" />} />
    <Route path="/reset-password" element={<TokenPage mode="reset" />} />
    <Route path="/app" element={<AppShell><Dashboard /></AppShell>} />
    <Route path="/app/log-food" element={<AppShell><LogFood /></AppShell>} />
    <Route path="/app/history" element={<AppShell><SectionPage kind="history" /></AppShell>} />
    <Route path="/app/meals" element={<AppShell><SectionPage kind="meals" /></AppShell>} />
    <Route path="/app/health" element={<AppShell><SectionPage kind="health" /></AppShell>} />
    <Route path="/app/targets" element={<AppShell><SectionPage kind="targets" /></AppShell>} />
    <Route path="/app/settings" element={<AppShell><Settings /></AppShell>} />
    <Route path="*" element={<NotFound />} />
  </Routes></ThemeProvider>
}
