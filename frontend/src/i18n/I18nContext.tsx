import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { en, type MessageKey } from './bundles/en'
import { zh } from './bundles/zh'

export type Lang = 'en' | 'zh'

const STORAGE_KEY = 'commitlens-lang'

const byLang: Record<Lang, typeof en> = { en, zh: zh as unknown as typeof en }

function detectLang(): Lang {
  try {
    const s = localStorage.getItem(STORAGE_KEY)
    if (s === 'en' || s === 'zh') return s
  } catch {
    /* ignore */
  }
  if (typeof navigator !== 'undefined' && navigator.language.toLowerCase().startsWith('zh')) {
    return 'zh'
  }
  return 'en'
}

type Ctx = {
  lang: Lang
  setLang: (l: Lang) => void
  t: (key: MessageKey) => string
  tf: (key: MessageKey, vars: Record<string, string | number>) => string
}

const I18nContext = createContext<Ctx | null>(null)

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(() => detectLang())

  const setLang = useCallback((l: Lang) => {
    setLangState(l)
    try {
      localStorage.setItem(STORAGE_KEY, l)
    } catch {
      /* ignore */
    }
  }, [])

  const t = useCallback(
    (key: MessageKey) => {
      const s = byLang[lang][key] ?? byLang.en[key] ?? (key as string)
      return s
    },
    [lang],
  )

  const tf = useCallback(
    (key: MessageKey, vars: Record<string, string | number>) => {
      let s: string = t(key) as string
      for (const [k, v] of Object.entries(vars)) {
        s = s.replaceAll(`{${k}}`, String(v))
      }
      return s
    },
    [t],
  )

  const value = useMemo(() => ({ lang, setLang, t, tf }), [lang, setLang, t, tf])
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

/** Hook used only under I18nProvider; colocated for simple bundling. */
// eslint-disable-next-line react-refresh/only-export-components
export function useI18n(): Ctx {
  const c = useContext(I18nContext)
  if (!c) throw new Error('useI18n must be used within I18nProvider')
  return c
}
