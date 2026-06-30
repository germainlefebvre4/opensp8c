import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

import enCommon from './locales/en/common.json'
import enNavigation from './locales/en/navigation.json'
import enKanban from './locales/en/kanban.json'
import enDetailPanel from './locales/en/detailPanel.json'
import enWorkspace from './locales/en/workspace.json'
import enSpecs from './locales/en/specs.json'
import enExplore from './locales/en/explore.json'
import enDialogs from './locales/en/dialogs.json'

import frCommon from './locales/fr/common.json'
import frNavigation from './locales/fr/navigation.json'
import frKanban from './locales/fr/kanban.json'
import frDetailPanel from './locales/fr/detailPanel.json'
import frWorkspace from './locales/fr/workspace.json'
import frSpecs from './locales/fr/specs.json'
import frExplore from './locales/fr/explore.json'
import frDialogs from './locales/fr/dialogs.json'

const savedLang = localStorage.getItem('lang') ?? 'en'

i18n.use(initReactI18next).init({
  lng: savedLang,
  fallbackLng: 'en',
  ns: ['common', 'navigation', 'kanban', 'detailPanel', 'workspace', 'specs', 'explore', 'dialogs'],
  defaultNS: 'common',
  resources: {
    en: {
      common: enCommon,
      navigation: enNavigation,
      kanban: enKanban,
      detailPanel: enDetailPanel,
      workspace: enWorkspace,
      specs: enSpecs,
      explore: enExplore,
      dialogs: enDialogs,
    },
    fr: {
      common: frCommon,
      navigation: frNavigation,
      kanban: frKanban,
      detailPanel: frDetailPanel,
      workspace: frWorkspace,
      specs: frSpecs,
      explore: frExplore,
      dialogs: frDialogs,
    },
  },
  interpolation: { escapeValue: false },
})

i18n.on('languageChanged', (lng) => {
  localStorage.setItem('lang', lng)
})

export default i18n
