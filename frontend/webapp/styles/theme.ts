import { DefaultTheme } from 'styled-components';

// Define your color palette
const colors = {
  primary: '#111',
  secondary: '#F9F9F9',
  dark_grey: '#151515',
  text: '#F9F9F9',
  border: '#525252',
  translucent_bg: '#1A1A1A',
  majestic_blue: '#444AD9',
  selected_hover: '#444AD93D',
  card: '#F9F9F90A',
  dropdown_bg: '#242424',
  blank_background: '#11111100',
  danger: '#EF7676',
  warning: '#E9CF35',
  white_opacity: {
    '004': 'rgba(249, 249, 249, 0.04)',
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
  gray: {
    '50': '#F7F7F8',
    '100': '#EBEBEF',
    '200': '#D1D1D8',
    '300': '#A9A9BC',
    '400': '#8A8AA3',
    '500': '#6C6C89',
    '600': '#55555D',
    '700': '#3F3F50',
    '800': '#282833',
    '900': '#121217',
    '950': '#121217',
  },
  blue: {
    '50': '#F4F1FD',
    '100': '#E2DAFB',
    '200': '#C6B6F7',
    '300': '#A99F3F3',
    '400': '#80C8EF',
    '500': '#7047EB',
    '600': '#543E7',
    '700': '#4316CA',
    '800': '#3712A5',
    '900': '#280E81',
    '950': '#280E81',
  },
  green: {
    '50': '#EEFBF4',
    '100': '#DFEBEA',
    '200': '#B2EECC',
    '300': '#84E4AE',
    '400': '#5BD990',
    '500': '#2DCA72',
    '600': '#26A95F',
    '700': '#4DDBAC',
    '800': '#17663A',
    '900': '#0F4527',
    '950': '#0F4527',
  },
  red: {
    '50': '#FEF0F4',
    '100': '#FDD8E1',
    '200': '#FBB1C4',
    '300': '#F98BA6',
    '400': '#F76489',
    '500': '#F53D6B',
    '600': '#F3164E',
    '700': '#D5083E',
    '800': '#AF0932',
    '900': '#880727',
    '950': '#880727',
  },
  orange: {
    '50': '#FFF2EE',
    '100': '#FFEBE1',
    '200': '#FFDCBD',
    '300': '#FFB399',
    '400': '#FF9876',
    '500': '#FF7D52',
    '600': '#FF571F',
    '700': '#EB3A00',
    '800': '#B82200',
    '900': '#852100',
    '950': '#852100',
  },
  yellow: {
    '50': '#FFF9EB',
    '100': '#FFF3D6',
    '200': '#FFEFAD',
    '300': '#FFDAB5',
    '400': '#FFEC5C',
    '500': '#FFC233',
    '600': '#AFAF00',
    '700': '#C28800',
    '800': '#8A6100',
    '900': '#523900',
    '950': '#523900',
  },
};

const text = {
  primary: '#111',
  secondary: '#F9F9F9',
  white: '#fff',
  grey: '#B8B8B8',
  dark_grey: '#7A7A7A',
  light_grey: '#CCD0D2',
  dark_button: '#0A1824',
  success: '#81AF65',
  error: '#EF7676',
};

const font_family = {
  primary: 'Inter, sans-serif',
  secondary: 'Kode Mono, sans-serif',
};

// Define the theme interface
interface ThemeInterface extends DefaultTheme {
  colors: typeof colors;
  text: typeof text;
  font_family: typeof font_family;
}

// Create your theme object
const theme: ThemeInterface = {
  colors,
  text,
  font_family,
};

// Export the theme
export default theme;
