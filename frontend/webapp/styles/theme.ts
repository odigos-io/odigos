import { DefaultTheme } from 'styled-components';

<<<<<<< HEAD
// Define your color palette
const colors = {
  primary: '#111',
=======
export const hexPercentValues = {
  '100': 'FF',
  '099': 'FC',
  '098': 'FA',
  '097': 'F7',
  '096': 'F5',
  '095': 'F2',
  '094': 'F0',
  '093': 'ED',
  '092': 'EB',
  '091': 'E8',
  '090': 'E6',
  '089': 'E3',
  '088': 'E0',
  '087': 'DE',
  '086': 'DB',
  '085': 'D9',
  '084': 'D6',
  '083': 'D4',
  '082': 'D1',
  '081': 'CF',
  '080': 'CC',
  '079': 'C9',
  '078': 'C7',
  '077': 'C4',
  '076': 'C2',
  '075': 'BF',
  '074': 'BD',
  '073': 'BA',
  '072': 'B8',
  '071': 'B5',
  '070': 'B3',
  '069': 'B0',
  '068': 'AD',
  '067': 'AB',
  '066': 'A8',
  '065': 'A6',
  '064': 'A3',
  '063': 'A1',
  '062': '9E',
  '061': '9C',
  '060': '99',
  '059': '96',
  '058': '94',
  '057': '91',
  '056': '8F',
  '055': '8C',
  '054': '8A',
  '053': '87',
  '052': '85',
  '051': '82',
  '050': '80',
  '049': '7D',
  '048': '7A',
  '047': '78',
  '046': '75',
  '045': '73',
  '044': '70',
  '043': '6E',
  '042': '6B',
  '041': '69',
  '040': '66',
  '039': '63',
  '038': '61',
  '037': '5E',
  '036': '5C',
  '035': '59',
  '034': '57',
  '033': '54',
  '032': '52',
  '031': '4F',
  '030': '4D',
  '029': '4A',
  '028': '47',
  '027': '45',
  '026': '42',
  '025': '40',
  '024': '3D',
  '023': '3B',
  '022': '38',
  '021': '36',
  '020': '33',
  '019': '30',
  '018': '2E',
  '017': '2B',
  '016': '29',
  '015': '26',
  '014': '24',
  '013': '21',
  '012': '1F',
  '011': '1C',
  '010': '1A',
  '009': '17',
  '008': '14',
  '007': '12',
  '006': '0F',
  '005': '0D',
  '004': '0A',
  '003': '08',
  '002': '05',
  '001': '03',
  '000': '00',
};

// Define your color palette
const colors = {
  primary: '#111111',
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  secondary: '#F9F9F9',
  dark_grey: '#151515',
  text: '#F9F9F9',
  border: '#525252',
  translucent_bg: '#1A1A1A',
  majestic_blue: '#444AD9',
<<<<<<< HEAD
  card: '#F9F9F90A',
  dropdown_bg: '#242424',
  blank_background: '#11111100',
=======
  card: '#F9F9F9' + hexPercentValues['004'],
  dropdown_bg: '#242424',
  dropdown_bg_2: '#333333',
  blank_background: '#111111' + hexPercentValues['000'],
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866

  dark_red: '#802828',
  darker_red: '#611F1F',
  dark_green: '#2D4323',

  warning: '#472300',
  error: '#431919',
  success: '#172013',
  info: '#242424',
  default: '#181944',

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
<<<<<<< HEAD
  primary: '#111',
  secondary: '#F9F9F9',
  white: '#fff',
=======
  primary: '#111111',
  secondary: '#F9F9F9',
  white: '#FFFFFF',
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  grey: '#B8B8B8',
  dark_grey: '#7A7A7A',
  light_grey: '#CCD0D2',
  dark_button: '#0A1824',

  warning: '#E9CF35',
  error: '#EF7676',
  success: '#81AF65',
  info: '#B8B8B8',
  default: '#AABEF7',
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
