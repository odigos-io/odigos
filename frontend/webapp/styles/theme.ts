const baseTheme = {
  colors: {
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
  text: {},
  font_family: {
    primary: 'Inter, sans-serif',
    secondary: 'Kode Mono, sans-serif',
    code: 'IBM Plex Mono, monospace',
  },
};

const darkModeTheme = {
  colors: {
    ...baseTheme.colors,
    // Base Colors
    primary: '#111111',
    secondary: '#F9F9F9',
    border: '#525252',
    dark_grey: '#151515',
    translucent_bg: '#1A1A1A',
    dropdown_bg: '#242424',
    dropdown_bg_2: '#333333',
    // Notification Colors
    warning: '#472300',
    error: '#431919',
    success: '#172013',
    info: '#242424',
    default: '#181944',
  },
  text: {
    // Base Colors
    primary: '#111111',
    secondary: '#F9F9F9',
    white: '#FFFFFF',
    grey: '#B8B8B8',
    dark_grey: '#8F8F8F',
    darker_grey: '#7A7A7A',
    light_grey: '#CCD0D2',
    dark_button: '#0A1824',
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
  font_family: baseTheme.font_family,
};

const lightModeTheme = {
  colors: {
    ...baseTheme.colors,
    // Base Colors
    primary: '#EEEEEE',
    secondary: '#060606',
    border: '#ADADAD',
    dark_grey: '#EAEAEA',
    translucent_bg: '#E5E5E5',
    dropdown_bg: '#DBDBDB',
    dropdown_bg_2: '#CCCCCC',
    // Notification Colors
    warning: '#F6F092',
    error: '#FACECE',
    success: '#C8DEB8',
    info: '#E0E0E0',
    default: '#CAD9FB',
  },
  text: {
    // Base Colors
    primary: '#EEEEEE',
    secondary: '#060606',
    white: '#000000',
    grey: '#474747',
    dark_grey: '#707070',
    darker_grey: '#858585',
    light_grey: '#332F2D',
    dark_button: '#F5E7DB',
    // Notification Colors
    warning: '#A56F12',
    warning_secondary: '#CA9416',
    error: '#B63A3A',
    error_secondary: '#DB5151',
    success: '#4E763A',
    success_secondary: '#67964C',
    info: '#525252',
    info_secondary: '#7A7A7A',
    default: '#444AD9',
    default_secondary: '#6C7AE8',
  },
  font_family: baseTheme.font_family,
};

export type ITheme = typeof darkModeTheme & typeof lightModeTheme;
export const getTheme = (darkMode: boolean): ITheme => (darkMode ? darkModeTheme : lightModeTheme);
