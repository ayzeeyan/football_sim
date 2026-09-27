/**
 * i18n layer (F13): a typed catalogue with a minimal resolver. English ships
 * first; the layer exists so copy is decoupled from components and future
 * locales have a complete, type-checked contract to fill.
 */
import { en, type MessageKey, type MessageParams } from './en';

export type Locale = 'en';

// Every locale must cover every key of the English catalogue. The record
// type makes an incomplete catalogue a compile error.
const catalogues: Record<Locale, Record<MessageKey, string>> = {
  en: { ...en },
};

let currentLocale: Locale = 'en';

/** Switches the active locale. Unknown locales keep the current one. */
export function setLocale(locale: Locale): void {
  if (locale in catalogues) {
    currentLocale = locale;
  }
}

/** Returns the active locale. */
export function getLocale(): Locale {
  return currentLocale;
}

/**
 * Resolves one message. Interpolates `{param}` placeholders; missing params
 * stay visible as `{param}` so gaps cannot silently render empty.
 */
export function t(key: MessageKey, params?: MessageParams): string {
  const template = catalogues[currentLocale][key] ?? en[key];
  if (!params) return template;
  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in params ? String(params[name]) : match,
  );
}

/** True when every locale covers every key with a non-empty string. */
export function assertCatalogueComplete(): boolean {
  for (const locale of Object.keys(catalogues) as Locale[]) {
    for (const key of Object.keys(en) as MessageKey[]) {
      const value = catalogues[locale][key];
      if (typeof value !== 'string' || value.trim() === '') return false;
    }
  }
  return true;
}

export type { MessageKey, MessageParams };
