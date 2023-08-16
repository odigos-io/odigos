import { METADATA } from '@/utils/constants';
import { Metadata } from 'next';

export const metadata: Metadata = METADATA;

export default function Layout({ children }: { children: React.ReactNode }) {
  return <div>{children}</div>;
}
