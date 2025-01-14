import { hexPercentValues } from './opacity';

const baseTheme = {
  colors: {
    // Notification Colors
    warning: '#472300',
    error: '#431919',
    success: '#172013',
    info: '#242424',
    default: '#181944',
    // Custom Colors
    majestic_blue: '#444AD9',
    orange_og: '#FE9239',
    orange_soft: '#FFB160',
    dark_red: '#802828',
    darker_red: '#611F1F',
    darkest_red: '#281515',
    darkest_red_hover: '#351515',
    dark_green: '#2D4323',
  },
  text: {
    // Notification Colors
    warning: '#E9CF35',
    warning_secondary: '#FFA349',
    error: '#EF7676',
    error_secondary: '#DB5151',
    success: '#81AF65',
    success_secondary: '#51DB51',
    info: '#B8B8B8',
    info_secondary: '#CCDDDD',
    default: '#AABEF7',
    default_secondary: '#8CBEFF',
  },
  font_family: {
    primary: 'Inter, sans-serif',
    secondary: 'Kode Mono, sans-serif',
    code: 'IBM Plex Mono, monospace',
  },
};

const darkModeTheme = {
  colors: {
    ...baseTheme.colors,

    primary: '#111111',
    secondary: '#F9F9F9',
    dark_grey: '#151515',
    text: '#F9F9F9',
    border: '#525252',
    translucent_bg: '#1A1A1A',
    card: '#F9F9F9' + hexPercentValues['004'],
    dropdown_bg: '#242424',
    dropdown_bg_2: '#333333',
    blank_background: '#111111' + hexPercentValues['000'],
    white_opacity: {
      '002': 'rgba(249, 249, 249, 0.02)',
      '004': 'rgba(249, 249, 249, 0.04)',
      '006': 'rgba(249, 249, 249, 0.06)',
      '008': 'rgba(249, 249, 249, 0.08)',
      '10': 'rgba(255, 255, 255, 0.1)',
      '20': 'rgba(255, 255, 255, 0.2)',
      '30': 'rgba(255, 255, 255, 0.3)',
      '40': 'rgba(255, 255, 255, 0.4)',
      '50': 'rgba(255, 255, 255, 0.5)',
      '60': 'rgba(255, 255, 255, 0.6)',
      '70': 'rgba(255, 255, 255, 0.7)',
      '80': 'rgba(255, 255, 255, 0.8)',
      '90': 'rgba(255, 255, 255, 0.9)',
    },
  },
  text: {
    ...baseTheme.text,

    primary: '#111111',
    secondary: '#F9F9F9',
    white: '#FFFFFF',
    grey: '#B8B8B8',
    dark_grey: '#8F8F8F',
    darker_grey: '#7A7A7A',
    light_grey: '#CCD0D2',
    dark_button: '#0A1824',
  },
  font_family: baseTheme.font_family,
};

const lightModeTheme = {
  colors: {
    ...baseTheme.colors,

    primary: '#EEEEEE',
    secondary: '#060606',
    dark_grey: '#EAEAEA',
    text: '#060606',
    border: '#ADADAD',
    translucent_bg: '#E5E5E5',
    card: '#060606' + hexPercentValues['004'],
    dropdown_bg: '#DBDBDB',
    dropdown_bg_2: '#CCCCCC',
    blank_background: '#EEEEEE' + hexPercentValues['000'],
    white_opacity: {
      '002': 'rgba(6, 6, 6, 0.98)',
      '004': 'rgba(6, 6, 6, 0.96)',
      '006': 'rgba(6, 6, 6, 0.94)',
      '008': 'rgba(6, 6, 6, 0.92)',
      '10': 'rgba(0, 0, 0, 0.9)',
      '20': 'rgba(0, 0, 0, 0.8)',
      '30': 'rgba(0, 0, 0, 0.7)',
      '40': 'rgba(0, 0, 0, 0.6)',
      '50': 'rgba(0, 0, 0, 0.5)',
      '60': 'rgba(0, 0, 0, 0.4)',
      '70': 'rgba(0, 0, 0, 0.3)',
      '80': 'rgba(0, 0, 0, 0.2)',
      '90': 'rgba(0, 0, 0, 0.1)',
    },
  },
  text: {
    ...baseTheme.text,

    primary: '#EEEEEE',
    secondary: '#060606',
    white: '#000000',
    grey: '#474747',
    dark_grey: '#707070',
    darker_grey: '#858585',
    light_grey: '#332F2D',
    dark_button: '#F5E7DB',
  },
  font_family: baseTheme.font_family,
};

export type ITheme = typeof darkModeTheme & typeof lightModeTheme;
export const getTheme = (darkMode: boolean): ITheme => (darkMode ? darkModeTheme : lightModeTheme);
