import 'styled-components';
import { ITheme } from './styles/theme';

declare module 'styled-components' {
  export interface DefaultTheme extends ITheme {}
}
