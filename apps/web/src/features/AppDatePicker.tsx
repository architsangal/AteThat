import { createContext, useContext, useEffect, useId, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { CalendarDays, ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react'

type DatePickerContextValue = { openPickerId: string | null; setOpenPickerId: (id: string | null) => void }
const DatePickerContext = createContext<DatePickerContextValue | null>(null)
export function DatePickerProvider({ children }: { children: ReactNode }) {
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

type PickerProps = { label: string; value: string; name?: string; onChange: (value: string) => void; className?: string; describedBy?: string; min?: string; max?: string; clearable?: boolean }
export function AppDatePicker({ label, value, name, onChange, className = '', describedBy, min, max, clearable=false }: PickerProps) {
  const context = useContext(DatePickerContext)
  if (!context) throw new Error('AppDatePicker must be rendered inside DatePickerProvider.')
  const pickerId = 'date-' + useId(), labelId = pickerId + '-label'
  const open = context.openPickerId === pickerId
  const root = useRef<HTMLDivElement>(null), trigger = useRef<HTMLButtonElement>(null), calendar = useRef<HTMLDivElement>(null)
  const [position, setPosition] = useState({left:12,top:12})
  const [visibleMonth, setVisibleMonth] = useState(() => parseLocalDate(value || isoLocalDate(new Date())))
  const selected = value ? parseLocalDate(value) : null
  const currentYear = new Date().getFullYear()
  const firstYear = Math.min(currentYear - 120, visibleMonth.getFullYear())
  const lastYear = Math.max(name === 'date_of_birth' ? currentYear : currentYear + 10, visibleMonth.getFullYear())
  const years = Array.from({ length: lastYear - firstYear + 1 }, (_, index) => lastYear - index)
  const days = useMemo(() => {
    const first = new Date(visibleMonth.getFullYear(), visibleMonth.getMonth(), 1)
    const dayCount = new Date(visibleMonth.getFullYear(), visibleMonth.getMonth() + 1, 0).getDate()
    return Array.from({ length:42 }, (_,index) => index < first.getDay() || index >= first.getDay()+dayCount ? null : new Date(first.getFullYear(),first.getMonth(),index-first.getDay()+1))
  }, [visibleMonth])
  const allowed = (day:string) => (!min || day >= min) && (!(max || (name === 'date_of_birth' ? isoLocalDate(new Date()) : '')) || day <= (max || isoLocalDate(new Date())))
  const close = () => { context.setOpenPickerId(null); trigger.current?.focus() }
  useLayoutEffect(() => {
    if (!open) return
    const place = () => {
      const rect = trigger.current?.getBoundingClientRect(), panel = calendar.current
      if (!rect || !panel) return
      const width=panel.offsetWidth, height=panel.offsetHeight, compact=window.innerWidth <= 600
      const left=compact ? (window.innerWidth-width)/2 : Math.min(Math.max(12,rect.left),window.innerWidth-width-12)
      const preferredTop=compact ? (window.innerHeight-height)/2 : rect.bottom+8
      const top=compact ? Math.max(12,preferredTop) : preferredTop+height > window.innerHeight-12 && rect.top-height-8 >= 12 ? rect.top-height-8 : Math.min(preferredTop,Math.max(12,window.innerHeight-height-12))
      setPosition({left,top})
    }
    place()
    window.addEventListener('resize',place)
    window.addEventListener('scroll',place,true)
    return () => { window.removeEventListener('resize',place); window.removeEventListener('scroll',place,true) }
  }, [open,visibleMonth])
  useEffect(() => {
    if (!open) return
    const outside = (event:PointerEvent) => { if (!root.current?.contains(event.target as Node) && !calendar.current?.contains(event.target as Node)) context.setOpenPickerId(null) }
    const escape = (event:KeyboardEvent) => { if (event.key === 'Escape') { event.preventDefault(); close() } }
    document.addEventListener('pointerdown',outside)
    window.addEventListener('keydown',escape)
    return () => { document.removeEventListener('pointerdown',outside); window.removeEventListener('keydown',escape) }
  }, [open,context])
  const choose = (day:string) => { if (allowed(day)) { onChange(day); close() } }
  const displayValue = selected?.toLocaleDateString('en-GB', { day:'2-digit',month:'short',year:'numeric' }) || 'Select date'
  return <div ref={root} className={'app-date-picker '+className}>
    <span id={labelId} className="app-date-label">{label}</span>
    {name && <input type="hidden" name={name} value={value} />}
    <button ref={trigger} className="app-date-trigger" type="button" aria-label={label+': '+displayValue} aria-describedby={describedBy} aria-haspopup="dialog" aria-expanded={open} aria-controls={open ? pickerId : undefined} onClick={() => { setVisibleMonth(selected || new Date()); context.setOpenPickerId(open ? null : pickerId) }}><CalendarDays size={17} aria-hidden="true" /><span>{displayValue}</span><ChevronDown size={16} aria-hidden="true" /></button>
    {open && createPortal(<div ref={calendar} id={pickerId} className="app-calendar app-calendar-shared" role="dialog" aria-label={label+' date picker'} style={position}>
      <div className="app-calendar-head">
        <button type="button" className="calendar-nav" aria-label="Previous month" onClick={() => setVisibleMonth(previous => new Date(previous.getFullYear(),previous.getMonth()-1,1))}><ChevronLeft size={18} /></button>
        <div className="calendar-selectors"><select aria-label={label+' month'} value={visibleMonth.getMonth()} onChange={event => setVisibleMonth(previous => new Date(previous.getFullYear(),Number(event.target.value),1))}>{Array.from({length:12},(_,month) => <option key={month} value={month}>{new Date(2000,month,1).toLocaleDateString('en-GB',{month:'long'})}</option>)}</select><select aria-label={label+' year'} value={visibleMonth.getFullYear()} onChange={event => setVisibleMonth(previous => new Date(Number(event.target.value),previous.getMonth(),1))}>{years.map(year => <option key={year} value={year}>{year}</option>)}</select></div>
        <button type="button" className="calendar-nav" aria-label="Next month" onClick={() => setVisibleMonth(previous => new Date(previous.getFullYear(),previous.getMonth()+1,1))}><ChevronRight size={18} /></button>
      </div>
      <div className="calendar-weekdays" aria-hidden="true">{['Su','Mo','Tu','We','Th','Fr','Sa'].map(day => <span key={day}>{day}</span>)}</div>
      <div className="calendar-days">{days.map((date,index) => date ? <button type="button" key={isoLocalDate(date)} className={'calendar-day '+(isoLocalDate(date) === value ? 'calendar-day-selected ' : '')+(isoLocalDate(date) === isoLocalDate(new Date()) ? 'calendar-day-today' : '')} aria-label={date.toLocaleDateString('en-GB',{day:'numeric',month:'long',year:'numeric'})} aria-pressed={isoLocalDate(date) === value} disabled={!allowed(isoLocalDate(date))} onClick={() => choose(isoLocalDate(date))}>{date.getDate()}</button> : <span key={'blank-'+index} aria-hidden="true" />)}</div>
      <div className="calendar-footer">{clearable&&value&&<button className="quiet-link" type="button" onClick={()=>{onChange('');close()}}>Clear date</button>}<button className="quiet-link" type="button" disabled={!allowed(isoLocalDate(new Date()))} onClick={() => choose(isoLocalDate(new Date()))}>Today</button><button className="quiet-link" type="button" onClick={close}>Close calendar</button></div>
    </div>,root.current?.closest('dialog') || document.body)}
  </div>
}
