import { useContext } from "react";
import { I18nContext } from "../App";
import en from "./en.json";
import es from "./es.json";
import pt from "./pt.json";
import fr from "./fr.json";

export const translations = { en, es, pt, fr };

export function getTranslation(lang, key) {
  const langDict = translations[lang] || translations.en;
  if (langDict && langDict[key]) {
    return langDict[key];
  }
  if (translations.en && translations.en[key]) {
    return translations.en[key];
  }
  return key;
}

export function useTranslation() {
  const context = useContext(I18nContext);
  const lang = context?.lang || "en";
  const t = (key) => getTranslation(lang, key);
  return { t, lang, setLang: context?.setLang };
}
