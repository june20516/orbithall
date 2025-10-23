import { createContext } from "preact";
import { useContext, useMemo } from "preact/hooks";
import { locales, type Locale, type TranslationKey } from "./locales";

interface I18nContextValue {
  locale: Locale;
  t: (key: TranslationKey) => string;
}

const I18nContext = createContext<I18nContextValue | null>(null);

interface I18nProviderProps {
  locale: Locale;
  children: preact.ComponentChildren;
}

export function I18nProvider({ locale, children }: I18nProviderProps) {
  const value = useMemo(() => {
    const t = (key: TranslationKey): string => {
      return locales[locale][key] || locales.ko[key] || key;
    };

    return { locale, t };
  }, [locale]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("useI18n must be used within I18nProvider");
  }
  return context;
}
